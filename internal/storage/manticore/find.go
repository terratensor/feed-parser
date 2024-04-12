package manticore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

func (c *Client) FindDuration(ctx context.Context, duration time.Duration) (chan feed.Entry, error) {

	chout := make(chan feed.Entry, 100)
	//maxMatches := offset + limit

	currentTime := time.Now().Unix()
	intervalDate := currentTime - int64(duration.Seconds())

	count := c.calculateLimitCount(intervalDate)
	limit := count
	maxMatches := count

	log.Println(count)

	go func() {
		defer close(chout)

		body := fmt.Sprintf("select * from %v where chunk=1 and published > %v order by published desc limit %v option max_matches=%v", c.Index, intervalDate, limit, maxMatches)

		resp, r, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(body).RawResponse(false).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			return
		}

		v := &Response{}
		data, err := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, v); err != nil {
			log.Fatalf("Parse response failed, reason: %v \n", err)
			return
		}

		log.Println(len(v.Hits.Hits))

		for _, hit := range v.Hits.Hits {

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

			id, err := strconv.ParseInt(hit.Id, 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse ID to int64: %v\n", hit)
				return
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

	}()

	return chout, nil
}

func (c *Client) calculateLimitCount(intervalDate int64) int {

	countBody := fmt.Sprintf("select count(*) from %v where chunk=1 and published > %v", c.Index, intervalDate)
	log.Println(countBody)
	resp1, r1, err := c.apiClient.UtilsAPI.Sql(context.Background()).Body(countBody).RawResponse(false).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UtilsAPI.Sql``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp1)
	}

	log.Println(r1)
	return 203
}
