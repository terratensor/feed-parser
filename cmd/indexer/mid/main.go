package main

import (
	"github.com/terratensor/feed-parser/internal/app"
	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/indexer/mid"
	"github.com/terratensor/feed-parser/internal/model/link"
	"github.com/terratensor/feed-parser/internal/splitter"
	"github.com/terratensor/feed-parser/internal/workerpool"
	"log"
	"sync"
)

func main() {
	cfg := config.MustLoad()

	//fp.UserAgent = "PostmanRuntime/7.36.3"
	ch := make(chan feed.Entry, cfg.EntryChanBuffer)

	wg := &sync.WaitGroup{}
	for _, url := range cfg.URLS {

		wg.Add(1)
		midIndexer := mid.NewIndexer(link.Link{
			Url:        url.Url,
			Lang:       url.Lang,
			ResourceID: url.ResourceID,
			UserAgent:  url.UserAgent,
		}, *cfg.Delay, *cfg.RandomDelay)

		go midIndexer.Run(ch, wg)
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
