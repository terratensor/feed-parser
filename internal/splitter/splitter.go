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

	contentChunks := sp.splitContent(entry.Content)

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

func (sp *Splitter) splitContent(entryContent string) []string {

	var contentBuilder strings.Builder
	contentBuilder.WriteString(entryContent)

	var builder strings.Builder
	var pars []string

	entryContentLen := utf8.RuneCountInString(contentBuilder.String())
	if entryContentLen > sp.maxParSize {

		// Заменяем строковый разделитель \n на обычный
		content := strings.Replace(contentBuilder.String(), `\n`, "\n", -1)
		contentBuilder.Reset()
		contentBuilder.WriteString(content)

		// Задаем разделители
		separators := []string{"\n", "<br>"}
		// Обрабатываем стандартный сценарий, делим по параграфам
		paragraphs := strings.SplitAfter(contentBuilder.String(), "</p>")

		var completed bool

		for _, paragraph := range paragraphs {
			completed = false
			// Если полученный параграф больше оптимального размера, то разбиваем его на части
			// по ранее заданному списку разделителей
			if utf8.RuneCountInString(paragraph) > sp.maxParSize {
				for _, separator := range separators {

					newParagraphs := strings.SplitAfter(paragraph, separator)
					// если после использованного разделителя получили 1 параграф,
					// то пропускаем и применяем следующий разделитель
					if len(newParagraphs) == 1 {
						continue
					}
					log.Printf("🚩🚩 Успешно обработано  по разделителю `%v`", separator)
					log.Printf("кол-во параграфов %v, %v", len(newParagraphs), newParagraphs)
					pars = append(pars, sp.processNewParagraphs(newParagraphs, &builder, separator)...)
					completed = true
					break
				}
			} else {
				builder.WriteString(paragraph)
			}

			// если проверки сверху не сработали записываем параграф как есть
			if utf8.RuneCountInString(paragraph) > sp.maxParSize && !completed {
				log.Printf("оставляем длинный параграф как есть: %v, %v", utf8.RuneCountInString(paragraph), paragraph)
				builder.WriteString(paragraph)
			}

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
			log.Printf("🚩🚩 запускаем разделение по предложениям %v", pars[0])
			var longBuilder strings.Builder
			longBuilder.WriteString(pars[0])
			pars = sp.splitLongParagraph(&longBuilder)
		}
	} else {
		pars = append(pars, contentBuilder.String())
		builder.Reset()
		contentBuilder.Reset()
	}

	count := len(pars)
	//if count > 1 {
	log.Printf("🚩 итого количество фрагментов в параграфе: %v", count)
	//}

	return pars
}

func (sp *Splitter) processNewParagraphs(newParagraphs []string, builder *strings.Builder, separator string) []string {
	var result []string
	for _, newParagraph := range newParagraphs {

		builder.WriteString("<p>")

		cleanParagraph := strings.Replace(newParagraph, separator, "", -1)
		builder.WriteString(cleanParagraph)

		builder.WriteString("</p>")

		if utf8.RuneCountInString(builder.String()) > sp.optParSize {
			result = append(result, builder.String())
			builder.Reset()
		}
	}

	return result
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

func trimPrefixes(longBuilder strings.Builder) {
	result := longBuilder.String()
	longBuilder.Reset()

	result = strings.TrimPrefix(result, "<div>")
	result = strings.TrimSuffix(result, "</div>")

	result = strings.TrimPrefix(result, "<p>")
	result = strings.TrimSuffix(result, "</p>")

	longBuilder.WriteString(result)
}
