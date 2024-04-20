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
)

var authorMap = map[int]string{
	1: "Президент Российской Федерации",
	2: "Министерство иностранных дел Российской Федерации",
	3: "Министерство обороны Российской Федерации",
}

func main() {

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	manticoreClient, err := manticore.New("feed")
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
		time.Sleep(30 * time.Second)
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
	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}

			link := makeEntryUrl(e.Url, e.Language)

			item := &rssfeed.RssItem{
				Title:       e.Title,
				Link:        link,
				PubDate:     rssfeed.AnyTimeFormat(time.RFC1123Z, *e.Published),
				Author:      populateAuthorField(e.Author, e.ResourceID),
				Content:     e.Content,
				Description: e.Summary,
			}

			svoddFeed.Add(item)
			count++
		}
		if count >= limit {
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

	log.Printf("🚩 Создан rss.xml. Всего записей: %d\n", count)
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
