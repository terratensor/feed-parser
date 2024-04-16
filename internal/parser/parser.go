package parser

import (
	"context"
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/htmlnode"
	"github.com/terratensor/feed-parser/internal/lib/logger/sl"
	"github.com/terratensor/feed-parser/internal/model/link"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Parser struct {
	Link        link.Link
	Delay       time.Duration
	RandomDelay time.Duration
}

// NewParser creates a new Parser with the given URL, delay, and randomDelay.
func NewParser(cfg config.Parser, delay time.Duration, randomDelay time.Duration) *Parser {

	newLink := link.NewLink(cfg.Url, cfg.Lang, cfg.ResourceID, cfg.UserAgent)
	if cfg.Delay != nil {
		delay = *cfg.Delay
	}
	if cfg.RandomDelay != nil {
		randomDelay = *cfg.RandomDelay
	}
	np := &Parser{
		Link:        *newLink,
		Delay:       delay,
		RandomDelay: randomDelay,
	}
	return np
}

func (p *Parser) Run(ch chan feed.Entry, fp *gofeed.Parser, wg *sync.WaitGroup) {

	log.Printf("🚩 run parser with params: delay: %v, random delay: %v, url: %v", p.Delay, p.RandomDelay, p.Link.Url)

	defer wg.Done()

	for {

		randomDelay := time.Duration(0)
		if p.RandomDelay != 0 {
			randomDelay = time.Duration(rand.Int63n(int64(p.RandomDelay)))
		}
		time.Sleep(p.Delay + randomDelay)

		log.Printf("started parser for given url: %v", p.Link.Url)
		entries := p.getEntries(fp)
		log.Printf("fetched the contents of a given url %v", p.Link.Url)

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

// Получает записи ленты, если сайт kremlin,
// то запускает kremlin парсер,
// иначе запускает gofeed.Parser
func (p *Parser) getEntries(fp *gofeed.Parser) []feed.Entry {

	var entries []feed.Entry
	if p.Link.ResourceID == 1 {
		entries = append(entries, p.parseKremlin(p.Link.Url)...)
	} else {
		gf, err := fp.ParseURL(p.Link.Url)
		if err != nil {
			log.Printf("ERROR: %v, %v", err, p.Link.Url)
			return nil
		}
		entries = append(entries, feed.MakeEntries(gf.Items, p.Link)...)
	}

	return entries
}

func (p *Parser) parseKremlin(url string) []feed.Entry {

	node, err := htmlnode.GetTopicBody(url)
	if os.IsTimeout(err) {
		log.Printf("server timeout error %v", err)
		return nil
	}
	if err != nil {
		log.Printf("failed to decode request body %v", sl.Err(err))
		return nil
	}

	return p.parseEntries(node)
}
