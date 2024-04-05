package app

import (
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
	"log"
	"os"
)

func NewEntriesStorage(index string) *feed.Entries {
	var storage feed.StorageInterface

	manticoreClient, err := manticore.New(index)
	if err != nil {
		log.Printf("failed to initialize manticore client, %v", err)
		os.Exit(1)
	}

	storage = manticoreClient

	return feed.NewFeedStorage(storage)
}
