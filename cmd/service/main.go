package main

import (
	"log"
	"net/url"
	"sync"

	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/internal/app"
	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/indexnow"
	"github.com/terratensor/feed-parser/internal/parser"
	"github.com/terratensor/feed-parser/internal/splitter"
	"github.com/terratensor/feed-parser/internal/workerpool"
)

func main() {

	cfg := config.MustLoad()

	fp := gofeed.NewParser()
	fp.UserAgent = cfg.UserAgent

	ch := make(chan feed.Entry, cfg.EntryChanBuffer)

	wg := &sync.WaitGroup{}
	for _, parserCfg := range cfg.Parsers {

		wg.Add(1)
		p := parser.NewParser(parserCfg, *cfg.Delay, *cfg.RandomDelay)

		go p.Run(ch, fp, wg)
	}

	var allTask []*workerpool.Task

	pool := workerpool.NewPool(allTask, cfg.Workers)
	sp := splitter.NewSplitter(cfg.Splitter.OptChunkSize, cfg.Splitter.MaxChunkSize)
	entriesStore := app.NewEntriesStorage(cfg.ManticoreIndex)

	// Передаем в конструктор indexNow параметр enabled инициализируем индексацию
	indexNow := indexnow.NewIndexNow(cfg.IndexNow)

	go func() {
		for {
			task := workerpool.NewTask(func(data interface{}) error {
				if cfg.Env != "prod" {
					return nil
				}
				e := data.(feed.Entry)
				processEntry(e, indexNow)
				return nil
			}, <-ch, sp, entriesStore)
			pool.AddTask(task)
		}
	}()

	pool.RunBackground()

	wg.Wait()
	log.Println("finished, all workers successfully stopped.")
}

func processEntry(e feed.Entry, indexNow *indexnow.IndexNow) {
	// если индексация не включена, то выходим
	if indexNow == nil {
		return
	}
	if e.Url != "" && e.Language == "ru" {

		var u = url.URL{
			Scheme: "https",
			Host:   "feed.svodd.ru",
			Path:   "entry",
		}
		q := u.Query()
		q.Set("url", e.Url)
		u.RawQuery = q.Encode()

		// if e.Language != "ru" && e.Language != "" {
		// 	u.Path = fmt.Sprintf("%v/entry", e.Language)
		// }

		err := indexNow.Get(u.String())

		if err != nil {
			log.Printf("indexNow error: %v", err)
		}
	}
}
