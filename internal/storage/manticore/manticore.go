package manticore

import (
	"context"
	"encoding/json"
	"fmt"
	openapiclient "github.com/manticoresoftware/manticoresearch-go"
	"github.com/terratensor/feed-parser/internal/entities/feed"

	"log"
	"os"
	"strconv"
	"time"
)

var _ feed.StorageInterface = &Client{}

type Response struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total         int    `json:"total"`
		TotalRelation string `json:"total_relation"`
		Hits          []struct {
			Id     string `json:"_id"`
			Score  int    `json:"_score"`
			Source struct {
				Title      string `json:"title"`
				Summary    string `json:"summary"`
				Content    string `json:"content"`
				Published  int    `json:"published"`
				Updated    int    `json:"updated"`
				Language   string `json:"language"`
				Url        string `json:"url"`
				Author     string `json:"author"`
				Number     string `json:"number"`
				ResourceID int    `json:"resource_id"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type DBEntry struct {
	Language   string `json:"language"`
	Title      string `json:"title"`
	Url        string `json:"url"`
	Updated    int64  `json:"updated"`
	Published  int64  `json:"published"`
	Summary    string `json:"summary"`
	Content    string `json:"content"`
	Author     string `json:"author"`
	Number     string `json:"number"`
	ResourceID int    `json:"resource_id"`
}

type Client struct {
	apiClient *openapiclient.APIClient
	Index     string
}

func NewDBEntry(entry *feed.Entry) *DBEntry {
	var updated int64
	if entry.Updated == nil {
		updated = 0
	} else {
		updated = entry.Updated.Unix()
	}

	var published int64
	if entry.Published == nil {
		published = 0
	} else {
		published = entry.Published.Unix()
	}

	dbe := &DBEntry{
		Language:   entry.Language,
		Title:      entry.Title,
		Url:        entry.Url,
		Updated:    updated,
		Published:  published,
		Summary:    entry.Summary,
		Content:    entry.Content,
		Author:     entry.Author,
		Number:     entry.Number,
		ResourceID: entry.ResourceID,
	}

	return dbe
}

func New(tbl string) (*Client, error) {
	// Initialize ApiClient
	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)

	query := fmt.Sprintf(`show tables like '%v'`, tbl)

	// Проверяем существует ли таблица tbl, если нет, то создаем
	resp, _, err := apiClient.UtilsAPI.Sql(context.Background()).Body(query).Execute()
	if err != nil {
		return nil, err
	}
	data := resp[0]["data"].([]interface{})

	if len(data) > 0 {
		myMap := data[0].(map[string]interface{})
		indexValue := myMap["Index"]

		if indexValue != tbl {
			err := createTable(apiClient, tbl)
			if err != nil {
				return nil, err
			}
		}
	} else {
		err := createTable(apiClient, tbl)
		if err != nil {
			return nil, err
		}
	}

	return &Client{apiClient: apiClient, Index: tbl}, nil
}

func createTable(apiClient *openapiclient.APIClient, tbl string) error {

	query := fmt.Sprintf(`create table %v(language string, url string, title text, summary text, content text, published timestamp, updated timestamp, author string, number string, resource_id int) engine='columnar' min_infix_len='3' index_exact_words='1' morphology='stem_en, stem_ru, libstemmer_de, libstemmer_fr, libstemmer_es, libstemmer_pt' html_remove_elements = 'style, script' html_strip = '1' index_sp='1'`, tbl)

	sqlRequest := apiClient.UtilsAPI.Sql(context.Background()).Body(query)
	_, _, err := apiClient.UtilsAPI.SqlExecute(sqlRequest)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Insert(ctx context.Context, entry *feed.Entry) (*int64, error) {

	dbe := NewDBEntry(entry)

	//marshal into JSON buffer
	buffer, err := json.Marshal(dbe)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v\n", err)
	}

	var doc map[string]interface{}
	err = json.Unmarshal(buffer, &doc)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling buffer: %v\n", err)
	}

	idr := openapiclient.InsertDocumentRequest{
		Index: c.Index,
		Doc:   doc,
	}

	resp, r, err := c.apiClient.IndexAPI.Insert(ctx).InsertDocumentRequest(idr).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, fmt.Errorf("Error when calling `IndexAPI.Insert``: %v\n", err)
	}

	return resp.Id, nil
}

