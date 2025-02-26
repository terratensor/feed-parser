package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/rssfeed"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
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

	// Основной фид, содержащий все новости
	svoddFeed := &rssfeed.RssFeed{
		Title:       "Поиск по сайтам Кремля, МИД и Минобороны",
		Link:        "https://feed.svodd.ru",
		Description: "Поиск по сайтам Президента России, Министерства иностранных дел Российской Федерации, Министерство обороны Российской Федерации",
	}

	// Фид для Кремля (ResourceID = 1)
	kremlinFeed := &rssfeed.RssFeed{
		Title:       "Новости Кремля",
		Link:        "https://rss.feed.svodd.ru/kremlin.xml",
		Description: "Новости с сайта Президента Российской Федерации",
	}

	// Фид для МИД (ResourceID = 2)
	midFeed := &rssfeed.RssFeed{
		Title:       "Новости МИД",
		Link:        "https://rss.feed.svodd.ru/mid.xml",
		Description: "Новости с сайта Министерства иностранных дел Российской Федерации",
	}

	// Фид для Минобороны (ResourceID = 3)
	milFeed := &rssfeed.RssFeed{
		Title:       "Новости Минобороны",
		Link:        "https://rss.feed.svodd.ru/mil.xml",
		Description: "Новости с сайта Министерства обороны Российской Федерации",
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

			// Пропускаем все записи, опубликованные на языке отличном от русского
			if e.Language != "ru" && e.Language != "" {
				limitCount++
				continue
			}

			// Создаем элемент RSS
			item := &rssfeed.RssItem{
				Title:       e.Title,
				Link:        link,
				PubDate:     rssfeed.AnyTimeFormat(time.RFC1123Z, *e.Published),
				Author:      populateAuthorField(e.Author, e.ResourceID),
				Content:     e.Content,
				Description: populateDescription(e),
			}

			// Если есть URL источника новости, добавляем его в <source>
			if e.Url != "" {
				sourceName := "Оригинальный источник" // Значение по умолчанию
				if name, ok := authorMap[e.ResourceID]; ok {
					sourceName = name // Используем значение из authorMap
				}
				item.Source = &rssfeed.RssSource{
					URL:  e.Url,      // URL источника новости
					Name: sourceName, // Название источника из authorMap
				}
			}

			// Добавляем новость в основной фид
			svoddFeed.Add(item)

			// Добавляем новость в соответствующий фид на основе ResourceID
			switch e.ResourceID {
			case 1:
				kremlinFeed.Add(item)
			case 2:
				midFeed.Add(item)
			case 3:
				milFeed.Add(item)
			}

			itemCount++
			limitCount++
		}
		if limitCount >= limit {
			break
		}
	}

	// Сохраняем все фиды в файлы
	saveFeedToFile(svoddFeed, "./static/rss.xml")
	saveFeedToFile(kremlinFeed, "./static/kremlin.xml")
	saveFeedToFile(midFeed, "./static/mid.xml")
	saveFeedToFile(milFeed, "./static/mil.xml")

	log.Printf("🚩 Созданы RSS-фиды. Всего записей: %d\n", itemCount)
}

// saveFeedToFile сохраняет RSS-фид в файл
func saveFeedToFile(feed *rssfeed.RssFeed, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("failed to create file %s: %v", filename, err)
		return
	}
	defer file.Close()

	err = rssfeed.WriteXML(feed, file)
	if err != nil {
		log.Printf("failed to write XML to file %s: %v", filename, err)
	}
}

// populateDescription generates description for a feed entry.
//
// It takes a feed.Entry as parameter and returns a string.
func populateDescription(entry feed.Entry) string {
	description := entry.Summary
	if description == "" {
		description = entry.Title
	}
	return description
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
