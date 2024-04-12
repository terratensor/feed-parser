package main

import (
	"context"
	"fmt"
	"github.com/gorilla/feeds"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	manticoreClient, err := manticore.New("feed")
	if err != nil {
		log.Printf("failed to initialize manticore client: %v", err)
		os.Exit(1)
	}
	entries := feed.NewFeedStorage(manticoreClient)

	limit := 203

	ch, err := entries.Find(ctx, 24*8*time.Hour)
	if err != nil {
		log.Fatalf("failed to find all entries: %v", err)
	}

	svoddFeed := &feeds.Feed{
		Title:       "Поиск по kremlin.ru, mid.ru, mil.ru",
		Link:        &feeds.Link{Href: "https://rss.feed.svodd.ru"},
		Description: "Поиск по сайтам Президента России, Министерства иностранных дел Российской Федерации, Министерство обороны Российской Федерации",
		Updated:     time.Now(),
		Created:     time.Now(),
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

			link := makeEntryUrl(e.Url)

			//log.Println(e.ID)
			item := &feeds.Item{
				Title:       e.Title,
				Id:          link,
				Updated:     *e.Published,
				Description: e.Summary,
				Link:        &feeds.Link{Href: link},
				Author:      &feeds.Author{Name: e.Author},
				Content:     e.Content,
			}

			//log.Println(item.Title)

			svoddFeed.Add(item)
			count++

			//svoddFeed.Add(&feeds.Item{
			//	Title:       e.Title,
			//	Id:          e.Url,
			//	Updated:     *e.UpdatedAt,
			//	Description: e.Summary,
			//	Author:      &feeds.Author{Name: e.Author},
			//	//Link:        &feeds.Link{Href: e.Url},
			//
			//})

			//rss, err := feed.ToRss()
			//if err != nil {
			//	log.Fatal(err)
			//}

			//json, err := feed.ToJSON()
			//if err != nil {
			//	log.Fatal(err)
			//}

			//fmt.Println(atom, "\n", rss, "\n", json)

		}
		if count >= limit {
			log.Println("here")
			break
		}
	}
	atom, err := svoddFeed.ToAtom()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(atom)
}

func makeEntryUrl(url string) string {
	return fmt.Sprintf("https://feed.svodd.ru/entry?url=%v", url)
}
