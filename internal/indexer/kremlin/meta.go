package kremlin

import (
	"fmt"
	"golang.org/x/net/html"
	"time"
)

type Meta struct {
	Updated *time.Time `json:"updated"`
	ID      string     `json:"id"`
	Self    string     `json:"self"`
	Prev    string     `json:"prev"`
	First   string     `json:"first"`
	Next    string     `json:"next"`
	Last    string     `json:"last"`
}

// NewMeta инициализирует и возвращает новый объект Meta.
//
// Он не принимает параметров и возвращает указатель на объект Meta.
// Это объект, который содержит навигационную информацию о страницах ленты.
// Время последнего обновления страницы ленты.
// Адрес текущей страницы, адрес следующей станицы, адрес предыдущей и адрес последней и первой страницы в ленте.
// С помощью этой информации можно совершать обход ленты.
func NewMeta() *Meta {
	meta := Meta{}
	return &meta
}

// parseMeta парсит данный html.Node для извлечения мета-информации.
//
// Принимает указатель на html.Node в качестве параметра и возвращает указатель на структуру Meta.
func (i *Indexer) parseMeta(node *html.Node) {
	var f func(*html.Node)

	meta := Meta{}

	f = func(node *html.Node) {

		// Если у ноды есть родитель и этот родитель — тэг feed
		if node.Parent != nil && node.Parent.Data == "feed" {

			if node.Type == html.ElementNode && node.Data == "updated" {
				t, err := time.Parse("2006-01-02T15:04:05-07:00", getInnerText(node))
				if err != nil {
					fmt.Println(err)
					return
				}
				meta.Updated = &t
			}
			if node.Type == html.ElementNode && node.Data == "id" {
				meta.ID = getInnerText(node)
			}
			//
			if node.Type == html.ElementNode && node.Data == "link" {

				if nodeHasRequiredRelAttr("self", node) {
					meta.Self = getRequiredDataAttr("href", node)
				}
				if nodeHasRequiredRelAttr("prev", node) {
					meta.Prev = getRequiredDataAttr("href", node)
				}
				if nodeHasRequiredRelAttr("first", node) {
					meta.First = getRequiredDataAttr("href", node)
				}
				if nodeHasRequiredRelAttr("next", node) {
					meta.Next = getRequiredDataAttr("href", node)
				}
				if nodeHasRequiredRelAttr("last", node) {
					meta.Last = getRequiredDataAttr("href", node)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(node)

	i.Meta = &meta
}
