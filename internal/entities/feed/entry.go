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

		e := &Entry{
			Language:   url.Lang,
			Title:      item.Title,
			Url:        item.Link,
			Updated:    item.UpdatedParsed,
			Published:  item.PublishedParsed,
			Summary:    item.Description,
			Content:    item.Content,
			ResourceID: url.ResourceID,
		}

		entries = append(entries, *e)
	}

	return entries
}
