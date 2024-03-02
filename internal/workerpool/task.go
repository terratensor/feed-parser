package workerpool

import (
	"context"
	"fmt"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/lib/logger/sl"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
	"log"
	"log/slog"
	"os"
	"time"
)

/**
Task содержит все необходимое для обработки задачи.
Мы передаем ей Data и функцию f, которая должна быть выполнена, с помощью функции process.
Функция f принимает Data в качестве параметра для обработки, а также храним возвращаемую ошибку
*/

type Task struct {
	Err     error
	Entries *feed.Entries
	Data    *feed.Entry
	f       func(interface{}) error
}

func NewTaskStorage() *feed.Entries {
	var storage feed.StorageInterface

	manticoreClient, err := manticore.New("feed")
	if err != nil {
		log.Fatalf("failed to initialize manticore client, %v", err)
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
		if !matchTimes(dbe, *e) {
			e.ID = dbe.ID
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

func matchTimes(dbe *feed.Entry, e feed.Entry) bool {
	// если поле updated nil, значит в ленте это значение не установлено, и значит мы не можем руководствоваться и обновлять
	// информацию с помощью этого поля, поэтому возвращаем true — совпадение, таска будет выполнена
	if dbe == nil || e.Updated == nil {
		return true
	}
	// Приводим время в обоих объектах к GMT+4, как на сайте Кремля
	loc, _ := time.LoadLocation("Etc/GMT-4")
	dbeTime := dbe.Updated.In(loc)
	eTime := e.Updated.In(loc)

	if dbeTime != eTime {
		log.Printf("`updated` fields do not match dbe updated %v", dbeTime)
		log.Printf("`updated` fields do not match prs updated %v", eTime)
		return false
	}
	return true
}
