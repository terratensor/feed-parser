package mil

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/terratensor/feed-parser/internal/entities/feed"
)

func (i *Indexer) parseNewsItems(rawURL string, userAgent string) ([]feed.Entry, error) {

	var entries []feed.Entry
	entry := feed.Entry{}

	c := colly.NewCollector()

	// Разрешить повторное посещение URL
	c.AllowURLRevisit = true

	c.UserAgent = userAgent

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	c.OnHTML("#center", func(e *colly.HTMLElement) {

		e.ForEach("div", func(_ int, e *colly.HTMLElement) {
			if e.Attr("class") == "newsitem" {

				// populate date
				date, err := time.Parse("02.01.2006 (15:04)", e.ChildText("span.date"))
				if err != nil {
					log.Printf("cannot parse date: %v", err)
					date = time.Time{}
				}

				entry.Published = &date
				//log.Printf("Crawling Date %#v", entry.Updated.Format("2006-01-02 15:04"))

				// populate url id
				urlValue := e.ChildAttr("a", "href")

				entry.Url = fmt.Sprintf("https://function.mil.ru:443/%v", urlValue)
				//log.Printf("Crawling Url %#v", entry.Url)

				// populate title
				entry.Title = e.ChildText("a")
				//log.Printf("Crawling Title %#v", entry.Title)

				// populate summary
				summary := e.Text

				var sb strings.Builder
				sentences := strings.SplitAfter(summary, "\n")
				for n, sum := range sentences {
					if n > 2 && len(sum) > 0 {
						sb.WriteString(strings.TrimSpace(sum))
					}
				}
				entry.Summary = sb.String()

				//log.Printf("Crawling Summary %#v", entry.Summary)

				entry.Language = i.Link.Lang
				entry.ResourceID = i.Link.ResourceID
				// append entry
				entries = append(entries, entry)
			}
		})
	})

	maxRetries := 10              // Максимальное количество попыток
	retryDelay := 5 * time.Second // Задержка между попытками

	for retry := 0; retry < maxRetries; retry++ {
		err := c.Visit(rawURL)
		if err == nil {
			// Успешное соединение, возвращаем результат
			return entries, nil
		}

		log.Printf("Crawler Error (attempt %d/%d): %v", retry+1, maxRetries, err)

		if retry < maxRetries-1 {
			// Если это не последняя попытка, делаем паузу перед повторной попыткой
			time.Sleep(retryDelay)
		}
	}

	// Если все попытки неудачны, возвращаем ошибку
	return nil, fmt.Errorf("failed to connect after %d attempts", maxRetries)
}
