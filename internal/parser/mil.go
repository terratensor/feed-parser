package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/terratensor/feed-parser/internal/entities/feed"
	"golang.org/x/net/html"
)

type ResponseData struct {
	Data []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Text    string `json:"text"`
		Link    string `json:"link"`
		Date    string `json:"date"`
		Preview string `json:"preview"`
	} `json:"data"`
}

// cleanHTML обрабатывает HTML, оставляя только содержимое тегов p (без атрибутов и вложенных тегов)
func cleanHTML(input string) string {
	var buf bytes.Buffer
	tokenizer := html.NewTokenizer(strings.NewReader(input))

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return buf.String()

		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, _ := tokenizer.TagName()
			if string(tagName) == "p" {
				// Начало <p>, начинаем читать содержимое
				var pContent strings.Builder

				readContent(tokenizer, &pContent)

				// Очищаем текст от лишних пробелов и переносов
				cleanText := normalizeWhitespace(pContent.String())

				// Пишем результат как простой <p> с очищенным текстом
				buf.WriteString("<p>")
				buf.WriteString(cleanText)
				buf.WriteString("</p>\n")
			}

		// Игнорируем остальные токены
		case html.TextToken, html.EndTagToken, html.CommentToken, html.DoctypeToken:
			// Пропускаем
		}
	}
}

// readContent рекурсивно считывает текстовое содержимое до закрывающего </p>
func readContent(tokenizer *html.Tokenizer, builder *strings.Builder) {
Loop:
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.TextToken:
			text := tokenizer.Text()
			builder.Write(text)
		case html.EndTagToken:
			tagName, _ := tokenizer.TagName()
			if string(tagName) == "p" {
				break Loop
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			// Пропускаем любой вложенный тег — просто переходим к следующему токену
			continue
		case html.ErrorToken:
			break Loop
		}
	}
}

// normalizeWhitespace убирает лишние пробелы и переносы строк
func normalizeWhitespace(s string) string {
	// Заменяем любые пробельные символы на одиночный пробел
	space := regexp.MustCompile(`\s+`)
	s = space.ReplaceAllString(s, " ")

	// Обрезаем пробелы по краям
	return strings.TrimSpace(s)
}

func (p *Parser) parseMil(url string) []feed.Entry {
	var resp *http.Response
	var body []byte
	var err error

	// Повторяем до 10 раз
	for attempt := 1; attempt <= 10; attempt++ {
		// Создаем новый HTTP-запрос
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), "0").Inc()
			log.Printf("failed to create request: %v", err)
			continue
		}

		// Устанавливаем User-Agent
		req.Header.Set("User-Agent", p.Link.UserAgent)

		// Выполняем запрос
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), "0").Inc()
			log.Printf("attempt %d: failed to fetch URL: %v", attempt, err)

			// Если это последняя попытка — пишем финальный лог
			if attempt == 10 {
				log.Printf("Failed after 10 attempts: %v, %v", err, p.Link.Url)
			}

			// Ждём перед повторной попыткой
			time.Sleep(1*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond)
			continue
		}

		// Читаем тело ответа
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), "0").Inc()
			log.Printf("attempt %d: failed to read response body: %v", attempt, err)

			if attempt == 10 {
				log.Printf("Failed after 10 attempts reading body: %v, %v", err, p.Link.Url)
			}

			time.Sleep(1*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond)
			continue
		}

		// Если всё прошло успешно — выходим из цикла
		break
	}

	// После 10 неудачных попыток resp может быть nil
	if body == nil {
		return nil
	}

	var response ResponseData
	err = json.Unmarshal(body, &response)
	if err != nil {
		p.metrics.ErrorRequests.WithLabelValues(p.Link.Url, err.Error(), "0").Inc()
		log.Printf("failed to unmarshal JSON: %v", err)
		return nil
	}

	var entries []feed.Entry

	for _, item := range response.Data {
		var publishedTime *time.Time
		t, err := time.Parse(time.RFC3339, item.Date)
		// Если не удалось распарсить дату, используем nil
		if err != nil {
			p.metrics.ErrorRequests.WithLabelValues(item.ID, err.Error(), "0").Inc()
			publishedTime = nil
		} else {
			publishedTime = &t
		}

		author := "Министерство обороны Российской Федерации"
		cleanedContent := cleanHTML(item.Text)

		entry := feed.Entry{
			Title:      item.Title,
			Url:        fmt.Sprintf("https://mil.ru/news/%s", item.ID),
			Content:    cleanedContent,
			Summary:    item.Preview,
			Published:  publishedTime,
			Language:   p.Link.Lang,
			ResourceID: p.Link.ResourceID,
			Author:     author,
		}

		entries = append(entries, entry)
	}

	// Увеличиваем счетчик успешных запросов
	p.metrics.SuccessRequests.WithLabelValues(p.Link.Url, "0").Inc()
	return entries
}
