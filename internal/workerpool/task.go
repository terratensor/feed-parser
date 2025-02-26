package workerpool

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/crawler"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/lib/logger/sl"
	"github.com/terratensor/feed-parser/internal/metrics"
	"github.com/terratensor/feed-parser/internal/splitter"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
)

/**
Task содержит все необходимое для обработки задачи.
Мы передаем ей Data и функцию f, которая должна быть выполнена, с помощью функции process.
Функция f принимает Data в качестве параметра для обработки, а также храним возвращаемую ошибку
*/

type Task struct {
	Err error
	//Entries *feed.Entries
	Data           *feed.Entry
	f              func(interface{}) error
	Splitter       splitter.Splitter
	EntriesStorage *feed.Entries
	Config         *config.Config
	metrics        *metrics.Metrics
}

func NewTaskStorage() *feed.Entries {
	var storage feed.StorageInterface

	manticoreClient, err := manticore.New("feed")
	if err != nil {
		log.Printf("failed to initialize manticore client, %v", err)
		os.Exit(1)
	}

	storage = manticoreClient

	return feed.NewFeedStorage(storage)
}

func NewTask(f func(interface{}) error, data feed.Entry, splitter *splitter.Splitter, storage *feed.Entries, cfg *config.Config, metrics *metrics.Metrics) *Task {
	return &Task{
		f:              f,
		Data:           &data,
		Splitter:       *splitter,
		EntriesStorage: storage,
		Config:         cfg,
		metrics:        metrics,
	}
}

func process(workerID int, task *Task) {
	fmt.Printf("Worker %d processes task %v\n", workerID, task.Data.Url)

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	store := task.EntriesStorage

	e := task.Data
	cfg := task.Config
	metrics := task.metrics

	var createdEntry feed.Entry

	dbe, err := store.Storage.FindAllByUrl(context.Background(), task.Data.Url)
	if err != nil {
		logger.Error("failed find entry by url", sl.Err(err))
	}

	// если записи в БД нет, то создаем записи
	if dbe == nil || len(dbe) == 0 {

		e, err = visitUrl(e, cfg, metrics)
		if err != nil {
			log.Printf("finishing task processing without inserting data in manticoresearch, %v", err)
			return
		}

		// разбиваем контент на части
		splitEntries := task.Splitter.SplitEntry(context.Background(), *e)
		// итерируемся по полученному срезу частей и каждую часть в БД
		for _, splitEntry := range splitEntries {
			err = insertNewEntry(&splitEntry, store.Storage, *logger)
			if err != nil {
				return
			}
		}
		createdEntry = *e
		// Увеличиваем счетчик вставок новостей с кол-вом фрагментов
		metrics.EntitiesInserted.WithLabelValues(e.Url, fmt.Sprintf("%d", len(splitEntries))).Inc()
	} else {
		if needUpdate(&dbe[0], *e) {
			log.Printf("требуется обновление, кол-во фрагментов в БД:, %v", len(dbe))
			e, err = visitUrl(e, cfg, metrics)
			if err != nil {
				log.Printf("finishing task processing without updating data in manticoresearch %v", err)
				return
			}

			splitEntries := task.Splitter.SplitEntry(context.Background(), *e)

			for n, splitEntry := range splitEntries {
				// Обязательно присваиваем created дату из БД, иначе будет перезаписан 0
				splitEntry.Created = dbe[0].Created

				timeNow := time.Unix(time.Now().Unix(), 0)
				splitEntry.UpdatedAt = &timeNow
				// Фиксируем дату публикации для ресурса МИД для соответствующих языков
				if e.ResourceID == 2 && (e.Language == "de" || e.Language == "fr" || e.Language == "pt" || e.Language == "es") {
					e.Published = dbe[0].Published
				}
				// пока n меньше, чем всего фрагментов в БД, обновляем, иначе создаем новые
				if n < len(dbe) {
					splitEntry.ID = dbe[n].ID
					err = updateOldEntry(&splitEntry, store.Storage, *logger)
					if err != nil {
						return
					}
				} else {
					err = insertNewEntry(&splitEntry, store.Storage, *logger)
					if err != nil {
						return
					}
				}
				// Увеличиваем счетчик обновления новостей с кол-вом фрагментов
				metrics.EntitiesUpdated.WithLabelValues(e.Url, fmt.Sprintf("%d", len(splitEntries))).Inc()
			}
		} else {
			//log.Printf("nothing to insert, ⌛ waiting incoming tasks…")
		}
	}

	task.Err = task.f(createdEntry)
}

func updateOldEntry(e *feed.Entry, store feed.StorageInterface, logger slog.Logger) error {
	log.Printf("before update url: %v, updated: %v", e.Url, e.Updated)
	err := store.Update(context.Background(), e)
	if err != nil {
		logger.Error(
			"failed update entry",
			slog.String("url", e.Url),
			sl.Err(err),
		)
		return err
	} else {
		logger.Info(
			"entry successful updated",
			slog.Int64("id", *e.ID),
			slog.String("url", e.Url),
		)
	}
	return nil
}

func insertNewEntry(e *feed.Entry, store feed.StorageInterface, logger slog.Logger) error {
	id, err := store.Insert(context.Background(), e)
	if err != nil {
		logger.Error(
			"failed insert entry",
			slog.String("url", e.Url),
			sl.Err(err),
		)
		return err
	}
	logger.Info(
		"entry successful inserted",
		slog.Int64("id", *id),
		slog.String("url", e.Url),
	)
	return nil
}

