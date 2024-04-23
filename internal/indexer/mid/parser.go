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

	if link.UserAgent != "" {
		c.UserAgent = link.UserAgent
	} else {
		c.UserAgent = "PostmanRuntime/7.37.0"
	}

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
					date, err = time.Parse("2 January 2006 15:04", datetime)
					if err != nil {
						log.Printf("cannot parse date: %v", err)
						date = time.Time{}
					}
				}

				entry.Published = &date

				// populate url id
				urlValue := e.ChildAttr("a", "href")
				var u = url.URL{
					Scheme: "http",
					Host:   "mid.ru",
					Path:   urlValue,
				}
				entry.Url = u.String()

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
