package main

import (
	"context"
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/service/internal/entities/feed"
	"github.com/terratensor/feed-parser/service/internal/workerpool"
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

type Link struct {
	Url        string
	Lang       string
	ResourceID int
}

func main() {

	// можно протащить logger через value контекста, и далее вызывать его не передавая
	// в качестве параметра функции или поля структуры, например, используя logger := ctx.Value("logger").(slog.Logger)
	//ctx, cancel := signal.NotifyContext(context.WithValue(context.Background(), "logger", *logger), os.Interrupt)
	//defer cancel()
	//
	//var storage feed.StorageInterface
	//
	//manticoreClient, err := manticore.New("feed")
	//if err != nil {
	//	log.Fatalf("failed to initialize manticore client, %v", err)
	//	os.Exit(1)
	//}
	//
	//storage = manticoreClient
	wg := &sync.WaitGroup{}

	fp := gofeed.NewParser()
	ch := make(chan feed.Entry, 20)

	urls := []Link{
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

	for _, url := range urls {
		wg.Add(1)
		go runParser(url, ch, fp, wg)
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

func runParser(url Link, ch chan feed.Entry, fp *gofeed.Parser, wg *sync.WaitGroup) {
	defer wg.Done()
	for {

		gf, err := fp.ParseURL(url.Url)
		if err != nil {
			log.Printf("ERROR: %v", err)
			continue
		}
		entries := makeEntries(gf.Items, url)

		select {
		case <-context.Background().Done():
			break
		default:
		}

		for _, entry := range entries {
			ch <- entry
		}

		n := 60 + rand.Intn(20)
		d := time.Duration(n)
		time.Sleep(d * time.Second)
	}
}

func makeEntries(items []*gofeed.Item, url Link) []feed.Entry {
	var entries []feed.Entry

	for _, item := range items {

		e := &feed.Entry{
			Language:   url.Lang,
			Title:      item.Title,
			Url:        item.Link,
			Updated:    item.UpdatedParsed,
			Published:  item.PublishedParsed,
			Summary:    item.Description,
			Content:    item.Content,
			ResourceID: url.ResourceID,
		}

		entries = append(entries, *e)
	}

	return entries
}
