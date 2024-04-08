package workerpool

import (
	"context"
	"fmt"
	"github.com/terratensor/feed-parser/internal/crawler"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/lib/logger/sl"
	"github.com/terratensor/feed-parser/internal/splitter"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"
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

func NewTask(f func(interface{}) error, data feed.Entry, splitter *splitter.Splitter, storage *feed.Entries) *Task {
	return &Task{
		f:              f,
		Data:           &data,
		Splitter:       *splitter,
		EntriesStorage: storage,
	}
}

func process(workerID int, task *Task) {
	fmt.Printf("Worker %d processes task %v\n", workerID, task.Data.Url)

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	store := task.EntriesStorage

	dbe, err := store.Storage.FindAllByUrl(context.Background(), task.Data.Url)
	e := task.Data

	if err != nil {
		logger.Error("failed find entry by url", sl.Err(err))
	}

	// если записи в БД нет, то создаем записи
	if dbe == nil || len(dbe) == 0 {

		e, err = visitUrl(e)
		if err != nil {
			log.Println("finishing task processing without updating data in manticoresearch")
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
	} else {
		if needUpdate(&dbe[0], *e) {
			log.Printf("требуется обновление, кол-во фрагментов в БД:, %v", len(dbe))
			e, err = visitUrl(e)
			if err != nil {
				log.Println("finishing task processing without updating data in manticoresearch")
				return
			}

			splitEntries := task.Splitter.SplitEntry(context.Background(), *e)

			// всего фрагментов
			count := len(splitEntries)

			for n, splitEntry := range splitEntries {
				// Обязательно присваиваем created дату из БД, иначе будет перезаписан 0
				splitEntry.Created = dbe[0].Created

				timeNow := time.Unix(time.Now().Unix(), 0)
				splitEntry.UpdatedAt = &timeNow

				if n+1 < count {
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
			}
		} else {
			//log.Printf("nothing to insert, ⌛ waiting incoming tasks…")
		}
	}

	task.Err = task.f(dbe)
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
func visitUrl(e *feed.Entry) (*feed.Entry, error) {
	switch e.ResourceID {
	case 2:
		ce, err := crawler.VisitMid(e)
		if err != nil {
			return nil, err
		}
		return ce, nil
	case 3:
		ce, err := crawler.VisitMil(e)
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
	////TODO обновление закомментировано, необходимо модифицировать и настроить обновление на сайте мид
	//intervalT := dbeTime.Add(1 * time.Hour)
	//log.Printf("dbeTime.Add(1*time.Hour), %v\n", intervalT)
	//log.Printf("current eTime, %v\n", eTime)
	//log.Printf("Sub(eTime), %v\n", intervalT.Sub(eTime))
	////Для ленты сайта mid
	//if dbeTime.Add(1*time.Hour).Sub(eTime) <= 0 && dbe.ResourceID == 2 {
	//	log.Printf("dbeTime.Add(1*time.Hour).Sub(eTime) <= 0 && dbe.ResourceID == 2, condition id true")
	//	log.Printf("Url %v `updated` fields do not match dbe updated dbe: %v, e: %v ", dbe.Url, dbeTime, eTime)
	//	//return true
	//	// Пока только фиксируем и не обновляем, возвращаем false
	//	return false
	//}

	return false
}
