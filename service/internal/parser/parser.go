package parser

import (
	"context"
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/service/internal/entities/feed"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Link struct {
	Url        string
	Lang       string
	ResourceID int
}

type Parser struct {
	Link        Link
	Delay       time.Duration
	RandomDelay time.Duration
}

// NewParser creates a new Parser with the given URL, delay, and randomDelay.
func NewParser(url Link, delay time.Duration, randomDelay time.Duration) *Parser {
	np := &Parser{
		Link:        url,
		Delay:       delay,
		RandomDelay: randomDelay,
	}
	return np
}

func (p *Parser) Run(ch chan feed.Entry, fp *gofeed.Parser, wg *sync.WaitGroup) {
	defer wg.Done()

	for {

		randomDelay := time.Duration(0)
		if p.RandomDelay != 0 {
			randomDelay = time.Duration(rand.Int63n(int64(p.RandomDelay)))
		}
		time.Sleep(p.Delay + randomDelay)

		log.Printf("started parser for given url: %v", p.Link.Url)
		gf, err := fp.ParseURL(p.Link.Url)
		if err != nil {
			log.Printf("ERROR: %v, %v", err, p.Link.Url)
			continue
		}
		log.Printf("fetched the contents of a given url %v", p.Link.Url)
		entries := makeEntries(gf.Items, p.Link)

		select {
		case <-context.Background().Done():
			break
		default:
		}

		for _, entry := range entries {
			ch <- entry
		}
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
