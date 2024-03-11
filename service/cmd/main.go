package main

import (
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/service/internal/config"
	"github.com/terratensor/feed-parser/service/internal/entities/feed"
	"github.com/terratensor/feed-parser/service/internal/parser"
	"github.com/terratensor/feed-parser/service/internal/workerpool"
	"log"
	"sync"
)

func main() {

	cfg := config.MustLoad()

	fp := gofeed.NewParser()
	//fp.UserAgent = "PostmanRuntime/7.36.3"
	ch := make(chan feed.Entry, cfg.EntryChanBuffer)

	wg := &sync.WaitGroup{}
	for _, url := range cfg.URLS {

		wg.Add(1)
		p := parser.NewParser(parser.Link{
			Url:        url.Url,
			Lang:       url.Lang,
			ResourceID: url.ResourceID,
		}, *cfg.Delay, *cfg.RandomDelay)

		go p.Run(ch, fp, wg)
	}

	var allTask []*workerpool.Task

	pool := workerpool.NewPool(allTask, 5)

	go func() {
		for {
			task := workerpool.NewTask(func(data interface{}) error {
				return nil
			}, <-ch)
			pool.AddTask(task)
		}
	}()

	pool.RunBackground()

	wg.Wait()
	log.Println("finished, all workers successfully stopped.")
}
