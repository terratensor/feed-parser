package crawler

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/terratensor/feed-parser/internal/config"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/metrics"
)

// VisitMil выполняет парсинг страницы с использованием конфигурации
func VisitMil(entry *feed.Entry, config *config.Crawler, metrics *metrics.Metrics) (*feed.Entry, error) {
	c := colly.NewCollector()

	c.AllowURLRevisit = true

	// Устанавливаем User-Agent из конфигурации
	c.UserAgent = config.UserAgent

	// Счетчик попыток
	retryCount := 0

	// Обработчик ошибок
	c.OnError(func(r *colly.Response, err error) {
		if retryCount < config.MaxRetries {
			retryCount++
			log.Printf("Error: %v. Retrying (%d/%d) in %v...", err, retryCount, config.MaxRetries, config.RetryDelay)
			// Увеличиваем счетчик ошибок с номером попытки
			metrics.ErrorRequests.WithLabelValues(r.Request.URL.String(), err.Error(), fmt.Sprintf("%d", retryCount)).Inc()
			time.Sleep(config.RetryDelay)
			r.Request.Retry()
		} else {
			log.Printf("Error: %v. Max retries reached (%d).", err, config.MaxRetries)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	// Устанавливаем лимит задержки из конфигурации
	c.Limit(&colly.LimitRule{
		RandomDelay: time.Duration(config.RandomDelayMin+rand.Intn(config.RandomDelayMax-config.RandomDelayMin)) * time.Second,
	})

	c.OnHTML("#center", func(e *colly.HTMLElement) {
		log.Printf("Crawling Url %#v", entry.Url)
		title := e.ChildText("h1")

		// Фильтруем ненужные данные
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

		log.Printf("Crawling Title: %v", entry.Title)
	})

	// Генерируем случайную задержку сна из конфигурации
	n := config.SleepMin + rand.Intn(config.SleepMax-config.SleepMin)
	d := time.Duration(n)
	time.Sleep(d * time.Second)

	// Посещаем URL
	err := c.Visit(entry.Url)
	if err != nil {
		log.Printf("Crawler Error: %v", err)
		// Увеличиваем счетчик ошибок с номером попытки
		metrics.ErrorRequests.WithLabelValues(entry.Url, err.Error(), fmt.Sprintf("%d", retryCount)).Inc()
		return nil, err
	}

	// Увеличиваем счетчик успешных запросов
	metrics.SuccessRequests.WithLabelValues(entry.Url, fmt.Sprintf("%d", retryCount)).Inc()

	return entry, nil
}

func VisitMid(entry *feed.Entry, config *config.Crawler, metrics *metrics.Metrics) (*feed.Entry, error) {

	c := colly.NewCollector()
	c.AllowURLRevisit = false

	c.UserAgent = ""
	//c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Crawl on Page", r.URL.String())
	})

	// iterating over the list of industry card

	// HTML elements
	c.Limit(&colly.LimitRule{
		Delay:       5 * time.Second,
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
			n = ((1 + count) * 10) + rand.Intn(30)
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

			if c.AllowURLRevisit && count <= config.MaxRetries {
				// Увеличиваем счетчик ошибок с номером попытки
				metrics.ErrorRequests.WithLabelValues(entry.Url, err.Error(), fmt.Sprintf("%d", count)).Inc()
				log.Printf("🔄 try again: %v url: %v", count, entry.Url)
				continue
			}

			// Увеличиваем счетчик ошибок с номером попытки
			metrics.ErrorRequests.WithLabelValues(entry.Url, err.Error(), fmt.Sprintf("%d", count)).Inc()

			return nil, err
		}
		break
	}

	// Увеличиваем счетчик успешных запросов
	metrics.SuccessRequests.WithLabelValues(entry.Url, fmt.Sprintf("%d", count)).Inc()

	return entry, nil
}
