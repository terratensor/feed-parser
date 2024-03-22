package kremlin

import (
	"context"
	"github.com/mmcdole/gofeed"
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

type Indexer struct {
	Link        link.Link
	Delay       time.Duration
	RandomDelay time.Duration
	Meta        *Meta
}

// NewIndexer creates a new Indexer with the given URL, delay, and randomDelay.
func NewIndexer(url link.Link, delay time.Duration, randomDelay time.Duration) *Indexer {
	np := &Indexer{
		Link:        url,
		Delay:       delay,
		RandomDelay: randomDelay,
		Meta:        NewMeta(),
	}
	return np
}

func (i *Indexer) Run(ch chan feed.Entry, fp *gofeed.Parser, wg *sync.WaitGroup) {
	defer wg.Done()

	for {

		randomDelay := time.Duration(0)
		if i.RandomDelay != 0 {
			randomDelay = time.Duration(rand.Int63n(int64(i.RandomDelay)))
		}
		time.Sleep(i.Delay + randomDelay)

		url := i.getUrl()
		parsedLink := link.NewLink(url, i.Link.Lang, i.Link.ResourceID)
		// Если url пустой, следующей достигнут конец RSS ленты,
		// следующей станицы не существует, заканчиваем парсинг
		if parsedLink.Url == "" {
			break
		}

		log.Printf("started parser for given url: %v", parsedLink.Url)
		//gf, err := fp.ParseURL(parsedLink.Url)
		//
		//if err != nil {
		//	log.Printf("ERROR: %v, %v", err, parsedLink.Url)
		//	continue
		//}
		//entries := feed.MakeEntries(gf.Items, *parsedLink)

		// Парсим объект мета со ссылками на следующую станицу
		node, err := htmlnode.GetTopicBody(url)
		if os.IsTimeout(err) {
			log.Printf("server timeout error %v", err)
			continue
		}
		if err != nil {
			log.Printf("failed to decode request body %v", sl.Err(err))
			continue
		}

		i.parseMeta(node)

		log.Printf("fetched the contents of a given url %v", parsedLink.Url)
		entries := i.parseEntries(node)

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

func (i *Indexer) getUrl() string {
	var url string
	// Если Meta только инициализирован, то Meta.Self и Meta.Next пусты,
	// устанавливаем то url равен начальному url,
	// иначе url равен ссылке на следующую страницу
	if i.Meta.Self == "" && i.Meta.Next == "" {
		url = i.Link.Url
	} else {
		url = i.Meta.Next
	}
	return url
}
