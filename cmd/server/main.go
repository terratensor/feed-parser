package main

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func handlerRssFeed(w http.ResponseWriter, r *http.Request) {

	streamXmlBytes, err := os.ReadFile("./static/rss.xml")

	if err != nil {
		log.Printf("error reading file: %v\n", err)
	}

	b := bytes.NewBuffer(streamXmlBytes)

	w.Header().Set("Content-type", "application/xml")

	if _, err := b.WriteTo(w); err != nil {
		log.Printf("error writing response: %v\n", err)
		fmt.Fprintf(w, "%s", err)
	}
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
	log.Println("Listening on port 8000")

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	mux := http.NewServeMux()
	handler := http.HandlerFunc(handlerRssFeed)
	mux.Handle("/rss.xml", logMiddleware(handler, logger))
	log.Fatal(http.ListenAndServe(":8000", mux))
}

func logMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(
			"request",
			slog.String("method", r.Method),
			slog.String("URL", r.URL.String()),
			slog.String("proto", r.Proto),
			slog.String("host", r.Host),
			slog.String("remote", r.RemoteAddr),
			slog.String("requestURI", r.RequestURI),
			slog.String("userAgent", r.UserAgent()),
		)
		next.ServeHTTP(w, r)
	})
}
