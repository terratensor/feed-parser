package mil

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"log"
	"math/rand"
	"strings"
	"time"
)

func (i *Indexer) parseNewsItems(rawURL string) ([]feed.Entry, error) {

	var entries []feed.Entry
	entry := feed.Entry{}

	c := colly.NewCollector()

	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

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

	n := 2 + rand.Intn(2)
	d := time.Duration(n)
	time.Sleep(d * time.Second)

	//c.Visit("https://function.mil.ru:443/news_page/country/more.htm?id=12502939@egNews")
	err := c.Visit(rawURL)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
		return nil, err
	}

	return entries, nil
}
