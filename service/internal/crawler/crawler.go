package crawler

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/terratensor/feed-parser/service/internal/entities/feed"
	"log"
	"math/rand"
	"strings"
	"time"
)

func VisitMil(entry *feed.Entry) *feed.Entry {
	c := colly.NewCollector()

	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	// iterating over the list of industry card

	// HTML elements
	c.Limit(&colly.LimitRule{
		RandomDelay: 20 * time.Second,
	})

	c.OnHTML("#center", func(e *colly.HTMLElement) {
		log.Printf("Crawling Url %#v", entry.Url)
		title := e.ChildText("h1")

		// filter out unwanted data
		content := e.ChildTexts("p")

		author := e.ChildText("div a.date")

		var sb strings.Builder
		for _, con := range content {
			if len(con) > 0 {
				sb.WriteString("<p>")
				sb.WriteString(con)
				sb.WriteString("</p>")
			}
		}

		entry.Title = title
		entry.Content = sb.String()
		entry.Author = author

		log.Printf("colly: %v", entry)

	})

	n := 1 + rand.Intn(10)
	d := time.Duration(n)
	time.Sleep(d * time.Second)

	//c.Visit("https://function.mil.ru:443/news_page/country/more.htm?id=12502939@egNews")
	err := c.Visit(entry.Url)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
	}

	return entry
}

func VisitMid(entry *feed.Entry) *feed.Entry {
	c := colly.NewCollector()

	//c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	// iterating over the list of industry card

	// HTML elements
	//c.Limit(&colly.LimitRule{
	//	RandomDelay: 20 * time.Second,
	//})

	c.OnHTML("div.page-body", func(e *colly.HTMLElement) {
		log.Printf("Crawling Url %#v", entry.Url)

		title := e.ChildText("h1.photo-content__title")

		number := e.ChildText("p.article-line__note.article-line__note_small")

		// filter out unwanted data
		content := e.ChildText("div.text.article-content")

		entry.Title = title
		entry.Content = content
		entry.Number = number

		log.Printf("colly: %v", entry)

	})

	n := 1 + rand.Intn(10)
	d := time.Duration(n)
	time.Sleep(d * time.Second)

	//c.Visit("https://function.mil.ru:443/news_page/country/more.htm?id=12502939@egNews")
	err := c.Visit(entry.Url)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
	}

	return entry
}