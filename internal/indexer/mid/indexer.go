package mid

import (
	"context"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/model/link"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Indexer struct {
	Link        link.Link
	Delay       time.Duration
	RandomDelay time.Duration
}

// NewIndexer creates a new Indexer with the given URL, delay, and randomDelay.
func NewIndexer(url link.Link, delay time.Duration, randomDelay time.Duration) *Indexer {
	np := &Indexer{
		Link:        url,
		Delay:       delay,
		RandomDelay: randomDelay,
	}
	return np
}

func (i *Indexer) Run(ch chan feed.Entry, wg *sync.WaitGroup) {
	defer wg.Done()
	var exitCount int
	for {

		randomDelay := time.Duration(0)
		if i.RandomDelay != 0 {
			randomDelay = time.Duration(rand.Int63n(int64(i.RandomDelay)))
		}
		time.Sleep(i.Delay + randomDelay)

		log.Printf("🚩 started parser for given url: %v", i.Link.Url)

		entries, err := i.parseAnnounceItems(i.Link)
		if err != nil {
			log.Printf("cannot parse url: %v", err)
			continue
		}

		log.Printf("✅ fetched the contents of a given url %v", i.Link.Url)

		newUrl, err := url.Parse(i.Link.Url)
		if err != nil {
			log.Printf("cannot parse url: %v", err)
			continue
		}

		values := newUrl.Query()
		f := values.Get("PAGEN_1")
		num, err := strconv.Atoi(f)
		if err != nil {
			log.Printf("cannot parse url PAGEN_1 param: %v", err)
			continue
		}
		values.Set("PAGEN_1", strconv.Itoa(num+1))
		newUrl.RawQuery = values.Encode()

		i.Link.Url = newUrl.String()

		select {
		case <-context.Background().Done():
			break
		default:
		}

		// если массив пустой, следующей станицы не существует
		if len(entries) == 0 {
			exitCount++
		}

		for _, entry := range entries {
			ch <- entry
		}

		// набираем счетчик до 5 и завершаем парсер
		if exitCount > 5 {
			break
		}
	}
}
