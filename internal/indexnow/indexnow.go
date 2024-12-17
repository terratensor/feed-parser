package indexnow

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type IndexNow struct {
	Key    string
	client *http.Client
}

func NewIndexNow(enabled bool) *IndexNow {
	if !enabled {
		return nil
	}
	key := os.Getenv("INDEX_NOW_KEY")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &IndexNow{
		Key:    key,
		client: client,
	}
}

func (i *IndexNow) Get(link string) error {

	var u = url.URL{
		Scheme: "https",
		Host:   "yandex.com",
		Path:   "indexnow",
	}

	q := u.Query()
	q.Set("key", i.Key)
	q.Set("url", link)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("ошибка при создании url: %v", err)
	}

	resp, err := i.client.Do(req)

	if err != nil {
		return fmt.Errorf("ошибка при отправке url: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		log.Printf("🚩🚩 IndexNow url: %v", link)
		log.Printf("🚩🚩 feed.svodd.ru url успешно передан: %v", link)
		break
	case http.StatusAccepted:
		log.Println("новый ключ ожидает проверки")
		break
	default:
		return fmt.Errorf("ошибка, код ответа %v, подробнее: https://yandex.ru/support/webmaster/indexnow/reference/get-url.html", resp.StatusCode)
	}
	return nil
}
