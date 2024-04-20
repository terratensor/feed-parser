package main

import (
	"context"
	"fmt"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/rssfeed"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
	"unicode/utf8"
)

var authorMap = map[int]string{
	1: "Президент Российской Федерации",
	2: "Министерство иностранных дел Российской Федерации",
	3: "Министерство обороны Российской Федерации",
}

func main() {

	log.Printf("Service started at %s", time.Now().Format("2006-01-02T15:04:05.000 MST"))
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	delayStr := os.Getenv("GENERATOR_DELAY")
	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		delay = 15 * time.Minute
	}
	log.Printf("Delay: %s", delay)

	index := os.Getenv("MANTICORE_INDEX")
	if index == "" {
		index = "feed"
	}
	log.Printf("Index: %s", index)

	manticoreClient, err := manticore.New(index)
	if err != nil {
		log.Printf("failed to initialize manticore client: %v", err)
		os.Exit(1)
	}
	entries := feed.NewFeedStorage(manticoreClient)

	duration := 24 * 8 * time.Hour

	wg := &sync.WaitGroup{}

	for {
		wg.Add(1)
		go generateFeed(ctx, entries, duration, wg)
		wg.Wait()
		time.Sleep(delay)
	}
}

func generateFeed(ctx context.Context, entries *feed.Entries, duration time.Duration, wg *sync.WaitGroup) {

	defer wg.Done()

	limit := entries.Storage.CalculateLimitCount(duration)
	ch, err := entries.Find(ctx, duration)
	if err != nil {
		log.Fatalf("failed to find all entries: %v", err)
	}

	svoddFeed := &rssfeed.RssFeed{
		Title:       "Поиск по сайтам Кремля, МИД и Минобороны",
		Link:        "https://rss.feed.svodd.ru",
		Description: "Поиск по сайтам Президента России, Министерства иностранных дел Российской Федерации, Министерство обороны Российской Федерации",
	}
	limitCount := 0
	itemCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}

			link := makeEntryUrl(e.Url, e.Language)

			if utf8.RuneCountInString(e.Title) > 200 {
				limitCount++
				continue
			}

			item := &rssfeed.RssItem{
				Title:       e.Title,
				Link:        link,
				PubDate:     rssfeed.AnyTimeFormat(time.RFC1123Z, *e.Published),
				Author:      populateAuthorField(e.Author, e.ResourceID),
				Content:     e.Content,
				Description: e.Summary,
			}

			svoddFeed.Add(item)
			itemCount++
			limitCount++
		}
		if limitCount >= limit {
			break
		}
	}

	file, err := os.Create("./static/rss.xml")
	if err != nil {
		log.Printf("failed to create file: %v", err)
	}
	defer file.Close()

	err = rssfeed.WriteXML(svoddFeed, file)
	if err != nil {
		log.Printf("failed to write xml: %v", err)
	}

	log.Printf("🚩 Создан rss.xml. Всего записей: %d\n", itemCount)
}

func populateAuthorField(author string, resourceID int) string {
	if val, ok := authorMap[resourceID]; ok {
		if resourceID == 3 && author != "" {
			return author
		}
		return val
	}
	return author
}

func makeEntryUrl(url string, language string) string {
	base := "https://feed.svodd.ru"
	if language != "ru" && language != "" {
		base += "/" + language
	}
	return fmt.Sprintf("%s/entry?url=%v", base, url)
}
