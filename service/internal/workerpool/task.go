package workerpool

import (
	"context"
	"fmt"
	"github.com/terratensor/feed-parser/service/internal/crawler"
	"github.com/terratensor/feed-parser/service/internal/entities/feed"
	"github.com/terratensor/feed-parser/service/internal/lib/logger/sl"
	"github.com/terratensor/feed-parser/service/internal/storage/manticore"
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
	Data *feed.Entry
	f    func(interface{}) error
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

func NewTask(f func(interface{}) error, data feed.Entry) *Task {
	return &Task{f: f, Data: &data}
}

func process(workerID int, task *Task) {
	fmt.Printf("Worker %d processes task %v\n", workerID, task.Data.Url)

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	store := NewTaskStorage()

	dbe, err := store.Storage.FindByUrl(context.Background(), task.Data.Url)
	e := task.Data

	if err != nil {
		logger.Error("failed find entry by url", sl.Err(err))
	}
	if dbe == nil {

		e, err = visitUrl(e)
		if err != nil {
			log.Println("finishing task processing without updating data in manticoresearch")
			return
		}

		id, err := store.Storage.Insert(context.Background(), e)
		if err != nil {
			logger.Error(
				"failed insert entry",
				slog.Int64("id", *id),
				slog.String("url", e.Url),
				sl.Err(err),
			)
		}
		logger.Info(
			"entry successful inserted",
			slog.Int64("id", *id),
			slog.String("url", e.Url),
		)
	} else {
		if needUpdate(dbe, *e) {
			e.ID = dbe.ID
			e, err = visitUrl(e)
			if err != nil {
				log.Println("finishing task processing without updating data in manticoresearch")
				return
			}
			// Обязательно присваиваем created дату из БД, иначе будет перезаписан 0
			e.Created = dbe.Created

			log.Printf("before update url: %v, updated: %v", e.Url, e.Updated)
			err = store.Storage.Update(context.Background(), e)
			if err != nil {
				logger.Error(
					"failed update entry",
					slog.Int64("id", *e.ID),
					slog.String("url", e.Url),
					sl.Err(err),
				)
			} else {
				logger.Info(
					"entry successful updated",
					slog.Int64("id", *e.ID),
					slog.String("url", e.Url),
				)
			}
		} else {
			//log.Printf("nothing to insert, ⌛ waiting incoming tasks…")
		}
	}

	task.Err = task.f(dbe)
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
	// Если контент в базе пустой, значит необходимо произвести обновление записи
	if len(strings.TrimSpace(dbe.Content)) == 0 {
		log.Printf("WARRNING! Content was empty, UPDATING entry %v", dbe.Url)
		return true
	}
	// если поле updated nil, значит в ленте это значение не установлено, и значит мы не можем руководствоваться и обновлять
	// информацию с помощью этого поля, поэтому возвращаем false
	if dbe == nil || e.Updated == nil {
		return false
	}
	// Приводим время в обоих объектах к GMT+4, как на сайте Кремля
	loc, _ := time.LoadLocation("Etc/GMT-4")
	dbeTime := dbe.Updated.In(loc)
	eTime := e.Updated.In(loc)

	if dbeTime != eTime {
		log.Printf("Url %v `updated` fields do not match dbe updated %v", dbe.Url, dbeTime)
		log.Printf("Url %v `updated` fields do not match prs updated %v", e.Url, eTime)
		return true
	}

	return false
}
