package main

import (
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/service/internal/entities/feed"
	"github.com/terratensor/feed-parser/service/internal/parser"
	"github.com/terratensor/feed-parser/service/internal/workerpool"
	"log"
	"sync"
	"time"
)

func main() {

	fp := gofeed.NewParser()
	//fp.UserAgent = "PostmanRuntime/7.36.3"
	ch := make(chan feed.Entry, 20)

	urls := []parser.Link{
		{Url: "http://kremlin.ru/events/all/feed/", Lang: "ru", ResourceID: 1},
		{Url: "http://en.kremlin.ru/events/all/feed", Lang: "en", ResourceID: 1},
		{Url: "https://mid.ru/ru/rss.php", Lang: "ru", ResourceID: 2},
		{Url: "https://mid.ru/en/rss.php", Lang: "en", ResourceID: 2},
		{Url: "https://mid.ru/de/rss.php", Lang: "de", ResourceID: 2},
		{Url: "https://mid.ru/fr/rss.php", Lang: "fr", ResourceID: 2},
		{Url: "https://mid.ru/es/rss.php", Lang: "es", ResourceID: 2},
		{Url: "https://mid.ru/pt/rss.php", Lang: "pt", ResourceID: 2},
		//{Url: "https://mid.ru/cn/rss.php", Lang: "ru", ResourceID: 1},
		//{Url: "https://mid.ru/ar/rss.php", Lang: "ru", ResourceID: 1},
		{Url: "https://function.mil.ru/rss_feeds/reference_to_general.htm?contenttype=xml", Lang: "ru", ResourceID: 3},
		//{Url: "https://tass.ru/rss/v2.xml", Lang: "ru", ResourceID: 4},
		//{Url: "https://tass.com/rss/v2.xml", Lang: "en", ResourceID: 4},
	}

	wg := &sync.WaitGroup{}
	for _, url := range urls {
		wg.Add(1)
		p := parser.NewParser(url, 60*time.Second, 150*time.Second)
		go p.Run(ch, fp, wg)
	}

	var allTask []*workerpool.Task

	pool := workerpool.NewPool(allTask, 5)

	go func() {
		for {
			task := workerpool.NewTask(func(data interface{}) error {
				//entry := data.(*feed.Entry)
				//log.Printf("data Url %#v\r\r\r\r", entry.Url)

				//taskID := entry.ID
				//time.Sleep(50 * time.Millisecond)
				//fmt.Printf("Task %v processed\n", entry.Url)
				return nil
			}, <-ch)
			pool.AddTask(task)
		}
	}()

	pool.RunBackground()

	wg.Wait()
	log.Println("finished, all workers successfully stopped.")
}
