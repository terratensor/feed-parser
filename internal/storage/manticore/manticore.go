package manticore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	openapiclient "github.com/manticoresoftware/manticoresearch-go"
	"github.com/terratensor/feed-parser/internal/entities/feed"
)

var _ feed.StorageInterface = &Client{}

type Response struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total         int    `json:"total"`
		TotalRelation string `json:"total_relation"`
		Hits          []struct {
			Id     int64 `json:"_id"`
			Score  int   `json:"_score"`
			Source struct {
				Title      string `json:"title"`
				Summary    string `json:"summary"`
				Content    string `json:"content"`
				ResourceID int    `json:"resource_id"`
				Chunk      int    `json:"chunk"`
				Published  int64  `json:"published"`
				Updated    int64  `json:"updated"`
				Created    int64  `json:"created"`
				UpdatedAt  int64  `json:"updated_at"`
				Language   string `json:"language"`
				Url        string `json:"url"`
				Author     string `json:"author"`
				Number     string `json:"number"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type IndexStatus struct {
	Columns []map[string]map[string]string `json:"columns"`
	Data    []map[string]string            `json:"data"`
	Total   int                            `json:"total"`
	Error   string                         `json:"error"`
	Warning string                         `json:"warning"`
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
	Chunk      int    `json:"chunk"`
	UpdatedAt  int64  `json:"updated_at"`
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
		Chunk:      entry.Chunk,
		UpdatedAt:  castTime(&created),
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
	apiClient := openapiclient.NewAPIClient(configuration)

	query := fmt.Sprintf(`show tables like '%v'`, tbl)

	// Проверяем существует ли таблица tbl, если нет, то создаем
	resp, _, err := apiClient.UtilsAPI.Sql(context.Background()).Body(query).Execute()
	if err != nil {
		return nil, err
	}

	// Обрабатываем новый формат ответа
	if len(resp) > 0 {
		data, exists := resp[0]["data"]
		if !exists || data == nil {
			// Нет данных - таблица не существует, создаем
			err := createTable(apiClient, tbl)
			if err != nil {
				return nil, err
			}
		} else {
			dataArray := data.([]interface{})
			if len(dataArray) == 0 {
				// Пустой массив данных - таблица не существует, создаем
				err := createTable(apiClient, tbl)
				if err != nil {
					return nil, err
				}
			} else {
				// Проверяем имя таблицы в ответе
				firstRow := dataArray[0].(map[string]interface{})
				if tableName, ok := firstRow["Table"].(string); !ok || tableName != tbl {
					err := createTable(apiClient, tbl)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	} else {
		// Пустой ответ - таблица не существует, создаем
		err := createTable(apiClient, tbl)
		if err != nil {
			return nil, err
		}
	}

	return &Client{apiClient: apiClient, Index: tbl}, nil
}

func createTable(apiClient *openapiclient.APIClient, tbl string) error {

	query := fmt.Sprintf(`create table %v(language string, url string, title text, summary text, content text, published timestamp, updated timestamp, author string, number string, resource_id int, created timestamp, updated_at timestamp, chunk int) min_infix_len='3' index_exact_words='1' morphology='stem_en, stem_ru, libstemmer_de, libstemmer_fr, libstemmer_es, libstemmer_pt' index_sp='1'`, tbl)

	sqlRequest := apiClient.UtilsAPI.Sql(context.Background()).Body(query)
	_, _, err := apiClient.UtilsAPI.SqlExecute(sqlRequest)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Insert(ctx context.Context, entry *feed.Entry) (*int64, error) {
	dbe := NewDBEntry(entry)

	// Marshal into JSON buffer
	buffer, err := json.Marshal(dbe)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}

	var doc map[string]interface{}
	err = json.Unmarshal(buffer, &doc)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling buffer: %v", err)
	}

	idr := openapiclient.InsertDocumentRequest{
		Index: c.Index,
		Doc:   doc,
	}

	resp, r, err := c.apiClient.IndexAPI.Insert(ctx).InsertDocumentRequest(idr).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		return nil, fmt.Errorf("error when calling `IndexAPI.Insert`: %v", err)
	}

	// ---- Обработка тела ответа для получения ID ----
	defer r.Body.Close()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var result struct {
		Id int64 `json:"id"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response body: %v", err)
	}

	id := result.Id
	return &id, nil
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
		Chunk:      entry.Chunk,
		UpdatedAt:  castTime(entry.UpdatedAt),
	}

	//marshal into JSON buffer
	buffer, err := json.Marshal(dbe)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	var doc map[string]interface{}
	err = json.Unmarshal(buffer, &doc)
	if err != nil {
		return fmt.Errorf("error unmarshaling buffer: %v", err)
	}

	idr := openapiclient.InsertDocumentRequest{
		Index: c.Index,
		Id:    entry.ID,
		Doc:   doc,
	}

	_, r, err := c.apiClient.IndexAPI.Replace(ctx).InsertDocumentRequest(idr).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return fmt.Errorf("error when calling `IndexAPI.Replace``: %v", err)
	}

	// log.Printf("Success `IndexAPI.Replace`: %v\n", r)

	return nil
}

