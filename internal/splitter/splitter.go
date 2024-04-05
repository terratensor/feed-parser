package splitter

import (
	"context"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"log"
	"strings"
	"unicode/utf8"
)

type Splitter struct {
	optParSize int
	maxParSize int
}

func NewSplitter(optParSize int, maxParSize int) *Splitter {
	return &Splitter{
		optParSize: optParSize,
		maxParSize: maxParSize,
	}
}

func (sp *Splitter) SplitEntry(ctx context.Context, entry feed.Entry) []feed.Entry {

	var entries []feed.Entry

	var contentBuilder strings.Builder
	contentBuilder.WriteString(entry.Content)

	contentChunks := sp.splitContent(&contentBuilder)

	for chunk, content := range contentChunks {

		newEntry := feed.Entry{
			Language:   entry.Language,
			Title:      entry.Title,
			Url:        entry.Url,
			Updated:    entry.Updated,
			Published:  entry.Published,
			Summary:    entry.Summary,
			Content:    content,
			Author:     entry.Author,
			Number:     entry.Number,
			ResourceID: entry.ResourceID,
			Created:    entry.Created,
			Chunk:      chunk + 1,
		}

		entries = append(entries, newEntry)
	}
	return entries
}

func (sp *Splitter) splitContent(contentBuilder *strings.Builder) []string {

	entryLen := utf8.RuneCountInString(contentBuilder.String())
	var builder strings.Builder

	var pars []string

	if entryLen > sp.maxParSize {
		log.Printf("🚩 Обрабатываем большой контент, делим на фрагменты: %v", utf8.RuneCountInString(contentBuilder.String()))

		paragraphs := strings.SplitAfter(contentBuilder.String(), "</p>")

		for _, paragraph := range paragraphs {

			builder.WriteString(paragraph)

			if utf8.RuneCountInString(builder.String()) > sp.optParSize {
				pars = append(pars, builder.String())
				//log.Printf("🚩 полученный chunk: %v", utf8.RuneCountInString(builder.String()))
				builder.Reset()
			}
		}

		if utf8.RuneCountInString(builder.String()) > 0 {
			//log.Printf("🚩🚩 полученный остаток chunk: %v", utf8.RuneCountInString(builder.String()))
			//log.Printf("(builder.String()) %v", builder.String())
			lastPar := pars[len(pars)-1]
			pars = pars[:len(pars)-1]
			newLastPar := lastPar + builder.String()
			//log.Printf("🚩🚩🚩🚩 %v", utf8.RuneCountInString(newLastPar))
			pars = append(pars, newLastPar)
		}

		if len(pars) == 1 && utf8.RuneCountInString(pars[0]) > sp.maxParSize+sp.optParSize {
			log.Printf("🚩🚩 long paragraph %v", pars[0])
			var longBuilder strings.Builder
			content := pars[0]
			longBuilder.WriteString(content)
			pars = sp.splitLongParagraph(&longBuilder)
		}
	} else {
		pars = append(pars, contentBuilder.String())
		builder.Reset()
	}

	count := len(pars)
	if count > 1 {
		log.Printf("🚩 итого количество фрагментов в параграфе: %v", count)
	}

	return pars
}

func (sp *Splitter) splitLongParagraph(longBuilder *strings.Builder) []string {

	count := utf8.RuneCountInString(longBuilder.String())
	log.Printf("🚩🚩🚩🚩 обрабатываем длинный фрагмент: %v\n", count)

	var pars []string

	var builder strings.Builder
	result := longBuilder.String()
	result = strings.TrimPrefix(result, "<div>")
	//result = strings.TrimPrefix(result, "<p>")
	result = strings.TrimSuffix(result, "</div>")
	//result = strings.TrimSuffix(result, "</p>")

	// sentences []string Делим параграф на предложения, разделитель точка
	sentences := strings.SplitAfter(result, ".")

	longBuilder.Reset()

	for _, sentence := range sentences {

		sentence = strings.TrimSpace(sentence)

		if (utf8.RuneCountInString(builder.String()) + utf8.RuneCountInString(sentence)) < sp.optParSize {
			builder.WriteString(sentence)
			builder.WriteString(" ")
		} else {
			longBuilder.WriteString("<p>")
			longBuilder.WriteString(strings.TrimSpace(builder.String()))
			longBuilder.WriteString("</p>")
			pars = append(pars, longBuilder.String())
			builder.Reset()
			longBuilder.Reset()
		}
	}

	if utf8.RuneCountInString(builder.String()) > 0 {
		longBuilder.WriteString("<p>")
		longBuilder.WriteString(strings.TrimSpace(builder.String()))
		longBuilder.WriteString("</p>")

		pars = append(pars, longBuilder.String())
		builder.Reset()
		longBuilder.Reset()
	}

	return pars
}
