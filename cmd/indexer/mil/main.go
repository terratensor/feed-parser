package main

import (
	"log"
	"sync"
	"time"

	"github.com/terratensor/feed-parser/internal/app"
	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/indexer/mil"
	"github.com/terratensor/feed-parser/internal/model/link"
	"github.com/terratensor/feed-parser/internal/splitter"
	"github.com/terratensor/feed-parser/internal/workerpool"
)

func main() {
	cfg := config.MustLoad()

	// output current time zone
	tnow := time.Now()
	tz, _ := tnow.Zone()
	log.Printf("Local time zone %s. Parser started at %s", tz, tnow.Format("2006-01-02T15:04:05.000 MST"))

	//fp.UserAgent = "PostmanRuntime/7.36.3"
	ch := make(chan feed.Entry, cfg.EntryChanBuffer)

	wg := &sync.WaitGroup{}
	for _, url := range cfg.Parsers {

		wg.Add(1)
		milIndexer := mil.NewIndexer(link.Link{
			Url:        url.Url,
			Lang:       url.Lang,
			ResourceID: url.ResourceID,
		}, cfg)

		go milIndexer.Run(ch, wg)
	}

	var allTask []*workerpool.Task

	pool := workerpool.NewPool(allTask, cfg.Workers)
	sp := splitter.NewSplitter(cfg.Splitter.OptChunkSize, cfg.Splitter.MaxChunkSize)
	entriesStore := app.NewEntriesStorage(cfg.ManticoreIndex)

	go func() {
		for {
			task := workerpool.NewTask(func(data interface{}) error {
				return nil
			}, <-ch, sp, entriesStore)
			pool.AddTask(task)
		}
	}()

	pool.RunBackground()

	wg.Wait()
	log.Println("Indexer finished, all workers successfully stopped.")
}