func (c *Client) Bulk(ctx context.Context, entries *[]feed.Entry) error {

	var serializedEntries string
	for _, e := range *entries {
		eJSON, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("error marshaling JSON: %v", err)
		}
		serializedEntries += string(eJSON) + "\n"
	}

	fmt.Println(serializedEntries)

	_, r, err := c.apiClient.IndexAPI.Bulk(ctx).Body(serializedEntries).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v", r)
		return fmt.Errorf("error when calling `IndexAPI.Insert``: %v", err)
	}
	// response from `Insert`: SuccessResponse
	fmt.Fprintf(os.Stdout, "Success Response from `IndexAPI.Insert`: %v", r)

	return nil
}

func (c *Client) FindByUrl(ctx context.Context, url string) (*feed.Entry, error) {
	// response from `Search`: SearchRequest
	searchRequest := *openapiclient.NewSearchRequest(c.Index)

	// Perform a search
	filter := map[string]interface{}{"url": url}
	query := map[string]interface{}{"equals": filter}
	searchRequest.SetQuery(query)

	const maxRetries = 10
	const retryDelay = 100 * time.Millisecond

	var resp *openapiclient.SearchResponse
	var r *http.Response
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, r, err = c.apiClient.SearchAPI.Search(ctx).SearchRequest(searchRequest).Execute()

		if err == nil {
			break
		}

		if attempt < maxRetries-1 {
			fmt.Fprintf(os.Stderr, "Attempt %d failed: %v\n", attempt+1, err)
			time.Sleep(retryDelay)
			continue
		}

		fmt.Fprintf(os.Stderr, "Full HTTP response: %v", r)
		log.Printf("error when calling `SearchAPI.Search.Equals `FindByUrl``: %v | query: %v", err, query)
		log.Printf("resp: %v", resp)
		log.Printf("r: %v", r)
		return nil, fmt.Errorf("error when calling `SearchAPI.Search.Equals `FindByUrl``: %v", err)
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
	updatedAt := time.Unix(dbe.UpdatedAt, 0)

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
		Chunk:      dbe.Chunk,
		UpdatedAt:  &updatedAt,
	}

	return ent, nil
}

func (c *Client) FindAllByUrl(ctx context.Context, url string) ([]feed.Entry, error) {
	// response from `Search`: SearchRequest
	searchRequest := *openapiclient.NewSearchRequest(c.Index)

	filter := map[string]interface{}{"url": url}
	query := map[string]interface{}{"equals": filter}
	limit := 1000
	sort := []map[string]interface{}{{"chunk": "asc"}}

	searchRequest.SetQuery(query)
	searchRequest.SetLimit(int32(limit))
	searchRequest.SetSort(sort)

	const maxRetries = 10
	const retryDelay = 100 * time.Millisecond

	var resp *openapiclient.SearchResponse
	var r *http.Response
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, r, err = c.apiClient.SearchAPI.Search(ctx).SearchRequest(searchRequest).Execute()

		if err == nil {
			break
		}

		if attempt < maxRetries-1 {
			fmt.Fprintf(os.Stderr, "Attempt %d failed: %v\n", attempt+1, err)
			time.Sleep(retryDelay)
			continue
		}

		fmt.Fprintf(os.Stderr, "Full HTTP response: %v", r)
		log.Printf("error when calling `SearchAPI.Search.Equals `FindAllByUrl`: %v | query: %v | limit: %v | sort: %v", err, query, limit, sort)
		log.Printf("resp: %v", resp)
		log.Printf("r: %v", r)
		return nil, fmt.Errorf("error when calling `SearchAPI.Search.Equals `FindAllByUrl`: %v", err)
	}

	res := &Response{}
	respBody, _ := io.ReadAll(r.Body)
	err = json.Unmarshal(respBody, res)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", resp)
	}

	hits := res.Hits.Hits
	var entries []feed.Entry
	for _, hit := range hits {
		id := hit.Id
		sr := hit.Source

		jsonData, err := json.Marshal(sr)
		if err != nil {
			log.Fatal(err)
		}

		var dbe DBEntry
		err = json.Unmarshal(jsonData, &dbe)
		if err != nil {
			log.Fatal(err)
		}

		updated := time.Unix(dbe.Updated, 0)
		published := time.Unix(dbe.Published, 0)
		created := time.Unix(dbe.Created, 0)
		updatedAt := time.Unix(dbe.UpdatedAt, 0)

		ent := &feed.Entry{
			ID:         &id,
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
			Chunk:      dbe.Chunk,
			UpdatedAt:  &updatedAt,
		}

		entries = append(entries, *ent)
	}

	return entries, nil
}