func (c *Client) Update(ctx context.Context, entry *feed.Entry) error {
	dbe := NewDBEntry(entry)

	//marshal into JSON buffer
	buffer, err := json.Marshal(dbe)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v\n", err)
	}

	var doc map[string]interface{}
	err = json.Unmarshal(buffer, &doc)
	if err != nil {
		return fmt.Errorf("error unmarshaling buffer: %v\n", err)
	}

	idr := openapiclient.InsertDocumentRequest{
		Index: c.Index,
		Id:    entry.ID,
		Doc:   doc,
	}

	_, r, err := c.apiClient.IndexAPI.Replace(ctx).InsertDocumentRequest(idr).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return fmt.Errorf("Error when calling `IndexAPI.Replace``: %v\n", err)
	}

	log.Printf("Success `IndexAPI.Replace`: %v\n", r)

	return nil
}

func (c *Client) Bulk(ctx context.Context, entries *[]feed.Entry) error {

	var serializedEntries string
	for _, e := range *entries {
		eJSON, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("error marshaling JSON: %v\n", err)
		}
		serializedEntries += string(eJSON) + "\n"
	}

	fmt.Println(serializedEntries)

	_, r, err := c.apiClient.IndexAPI.Bulk(ctx).Body(serializedEntries).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return fmt.Errorf("Error when calling `IndexAPI.Insert``: %v\n", err)
	}
	// response from `Insert`: SuccessResponse
	fmt.Fprintf(os.Stdout, "Success Response from `IndexAPI.Insert`: %v\n", r)

	return nil
}

func (c *Client) FindByUrl(ctx context.Context, url string) (*feed.Entry, error) {
	// response from `Search`: SearchRequest
	searchRequest := *openapiclient.NewSearchRequest(c.Index)

	// Perform a search
	// Пример для запроса фильтра по url
	filter := map[string]interface{}{"url": url}
	query := map[string]interface{}{"equals": filter}

	searchRequest.SetQuery(query)
	resp, r, err := c.apiClient.SearchAPI.Search(ctx).SearchRequest(searchRequest).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, fmt.Errorf("Error when calling `SearchAPI.Search.Equals``: %v\n", err)
	}

	id, err := getEntryID(resp)
	if err != nil {
		return nil, err
	}

	dbe := makeDBEntry(resp)
	if dbe == nil {
		return nil, nil
	}
	//log.Printf("id %d\n", *id)
	//log.Printf("dbe %v\n", dbe)
	updated := time.Unix(dbe.Updated, 0)
	published := time.Unix(dbe.Published, 0)

	ent := &feed.Entry{
		ID:         id,
		Language:   dbe.Language,
		Title:      dbe.Title,
		Url:        dbe.Url,
		Updated:    &updated,
		Published:  &published,
		Summary:    dbe.Summary,
		Content:    dbe.Content,
		Author:     dbe.Author,
		Number:     dbe.Number,
		ResourceID: dbe.ResourceID,
	}

	return ent, nil
}

func makeDBEntry(resp *openapiclient.SearchResponse) *DBEntry {
	var hits []map[string]interface{}
	hits = resp.Hits.Hits

	// Если слайс Hits пустой (0) значит нет совпадений
	if len(hits) == 0 {
		return nil
	}

	hit := hits[0]

	sr := hit["_source"]
	jsonData, err := json.Marshal(sr)

	var dbe DBEntry
	err = json.Unmarshal(jsonData, &dbe)
	if err != nil {
		log.Fatal(err)
	}

	return &dbe
}

func getEntryID(resp *openapiclient.SearchResponse) (*int64, error) {
	var hits []map[string]interface{}
	var _id interface{}

	hits = resp.Hits.Hits

	// Если слайс Hits пустой (0) значит нет совпадений
	if len(hits) == 0 {
		return nil, nil
	}

	hit := hits[0]

	_id = hit["_id"]
	id, err := strconv.ParseInt(_id.(string), 10, 64)
	//log.Printf("id %d\n", id)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse ID to int64: %v\n", resp)
	}

	return &id, nil
}
