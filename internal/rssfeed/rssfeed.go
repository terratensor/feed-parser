package rssfeed

import (
	"encoding/xml"
	"io"
	"time"
)

type RssFeedXml struct {
	XMLName          xml.Name `xml:"rss"`
	Version          string   `xml:"version,attr"`
	ContentNamespace string   `xml:"xmlns:content,attr,omitempty"`
	YandexNamespace  string   `xml:"xmlns:yandex,attr"`
	MediaNamespace   string   `xml:"xmlns:media,attr"`
	Channel          *RssFeed
}

type RssFeed struct {
	XMLName     xml.Name   `xml:"channel"`
	Title       string     `xml:"title,omitempty"`
	Link        string     `xml:"link,omitempty"`
	Description string     `xml:"description,omitempty"`
	Items       []*RssItem `xml:"item"`
}

type RssItem struct {
	XMLName     xml.Name   `xml:"item"`
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	PubDate     string     `xml:"pubDate"`
	Author      string     `xml:"author,omitempty"`
	Content     string     `xml:"yandex:full-text"`
	Description string     `xml:"description,omitempty"`
	Source      *RssSource `xml:"source,omitempty"` // Добавляем поле для источника
}

type RssSource struct {
	XMLName xml.Name `xml:"source"`    // Указываем XML-тег для элемента <source>
	URL     string   `xml:"url,attr"`  // URL источника (атрибут)
	Name    string   `xml:",chardata"` // Название источника (текст внутри элемента)
}

func (rf *RssFeed) Add(item *RssItem) {
	rf.Items = append(rf.Items, item)
}

// AnyTimeFormat returns the first non-zero time formatted as a string or ""
func AnyTimeFormat(format string, times ...time.Time) string {
	for _, t := range times {
		if !t.IsZero() {
			return t.Format(format)
		}
	}
	return ""
}

type XmlFeed interface {
	FeedXml() interface{}
}

// WriteXML writes a feed object (either a Feed, AtomFeed, or RssFeed) as XML into
// the writer. Returns an error if XML marshaling fails.
func WriteXML(feed XmlFeed, w io.Writer) error {
	x := feed.FeedXml()
	// write default xml header, without the newline
	if _, err := w.Write([]byte(xml.Header[:len(xml.Header)-1])); err != nil {
		return err
	}
	e := xml.NewEncoder(w)
	e.Indent("", "  ")
	return e.Encode(x)
}

// FeedXml returns an XML-ready object for an RssFeed object
func (rf *RssFeed) FeedXml() interface{} {
	return &RssFeedXml{
		Version:         "2.0",
		Channel:         rf,
		YandexNamespace: "http://news.yandex.ru",
		MediaNamespace:  "http://search.yahoo.com/mrss/",
	}
}
