package feed

import (
	"context"
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/internal/model/link"
	"log"
	"strings"
	"time"
)

type Entry struct {
	ID         *int64     `json:"id"`
	Language   string     `json:"language"`
	Title      string     `json:"title"`
	Url        string     `json:"url"`
	Updated    *time.Time `json:"updated"`
	Published  *time.Time `json:"published"`
	Created    *time.Time `json:"created"`
	UpdatedAt  *time.Time `json:"updated_at"`
	Summary    string     `json:"summary"`
	Content    string     `json:"content"`
	Author     string     `json:"author"`
	Number     string     `json:"number"`
	ResourceID int        `json:"resource_id"`
	Chunk      int        `json:"chunk"`
}

type StorageInterface interface {
	FindByUrl(ctx context.Context, url string) (*Entry, error)
	FindAllByUrl(ctx context.Context, url string) ([]Entry, error)
	Insert(ctx context.Context, entry *Entry) (*int64, error)
	Update(ctx context.Context, entry *Entry) error
	Bulk(ctx context.Context, entries *[]Entry) error
	FindAll(ctx context.Context, limit int) (chan Entry, error)
	FindDuration(ctx context.Context, duration time.Duration) (chan string, error)
	CalculateLimitCount(duration time.Duration) int
	Delete(ctx context.Context, id *int64) error
}

type Entries struct {
	Storage StorageInterface
}

func NewFeedStorage(store StorageInterface) *Entries {
	return &Entries{
		Storage: store,
	}
}

func (es *Entries) FindAll(ctx context.Context, limit int) (chan Entry, error) {

	chin, err := es.Storage.FindAll(ctx, limit)
	if err != nil {
		return nil, err
	}

	chout := make(chan Entry, 100)
	go func() {
		defer close(chout)
		for {
			select {
			case <-ctx.Done():
				return
			case l, ok := <-chin:
				if !ok {
					return
				}
				chout <- l
			}
		}
	}()
	return chout, nil
}

func (es *Entries) Find(ctx context.Context, duration time.Duration) (chan Entry, error) {
	chin, err := es.Storage.FindDuration(ctx, duration)
	if err != nil {
		return nil, err
	}

	chout := make(chan Entry, 100)
	go func() {
		defer close(chout)
		for {
			select {
			case <-ctx.Done():
				return
			case l, ok := <-chin:
				if !ok {
					return
				}

				chunks, err := es.Storage.FindAllByUrl(ctx, l)
				if err != nil {
					log.Printf("failed to find all entries: %v", err)
					return
				}

				var builder strings.Builder

				for _, chunk := range chunks {
					builder.WriteString(chunk.Content)
				}

				entry := Entry{
					Language:   chunks[0].Language,
					Title:      chunks[0].Title,
					Url:        chunks[0].Url,
					Updated:    chunks[0].Updated,
					Published:  chunks[0].Published,
					Created:    chunks[0].Created,
					UpdatedAt:  chunks[0].UpdatedAt,
					Summary:    chunks[0].Summary,
					Content:    builder.String(),
					Author:     chunks[0].Author,
					Number:     chunks[0].Number,
					ResourceID: chunks[0].ResourceID,
				}

				chout <- entry
			}
		}
	}()
	return chout, nil
}

func MakeEntries(items []*gofeed.Item, url link.Link) []Entry {
	var entries []Entry

	for _, item := range items {

		content := populateContentField(item, url)

		e := &Entry{
			Language:   url.Lang,
			Title:      item.Title,
			Url:        item.Link,
			Updated:    item.UpdatedParsed,
			Published:  item.PublishedParsed,
			Summary:    item.Description,
			Content:    content,
			ResourceID: url.ResourceID,
		}

		entries = append(entries, *e)
	}

	return entries
}

func populateContentField(item *gofeed.Item, url link.Link) string {

	var content string
	content = item.Content

	//if url.ResourceID == 4 {
	//	feedExt := item.Extensions["yandex"]["full-text"][0]
	//	content = striphtml.StripHtmlTags(feedExt.Value)
	//
	//	var sb strings.Builder
	//	sentences := strings.SplitAfter(content, "\n")
	//	for _, con := range sentences {
	//		if len(con) > 0 {
	//			sb.WriteString("<p>")
	//			sb.WriteString(con)
	//			sb.WriteString("</p>")
	//		}
	//	}
	//
	//	content = sb.String()
	//}

	return content
}
