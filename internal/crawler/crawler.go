package crawler

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"log"
	"math/rand"
	"strings"
	"time"
)

func VisitMil(entry *feed.Entry) (*feed.Entry, error) {
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

		log.Printf("colly: %v", entry.Title)

	})

	n := 1 + rand.Intn(10)
	d := time.Duration(n)
	time.Sleep(d * time.Second)

	//c.Visit("https://function.mil.ru:443/news_page/country/more.htm?id=12502939@egNews")
	err := c.Visit(entry.Url)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
		return nil, err
	}

	return entry, nil
}

func VisitMid(entry *feed.Entry) (*feed.Entry, error) {

	c := colly.NewCollector()
	c.AllowURLRevisit = false

	c.UserAgent = "PostmanRuntime/7.37.0"
	//c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	// iterating over the list of industry card

	// HTML elements
	c.Limit(&colly.LimitRule{
		RandomDelay: 10 * time.Second,
	})

	c.OnHTML("div.photo-content", func(e *colly.HTMLElement) {
		log.Printf("Crawling Url %#v", entry.Url)

		title := e.ChildText("h1.photo-content__title")

		number := e.ChildText("p.article-line__note.article-line__note_small")

		// filter out unwanted data
		content := e.ChildTexts("div.text.article-content p")

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
		entry.Number = number

		log.Printf("Mid photo-content: %v", entry.Title)
	})

	// Если опубликовано как анонс
	c.OnHTML("ul.announcements", func(e *colly.HTMLElement) {
		log.Printf("Crawling Url %#v", entry.Url)

		title := e.ChildText("h3.announcement__title")

		number := e.ChildText("div.announcement__doc-num")

		// filter out unwanted data
		content := e.ChildTexts("div.announcement__text > p")

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
		entry.Number = number

		log.Printf("Mid announcements: %v", entry.Title)
	})

	count := 0
	for {
		// ожидаем после запроса рандомно 10 - 30 секунд
		// с увеличением паузы после неудачной попытки
		var n int
		if c.AllowURLRevisit {
			n = (1+count)*10 + rand.Intn(30)
		} else {
			n = (1 + count) * 10
		}
		d := time.Duration(n)
		time.Sleep(d * time.Second)

		// Посещает ссылку, если ошибка, обычно connection reset by peer,
		// то повторяет попытку и увеличивает счетчик попыток,
		// пытается получить контент 10 раз
		err := c.Visit(entry.Url)
		if err != nil {
			count++
			log.Printf("Crawler Error: %v", err)
			log.Printf("🔄 try again: %v url: %v", count, entry.Url)
			if c.AllowURLRevisit && count <= 10 {
				continue
			}
			return nil, err
		}
		break
	}

	return entry, nil
}
