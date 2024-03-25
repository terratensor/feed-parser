package feed

import (
	"context"
	"github.com/mmcdole/gofeed"
	"github.com/terratensor/feed-parser/internal/model/link"
	"time"
)

type Entry struct {
	ID         *int64     `json:"id"`
	Language   string     `json:"language"`
	Title      string     `json:"title"`
	Url        string     `json:"url"`
	Updated    *time.Time `json:"updated"`
	Published  *time.Time `json:"published"`
	Summary    string     `json:"summary"`
	Content    string     `json:"content"`
	Author     string     `json:"author"`
	Number     string     `json:"number"`
	ResourceID int        `json:"resource_id"`
	Created    *time.Time `json:"created"`
}

type StorageInterface interface {
	FindByUrl(ctx context.Context, url string) (*Entry, error)
	Insert(ctx context.Context, entry *Entry) (*int64, error)
	Update(ctx context.Context, entry *Entry) error
	Bulk(ctx context.Context, entries *[]Entry) error
}

type Entries struct {
	Storage StorageInterface
}

func NewFeedStorage(store StorageInterface) *Entries {
	return &Entries{
		Storage: store,
	}
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