// visitUrl вызывает crawler, который парсит контент по ссылке,
// если crawler вернет ошибку, например в следствии read: connection reset by peer,
// соединение с сайтом разорвалось, то функция возвращает запись entry без изменений,
// если crawler вернул новую спарсенную entry (ce), то функция возвращает обновленную entry
func visitUrl(e *feed.Entry, cfg *config.Config, metrics *metrics.Metrics) (*feed.Entry, error) {

	// получаем конфигурацию для ресурса
	crawlerConfig, err := GetCrawlerConfigByResourceID(cfg, e.ResourceID, e.Language)
	if err != nil {
		return nil, fmt.Errorf("error getting crawler config: %v", err)
	}

	switch e.ResourceID {
	case 2:
		ce, err := crawler.VisitMid(e, crawlerConfig, metrics)
		if err != nil {
			return nil, err
		}
		return ce, nil
	case 3:
		ce, err := crawler.VisitMil(e, crawlerConfig, metrics)
		if err != nil {
			return e, nil
		}
		return ce, nil
	}

	return e, nil
}

func needUpdate(dbe *feed.Entry, e feed.Entry) bool {
	// Если заголовок в базе пустой, значит необходимо произвести обновление записи.
	// Было замечено, что иногда с сайта МО записи попадают с пустыми значениями заголовка и контента
	// Хотя позже проверя источник видно, что и заголовок и контент присутствует,
	// но т.к. на ленте МО нет сигнализации об обновлении записей, принято решение с помощью дополнительных условий
	// проверять необходимость обновления полученной ранее записи
	if len(strings.TrimSpace(dbe.Title)) == 0 {
		log.Printf("WARRNING! Title was empty, UPDATING entry %v", dbe.Url)
		return true
	}
	// Если контент в базе пустой, и это не сайт kremlin.ru, производим обновление записи
	if len(strings.TrimSpace(dbe.Content)) == 0 && dbe.ResourceID != 1 {
		log.Printf("WARRNING! Content was empty, UPDATING entry %v", dbe.Url)
		return true
	}
	// если поле updated nil, значит в ленте это значение не установлено, и значит мы не можем руководствоваться и обновлять
	// информацию с помощью этого поля, поэтому возвращаем false
	if dbe == nil || e.Updated == nil {
		return false
	}

	// Приводим время в обоих объектах к нужному часовому поясу в зависимости от ресурса.
	// GMT+4 на сайте Кремля, GMT+3 на других ресурсах.
	var loc *time.Location
	var dbeTime, eTime time.Time
	if dbe.ResourceID == 1 {
		loc, _ = time.LoadLocation("Etc/GMT-4")
		dbeTime = dbe.Updated.In(loc)
		eTime = e.Updated.In(loc)
	} else {
		loc, _ = time.LoadLocation("Etc/GMT-3")
		dbeTime = dbe.Updated.In(loc)
		eTime = e.Updated.In(loc)
	}

	if dbeTime != eTime && dbe.ResourceID != 2 {
		log.Printf("Url %v `updated` fields do not match dbe updated %v, e: %v", dbe.Url, dbeTime, eTime)
		return true
	}

	//Для ленты сайта mid language ru. Проверка обновления записи каждые 6 часов
	if dbe.Language == "ru" {
		if dbeTime.Add(6*time.Hour).Sub(eTime) <= 0 && dbe.ResourceID == 2 {
			log.Printf("dbeTime.Add(1*time.Hour).Sub(eTime) <= 0 && RID == 2, lang %v", dbe.Language)
			log.Printf("🚩 Url %v updated. DB time %v, current time: %v ", dbe.Url, dbeTime, eTime)
			return true
		}
	}
	//Для ленты сайта mid language en. Проверка обновления записи каждые 6 часов
	if dbe.Language == "en" {
		if dbeTime.Add(6*time.Hour).Sub(eTime) <= 0 && dbe.ResourceID == 2 {
			log.Printf("dbeTime.Add(6*time.Hour).Sub(eTime) <= 0 && RID == 2, lang %v", dbe.Language)
			log.Printf("🚩 Url %v updated. DB time %v, current time: %v ", dbe.Url, dbeTime, eTime)
			return true
		}
	}
	//Для ленты сайта mid language de,fr,ed,pt. Проверка обновления записи каждые 24 часа
	if dbe.Language == "de" || dbe.Language == "fr" || dbe.Language == "es" || dbe.Language == "pt" {
		if dbeTime.Add(24*time.Hour).Sub(eTime) <= 0 && dbe.ResourceID == 2 {
			log.Printf("dbeTime.Add(24*time.Hour).Sub(eTime) <= 0 && RID == 2, lang %v", dbe.Language)
			log.Printf("🚩 Url %v updated. DB time %v, current time: %v ", dbe.Url, dbeTime, eTime)
			return true
		}
	}

	return false
}

func GetCrawlerConfigByResourceID(cfg *config.Config, resourceID int, lang string) (*config.Crawler, error) {
	for _, parser := range cfg.Parsers {
		if parser.ResourceID == resourceID && parser.Lang == lang {
			return &parser.Crawler, nil
		}
	}
	return nil, fmt.Errorf("crawler config not found for resource_id: %d and lang: %s", resourceID, lang)
}