func makeDBEntry(resp *openapiclient.SearchResponse) *DBEntry {
	var hits []map[string]interface{} = resp.Hits.Hits

	// Если слайс Hits пустой (0) значит нет совпадений
	if len(hits) == 0 {
		return nil
	}

	hit := hits[0]

	sr := hit["_source"]
	jsonData, err := json.Marshal(sr)
	if err != nil {
		log.Fatal(err)
	}

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

	if err != nil {
		return nil, fmt.Errorf("failed to parse ID to int64: %v", resp)
	}

	return &id, nil
}

func (c *Client) FindAll(ctx context.Context, limit int) (chan feed.Entry, error) {

	chout := make(chan feed.Entry, 100)

	indexedDocuments := getIndexStatus(c)
	log.Printf("indexedDocuments: %d\n", indexedDocuments)

	var lastCount int
	if indexedDocuments%limit > 0 {
		lastCount = indexedDocuments/limit + 1
	} else {
		lastCount = indexedDocuments / limit
	}

	//lastCount = 1
	go func() {
		defer close(chout)

		count := 0

		for {
			offset := count * limit
			//maxMatches := offset + limit
			body := fmt.Sprintf("select * from %v limit %v offset %v option max_matches=%v", c.Index, limit, offset, indexedDocuments)
			//body := fmt.Sprintf("select * from feed order by id asc limit %v offset %v option max_matches=%v", limit, offset, indexedDocuments)

			resp, r, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(body).RawResponse(false).Execute()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
				fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
				return
			}

			v := &Response{}
			data, err := io.ReadAll(r.Body)
			if err != nil {
				log.Fatalf("Failed reading response body: %v", err)
				return
			}
			if err := json.Unmarshal(data, v); err != nil {
				log.Fatalf("Parse response failed, reason: %v \n", err)
				return
			}

			for _, hit := range v.Hits.Hits {

				id := hit.Id
				source := hit.Source

				dbe := &DBEntry{
					Language:   source.Language,
					Title:      source.Title,
					Url:        source.Url,
					Updated:    source.Updated,
					Published:  source.Published,
					Summary:    source.Summary,
					Content:    source.Content,
					Author:     source.Author,
					Number:     source.Number,
					ResourceID: source.ResourceID,
					Created:    source.Created,
					Chunk:      source.Chunk,
					UpdatedAt:  source.UpdatedAt,
				}

				updated := time.Unix(dbe.Updated, 0)
				published := time.Unix(dbe.Published, 0)
				created := time.Unix(dbe.Created, 0)
				updatedAt := time.Unix(dbe.UpdatedAt, 0)

				chout <- feed.Entry{
					ID:         &id,
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
					Chunk:      dbe.Chunk,
					UpdatedAt:  &updatedAt,
				}
			}

			count++
			log.Printf("🚩 count: %v\n", count)
			if count > lastCount {
				break
			}
		}
	}()

	return chout, nil
}

func getIndexStatus(c *Client) int {
	body := fmt.Sprintf("show index %v status", c.Index)
	resp, r, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(body).RawResponse(true).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}

	var sqlResp []map[string]interface{} = resp
	data := sqlResp[0]["data"].([]interface{})

	for _, rows := range data {
		row := rows.(map[string]interface{})
		if row["Variable_name"] == "indexed_documents" {
			value := row["Value"].(string)

			intValue, err := strconv.Atoi(value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error when parse int: %v\n", err)
			}
			return intValue
		}
	}
	return 0
}

func (c *Client) Delete(ctx context.Context, id *int64) error {

	deleteDocumentRequest := *openapiclient.NewDeleteDocumentRequest(c.Index) // DeleteDocumentRequest |

	deleteDocumentRequest.Id = id

	resp, r, err := c.apiClient.IndexAPI.Delete(context.Background()).DeleteDocumentRequest(deleteDocumentRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `IndexAPI.Delete` ID: %v, Error: %v\n", *id, err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return err
	}
	// response from `Delete`: DeleteResponse
	fmt.Fprintf(os.Stdout, "Response from `IndexAPI.Delete`: %v\n", resp)
	return nil
}
