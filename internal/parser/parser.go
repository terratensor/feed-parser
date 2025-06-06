package parser

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/htmlnode"
	"github.com/terratensor/feed-parser/internal/lib/logger/sl"
	"github.com/terratensor/feed-parser/internal/metrics"
	"github.com/terratensor/feed-parser/internal/model/link"
)

type Parser struct {
	Link        link.Link
	Delay       time.Duration
	RandomDelay time.Duration
	metrics     *metrics.Metrics
}

// NewParser creates a new Parser instance with configuration from both main config and parser-specific config.
// It initializes a Parser with URL, language, resource ID, user agent, delay settings, and metrics.
// The delay and random delay values can be overridden by parser-specific config if provided.
//
// Parameters:
//   - cfg: Parser-specific configuration
//   - mainCfg: Main application configuration
//   - metrics: Metrics instance for tracking parser performance
//
// Returns:
//   - *Parser: A new configured Parser instance
func NewParser(cfg config.Parser, mainCfg config.Config, metrics *metrics.Metrics) *Parser {

	newLink := link.NewLink(cfg.Url, cfg.Lang, cfg.ResourceID, cfg.UserAgent)

	delay := *mainCfg.Delay
	if cfg.Delay != nil {
		delay = *cfg.Delay
	}

	randomDelay := *mainCfg.RandomDelay
	if cfg.RandomDelay != nil {
		randomDelay = *cfg.RandomDelay
	}

	np := &Parser{
		Link:        *newLink,
		Delay:       delay,
		RandomDelay: randomDelay,
		metrics:     metrics,
	}
	return np
}

func (p *Parser) Run(ch chan feed.Entry, fp *gofeed.Parser, wg *sync.WaitGroup) {

	log.Printf("🚩 run parser: delay: %v, random delay: %v, url: %v", p.Delay, p.RandomDelay, p.Link.Url)

	defer wg.Done()

	for {
		select {
		case <-context.Background().Done():
			break
		default:
		}

		log.Printf("started parser for given url: %v", p.Link.Url)
		entries := p.getEntries(fp)
		log.Printf("fetched the contents of a given url %v", p.Link.Url)

		for _, entry := range entries {
			ch <- entry
		}

		// Ожидаем установленное время до следующией итерации парсинга
		randomDelay := time.Duration(0)
		if p.RandomDelay != 0 {
			randomDelay = time.Duration(rand.Int63n(int64(p.RandomDelay)))
		}
		time.Sleep(p.Delay + randomDelay)
	}
}

// Получает записи ленты, если сайт kremlin,
// то запускает kremlin парсер,
// иначе запускает gofeed.Parser
func (p *Parser) getEntries(fp *gofeed.Parser) []feed.Entry {

	var entries []feed.Entry
	if p.Link.ResourceID == 1 {
		entries = append(entries, p.parseKremlin(p.Link.Url)...)
	}
	// Если ресурс ID равен 3, то запускаем новый парсер для mil.ru
	if p.Link.ResourceID == 3 {
		entries = append(entries, p.parseMil(p.Link.Url)...)
	} else {
		var gf *gofeed.Feed
		var err error
		for i := 0; i < 10; i++ {
			gf, err = fp.ParseURL(p.Link.Url)
			if err == nil {
				// Увеличиваем счетчик успешных запросов
				p.metrics.SuccessRequests.WithLabelValues(p.Link.Url, fmt.Sprintf("%d", i+1)).Inc()
				break
			}
			// Увеличиваем счетчик ошибок с номером попытки
			p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), fmt.Sprintf("%d", i+1)).Inc()

			log.Printf("Attempt %d: ERROR: %v, %v", i+1, err, p.Link.Url)
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			// Увеличиваем счетчик ошибок с номером попытки
			p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), fmt.Sprintf("%d", 10)).Inc()

			log.Printf("Failed after 10 attempts: %v, %v", err, p.Link.Url)
			return nil
		}
		entries = append(entries, feed.MakeEntries(gf.Items, p.Link)...)
	}
	return entries
}

func (p *Parser) parseKremlin(url string) []feed.Entry {

	node, err := htmlnode.GetTopicBody(url, p.Link.UserAgent)
	if os.IsTimeout(err) {
		// Увеличиваем счетчик ошибок
		p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), "0").Inc()
		log.Printf("server timeout error %v", err)
		return nil
	}
	if err != nil {
		// Увеличиваем счетчик ошибок
		p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), "0").Inc()
		log.Printf("failed to decode request body %v", sl.Err(err))
		return nil
	}

	// Увеличиваем счетчик успешных запросов
	p.metrics.SuccessRequests.WithLabelValues(p.Link.Url, "0").Inc()
	return p.parseEntries(node)
}
