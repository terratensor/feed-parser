package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"log"
	"math/rand"
	"time"
)

type Industry struct {
	Title   string
	Content string
	Author  string
	Number  string
}

func main() {
	c := colly.NewCollector()

	//c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	//c.OnResponse(func(r *colly.Response) {
	//	fmt.Println("Visited", r.Body)
	//
	//	log.Println(string(r.Body))
	//})

	// iterating over the list of industry card

	// HTML elements

	c.OnHTML("div.page-body", func(e *colly.HTMLElement) {

		title := e.ChildText("h1.photo-content__title")

		number := e.ChildText("p.article-line__note.article-line__note_small")

		// filter out unwanted data
		content := e.ChildText("div.text.article-content")

		industry := Industry{

			Title:   title,
			Content: content,
			Number:  number,
		}

		log.Printf("industry: %v", industry)

	})

	//c.OnHTML("title", func(e *colly.HTMLElement) {
	//	fmt.Println(e.Text)
	//})

	n := 1 + rand.Intn(2)
	d := time.Duration(n)
	time.Sleep(d * time.Second)

	err := c.Visit("https://mid.ru/en/foreign_policy/news/1935743/?lang=ru")
	log.Printf("Crawler Error: %v", err)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
	}

	//c.Visit("https://mid.ru/en/foreign_policy/news/1935743/")
}
