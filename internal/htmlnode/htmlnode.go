package htmlnode

import (
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"time"
)

func GetTopicBody(url string) (*html.Node, error) {

	resp, err := call(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("status code error: %d %s\r\n", resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}
	doc, err := html.Parse(resp.Body)

	if err != nil {
		return nil, err // Handle error
	}
	return doc, nil
}

// call is a Go function that makes a GET request to the provided URL and returns the response and an error, if any.
//
// It takes a string 'url' as a parameter and returns a pointer to http.Response and an error.
func call(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Без user-agent kremlin.ru не отдает данные
	req.Header.Add("User-Agent", "SvoddProgram/0.01")
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, err
}
