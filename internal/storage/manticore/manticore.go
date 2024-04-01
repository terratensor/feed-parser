package manticore

import (
	"context"
	"encoding/json"
	"fmt"
	openapiclient "github.com/manticoresoftware/manticoresearch-go"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"io"
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
				Published  int64  `json:"published"`
				Updated    int64  `json:"updated"`
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
	Created    int64  `json:"created"`
}

type Client struct {
	apiClient *openapiclient.APIClient
	Index     string
}

// NewDBEntry только для создания новой записи insert, в этой записи присваивается поле created,
// далее это поле никогда не изменяется при редактировании записи.
func NewDBEntry(entry *feed.Entry) *DBEntry {

	created := time.Now()

	dbe := &DBEntry{
		Language:   entry.Language,
		Title:      entry.Title,
		Url:        entry.Url,
		Updated:    castTime(entry.Updated),
		Published:  castTime(entry.Published),
		Summary:    entry.Summary,
		Content:    entry.Content,
		Author:     entry.Author,
		Number:     entry.Number,
		ResourceID: entry.ResourceID,
		Created:    castTime(&created),
	}

	return dbe
}

func castTime(value *time.Time) int64 {
	if value == nil {
		return 0
	} else if time.Time.IsZero(*value) {
		return 0
	} else {
		return value.Unix()
	}
}

func New(tbl string) (*Client, error) {
	// Initialize ApiClient
	configuration := openapiclient.NewConfiguration()
	configuration.Servers = openapiclient.ServerConfigurations{
		{
			URL: "http://manticore_feed:9308",
			//URL:         "http://localhost:9308",
			Description: "Default Manticore Search HTTP",
		},
	}
	//configuration.ServerURL(1, map[string]string{"URL": "http://manticore:9308"})
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

	query := fmt.Sprintf(`create table %v(language string, url string, title text, summary text, content text, published timestamp, updated timestamp, author string, number string, resource_id int, created timestamp) engine='columnar' min_infix_len='3' index_exact_words='1' morphology='stem_en, stem_ru, libstemmer_de, libstemmer_fr, libstemmer_es, libstemmer_pt' index_sp='1'`, tbl)

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

	dbe := &DBEntry{
		Language:   entry.Language,
		Title:      entry.Title,
		Url:        entry.Url,
		Updated:    castTime(entry.Updated),
		Published:  castTime(entry.Published),
		Summary:    entry.Summary,
		Content:    entry.Content,
		Author:     entry.Author,
		Number:     entry.Number,
		ResourceID: entry.ResourceID,
		Created:    castTime(entry.Created),
	}

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

	updated := time.Unix(dbe.Updated, 0)
	published := time.Unix(dbe.Published, 0)
	created := time.Unix(dbe.Created, 0)

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
		Created:    &created,
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

func (c *Client) FindAll(ctx context.Context, limit int) {
	body1 := "show index feed status"
	resp, r, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(body1).RawResponse(true).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
	}
	fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
	return

	count := 0
	for {
		offset := count * limit
		maxMatches := offset + limit
		body := fmt.Sprintf("select * from feed limit %v offset %v option max_matches=%v", limit, offset, maxMatches)

		resp, r, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(body).RawResponse(false).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		}
		// response from `Sql`: []map[string]interface{}
		//fmt.Fprintf(os.Stdout, "Response from `UtilsAPI.Sql`: %v\n", resp)

		// Assuming the data is stored in a variable called 'jsonData'
		//var data map[string]interface{}
		//fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r.Body)
		v := &Response{}
		data, err := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, v); err != nil {
			log.Fatalf("Parse response failed, reason: %v \n", err)
		}

		//log.Printf("Response from `UtilsAPI.Sql`: %v\n", v.Hits.Hits)

		for _, h := range v.Hits.Hits {
			log.Printf("h.id: %v h.Title %v\n", h.Source.Url, h.Source.Title)
		}

		//var ids []string
		//for _, hit := range hits {
		//	//log.Printf("hit: %v\n", hit)
		//	id := hit.(map[string]interface{})["_id"].(string)
		//	source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		//	log.Printf("source: %v\n", source)
		//	dbe := &DBEntry{
		//		Language:   source["language"].(string),
		//		Title:      source["title"].(string),
		//		Url:        source["url"].(string),
		//		Updated:    source["updated"].(int64),
		//		Published:  source["published"].(int64),
		//		Summary:    source["summary"].(string),
		//		Content:    source["content"].(string),
		//		Author:     source["author"].(string),
		//		Number:     source["number"].(string),
		//		ResourceID: source["resource_id"].(int),
		//		Created:    source["created"].(int64),
		//	}
		//	log.Printf("dbe: %v\n", dbe)
		//	ids = append(ids, id)
		//}
		//
		//log.Printf("ids: %v\n", ids)

		count++
		log.Printf("🚩 count: %v\n", count)
		if count > 10 {
			break
		}
	}

}
