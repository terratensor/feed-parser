package parser

import (
	"fmt"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"golang.org/x/net/html"
	"regexp"
	"time"
)

func (p *Parser) parseEntries(n *html.Node) []feed.Entry {

	var entries []feed.Entry
	var f func(*html.Node)

	e := feed.Entry{}

	f = func(n *html.Node) {

		if n.Type == html.ElementNode && n.Data == "entry" {
			for cl := n.FirstChild; cl != nil; cl = cl.NextSibling {

				if cl.Type == html.ElementNode && cl.Data == "title" {
					e.Title = getInnerText(cl)
				}

				if cl.Type == html.ElementNode && cl.Data == "id" {
					e.Url = getInnerText(cl)
				}

				if cl.Type == html.ElementNode && cl.Data == "updated" {

					t, err := time.Parse("2006-01-02T15:04:05-07:00", getInnerText(cl))
					if err != nil {
						fmt.Println(err)
						return
					}
					e.Updated = &t
				}

				if cl.Type == html.ElementNode && cl.Data == "published" {
					t, err := time.Parse("2006-01-02T15:04:05-07:00", getInnerText(cl))
					if err != nil {
						fmt.Println(err)
						return
					}
					e.Published = &t
				}

				if cl.Type == html.ElementNode && cl.Data == "summary" {
					e.Summary = deleteImgTag(getInnerText(cl))
				}

				if cl.Type == html.ElementNode && cl.Data == "content" {
					e.Content = deleteImgTag(getInnerText(cl))
				}

			}

			e.Language = p.Link.Lang
			e.ResourceID = p.Link.ResourceID
			entries = append(entries, e) //fmt.Println(entry)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)

	return entries
}

// getInnerText returns the inner text of the HTML node.
//
// It takes a pointer to a html.Node as a parameter and returns a string.
func getInnerText(node *html.Node) string {
	for el := node.FirstChild; el != nil; el = el.NextSibling {
		if el.Type == html.TextNode {
			return el.Data
		}
	}
	return ""
}

// nodeHasRequiredRelAttr checks if the given html.Node has the required rel attribute.
//
// It takes a string rcc and a *html.Node n as parameters and returns a boolean.
func nodeHasRequiredRelAttr(rcc string, n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "rel" && attr.Val == rcc {
			return true
		}
	}
	return false
}

// getRequiredDataAttr returns the value of the specified attribute from the given html.Node.
//
// rda string - the attribute key to search for.
// n *html.Node - the html node to search within.
// string - the value of the specified attribute, or an empty string if not found.
func getRequiredDataAttr(rda string, n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == rda {
			return attr.Val
		}
	}
	return ""
}

// deleteImgTag очищает строку от тега изображения
func deleteImgTag(str string) string {

	re := regexp.MustCompile(`<img[^>]*>`)

	// Remove the img tag
	htmlWithoutImg := re.ReplaceAllString(str, "")

	return htmlWithoutImg
}
