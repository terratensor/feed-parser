package main

import (
	"context"
	flag "github.com/spf13/pflag"
	"github.com/terratensor/feed-parser/internal/entities/feed"
	"github.com/terratensor/feed-parser/internal/splitter"
	"github.com/terratensor/feed-parser/internal/storage/manticore"
	"log"
	"os"
	"os/signal"
	"time"
)

var optParSize, maxParSize int
var copyTable, processLongPar bool

func main() {

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	flag.IntVarP(&optParSize, "optParSize", "o", 1800, "граница оптимального размера параграфа в символах, если 0, то без склейки параграфов")
	flag.IntVarP(&maxParSize, "maxParSize", "m", 3600, "граница максимального размера параграфа в символах, если 0, то без склейки параграфов")
	flag.BoolVarP(&copyTable, "copy", "c", false, "только копирование записей в новую таблицу без разбивки")
	flag.Parse()

	// БД из которой читаем записи
	manticoreClient, err := manticore.New("feed")
	if err != nil {
		log.Printf("failed to initialize manticore client: %v", err)
		os.Exit(1)
	}
	entries := feed.NewFeedStorage(manticoreClient)

	// БД в которую пишем разделенные на фрагменты записи
	newManticoreClient, err := manticore.New("feed_new")
	if err != nil {
		log.Printf("failed to initialize manticore client: %v", err)
		os.Exit(1)
	}
	splitEntries := feed.NewFeedStorage(newManticoreClient)

	defer duration(track("🚩 выполнено за"))

	ch, err := entries.FindAll(ctx, 10000)
	if err != nil {
		log.Fatalf("failed to find all entries: %v", err)
	}

	sp := splitter.NewSplitter(optParSize, maxParSize)

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}

			if copyTable {
				id, err := splitEntries.Storage.Insert(ctx, &e)
				if err != nil {
					log.Fatalf("failed to insert entry: %v", err)
				}
				log.Printf("Entry inserted: %v", *id)
			} else {
				newEntries := sp.SplitEntry(ctx, e)

				for _, newEntry := range newEntries {
					_, err := splitEntries.Storage.Insert(ctx, &newEntry)
					if err != nil {
						log.Fatalf("failed to insert entry: %v", err)
					}
					//log.Printf("Entry inserted: %v", *id)
				}
			}

		}
	}
}

func track(msg string) (string, time.Time) {
	return msg, time.Now()
}

func duration(msg string, start time.Time) {
	log.Printf("%v: %v\n", msg, time.Since(start))
}
