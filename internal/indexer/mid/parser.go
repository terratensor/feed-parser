package mid

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/model/link"
	"log"
	"math/rand"
	"net/url"
	"time"
)

func (i *Indexer) parseAnnounceItems(link link.Link) ([]feed.Entry, error) {

	var entries []feed.Entry
	entry := feed.Entry{}

	c := colly.NewCollector()

	//c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	c.OnHTML("ul.announce.announce_articles", func(e *colly.HTMLElement) {

		e.ForEach("li", func(_ int, e *colly.HTMLElement) {
			if e.Attr("class") == "announce__item" {

				// populate date
				adate := e.ChildText("span.announce__date")
				atime := e.ChildText("span.announce__time")

				datetime := fmt.Sprintf("%s %s", adate, atime)
				date, err := time.Parse("02.01.2006 15:04", datetime)
				if err != nil {
					log.Printf("cannot parse date: %v", err)
					date = time.Time{}
				}

				entry.Published = &date
				//log.Printf("Crawling Date %#v", entry.Published.Format("2006-01-02 15:04"))

				// populate url id
				urlValue := e.ChildAttr("a", "href")
				var u = url.URL{
					Scheme: "http",
					Host:   "mid.ru",
					Path:   urlValue,
				}
				entry.Url = u.String()

				//log.Printf("Crawling Url %#v", entry.Url)

				// populate title
				entry.Title = e.ChildText("a")
				//log.Printf("Crawling Title %#v", entry.Title)

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
	err := c.Visit(link.Url)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
		return nil, err
	}

	return entries, nil
}
