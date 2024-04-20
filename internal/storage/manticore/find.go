package manticore

import (
	"context"
	"fmt"
	"os"
	"time"
)

func (c *Client) FindDuration(ctx context.Context, duration time.Duration) (chan string, error) {

	chout := make(chan string, 100)
	//maxMatches := offset + limit

	intervalDate := getIntervalDate(duration)
	count := c.CalculateLimitCount(duration)

	limit := count
	maxMatches := count

	go func() {
		defer close(chout)

		body := fmt.Sprintf("SELECT url FROM %v WHERE chunk=1 AND published > %v ORDER BY published DESC limit %v option max_matches=%v", c.Index, intervalDate, limit, maxMatches)

		resp, _, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(body).RawResponse(true).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			return
		}

		var sqlResp []map[string]interface{}
		sqlResp = resp
		data := sqlResp[0]["data"].([]interface{})

		for _, rows := range data {
			row := rows.(map[string]interface{})
			chout <- row["url"].(string)
		}

		//v := &Response{}
		//data, err := io.ReadAll(r.Body)
		//if err := json.Unmarshal(data, v); err != nil {
		//	log.Fatalf("Parse response failed, reason: %v \n", err)
		//	return
		//}
		//
		////log.Println(len(v.Hits.Hits))
		//
		//for _, hit := range v.Hits.Hits {
		//
		//	source := hit.Source
		//
		//	dbe := &DBEntry{
		//		Language:   source.Language,
		//		Title:      source.Title,
		//		Url:        source.Url,
		//		Updated:    source.Updated,
		//		Published:  source.Published,
		//		Summary:    source.Summary,
		//		Content:    source.Content,
		//		Author:     source.Author,
		//		Number:     source.Number,
		//		ResourceID: source.ResourceID,
		//		Created:    source.Created,
		//		Chunk:      source.Chunk,
		//		UpdatedAt:  source.UpdatedAt,
		//	}
		//
		//	id, err := strconv.ParseInt(hit.Id, 10, 64)
		//	if err != nil {
		//		fmt.Fprintf(os.Stderr, "Failed to parse ID to int64: %v\n", hit)
		//		return
		//	}
		//
		//	updated := time.Unix(dbe.Updated, 0)
		//	published := time.Unix(dbe.Published, 0)
		//	created := time.Unix(dbe.Created, 0)
		//	updatedAt := time.Unix(dbe.UpdatedAt, 0)
		//
		//	chout <- feed.Entry{
		//		ID:         &id,
		//		Language:   dbe.Language,
		//		Title:      dbe.Title,
		//		Url:        dbe.Url,
		//		Updated:    &updated,
		//		Published:  &published,
		//		Summary:    dbe.Summary,
		//		Content:    dbe.Content,
		//		Author:     dbe.Author,
		//		Number:     dbe.Number,
		//		ResourceID: dbe.ResourceID,
		//		Created:    &created,
		//		Chunk:      dbe.Chunk,
		//		UpdatedAt:  &updatedAt,
		//	}
		//}

	}()

	return chout, nil
}

func (c *Client) CalculateLimitCount(duration time.Duration) int {

	intervalDate := getIntervalDate(duration)

	countBody := fmt.Sprintf("SELECT count(*) FROM %v WHERE chunk=1 AND published > %v", c.Index, intervalDate)

	resp, _, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(countBody).RawResponse(true).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
	}

	var sqlResp []map[string]interface{}
	sqlResp = resp
	data := sqlResp[0]["data"].([]interface{})

	var count int
	for _, rows := range data {
		row := rows.(map[string]interface{})
		strCount := row["count(*)"]
		count = int(strCount.(float64))
	}
	return count
}

func getIntervalDate(duration time.Duration) int64 {
	currentTime := time.Now().Unix()
	intervalDate := currentTime - int64(duration.Seconds())
	return intervalDate
}
