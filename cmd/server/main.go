package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Обработчик для основного RSS-фида
func handlerRssFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/rss.xml")
}

// Обработчик для RSS-фида Кремля
func handlerKremlinFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/kremlin-rss.xml")
}

// Обработчик для RSS-фида МИД
func handlerMidFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/mid-rss.xml")
}

// Обработчик для RSS-фида Минобороны
func handlerMilFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/mil-rss.xml")
}

// serveRssFile читает файл и отправляет его как ответ
func serveRssFile(w http.ResponseWriter, r *http.Request, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			log.Printf("error opening file %s: %v (client: %s)\n", filename, err, r.RemoteAddr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Printf("error getting file info %s: %v (client: %s)\n", filename, err, r.RemoteAddr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	http.ServeContent(w, r, filename, stat.ModTime(), file)
}

func main() {
	// Настройка временной зоны
	if tz := os.Getenv("TZ"); tz != "" {
		var err error
		time.Local, err = time.LoadLocation(tz)
		if err != nil {
			log.Printf("error loading location '%s': %v\n", tz, err)
		}
	}

	// Вывод текущей временной зоны
	tnow := time.Now()
	tz, _ := tnow.Zone()
	log.Printf("Local time zone %s. Server started at %s", tz,
		tnow.Format("2006-01-02T15:04:05.000 MST"))
	log.Println("Listening on port 8000")

	// Настройка логгера
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)

	// Создание мультиплексора
	mux := http.NewServeMux()

	// Обработчики для RSS-фидов
	mux.Handle("/rss.xml", logMiddleware(http.HandlerFunc(handlerRssFeed), logger))
	mux.Handle("/kremlin-rss.xml", logMiddleware(http.HandlerFunc(handlerKremlinFeed), logger))
	mux.Handle("/mid-rss.xml", logMiddleware(http.HandlerFunc(handlerMidFeed), logger))
	mux.Handle("/mil-rss.xml", logMiddleware(http.HandlerFunc(handlerMilFeed), logger))

	// Обработчик для статических файлов
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Настройка сервера с тайм-аутами
	server := &http.Server{
		Addr:         ":8000",
		Handler:      mux,
		ReadTimeout:  10 * time.Second, // Время ожидания данных от клиента
		WriteTimeout: 10 * time.Second, // Время ожидания отправки данных клиенту
		IdleTimeout:  60 * time.Second, // Время ожидания idle-соединения
	}

	// Запуск сервера
	log.Fatal(server.ListenAndServe())
}

// Middleware для логирования
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
			slog.String("X-Forwarded-For", r.Header.Get("X-Forwarded-For")),
			slog.String("X-Forwarded-Host", r.Header.Get("X-Forwarded-Host")),
			slog.String("X-Forwarded-Port", r.Header.Get("X-Forwarded-Port")),
			slog.String("X-Forwarded-Proto", r.Header.Get("X-Forwarded-Proto")),
			slog.String("X-Forwarded-Server", r.Header.Get("X-Forwarded-Server")),
			slog.String("X-Real-Ip", r.Header.Get("X-Real-Ip")),
			slog.String("referer", r.Referer()),
		)
		next.ServeHTTP(w, r)
	})
}
