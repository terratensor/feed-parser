package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
	if tz := os.Getenv("TZ"); tz != "" {
		var err error
		time.Local, err = time.LoadLocation(tz)
		if err != nil {
			log.Printf("error loading location '%s': %v\n", tz, err)
		}
	}

	// output current time zone
	tnow := time.Now()
	tz, _ := tnow.Zone()
	log.Printf("Local time zone %s. Server started at %s", tz,
		tnow.Format("2006-01-02T15:04:05.000 MST"))
	log.Printf("Listening on port %s", os.Getenv("SRV_PORT"))

	http.HandleFunc("/rss.xml", handler)
	log.Fatal(http.ListenAndServe(":8011", nil))
}
