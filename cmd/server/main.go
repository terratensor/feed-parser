package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Метрики Prometheus
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// Инициализация метрик
func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
}

// Обработчик для основного RSS-фида
func handlerRssFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/rss.xml")
}

// Обработчик для RSS-фида Кремля
func handlerKremlinFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/kremlin.xml")
}

// Обработчик для RSS-фида МИД
func handlerMidFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/mid.xml")
}

// Обработчик для RSS-фида Минобороны
func handlerMilFeed(w http.ResponseWriter, r *http.Request) {
	serveRssFile(w, r, "./static/mil.xml")
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
	mux.Handle("/kremlin.xml", logMiddleware(http.HandlerFunc(handlerKremlinFeed), logger))
	mux.Handle("/mid.xml", logMiddleware(http.HandlerFunc(handlerMidFeed), logger))
	mux.Handle("/mil.xml", logMiddleware(http.HandlerFunc(handlerMilFeed), logger))

	// Обработчик для статических файлов
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Обработчик для метрик Prometheus
	mux.Handle("/metrics", promhttp.Handler())

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
		start := time.Now()

		// Обертка для ResponseWriter, чтобы перехватить код статуса
		rw := &responseWriterWrapper{w, http.StatusOK}

		// Получаем реальный IP-адрес клиента
		realIP := r.Header.Get("X-Real-IP")
		if realIP == "" {
			// Если X-Real-IP не установлен, используем X-Forwarded-For
			forwardedFor := r.Header.Get("X-Forwarded-For")
			if forwardedFor != "" {
				// Берем первый IP из списка (реальный IP клиента)
				realIP = forwardedFor
			} else {
				// Если заголовки отсутствуют, используем RemoteAddr
				realIP = r.RemoteAddr
			}
		}

		// Логируем начало запроса
		logger.Info(
			"request started",
			slog.String("method", r.Method),
			slog.String("URL", r.URL.String()),
			slog.String("remote", realIP), // Используем реальный IP
			slog.String("userAgent", r.UserAgent()),
		)

		// Выполняем запрос
		next.ServeHTTP(rw, r)

		// Логируем завершение запроса
		logger.Info(
			"request completed",
			slog.String("method", r.Method),
			slog.String("URL", r.URL.String()),
			slog.String("remote", realIP), // Используем реальный IP
			slog.Int("status", rw.status),
			slog.Duration("duration", time.Since(start)),
		)

		// Обновляем метрики Prometheus
		requestsTotal.WithLabelValues(r.Method, r.URL.Path, http.StatusText(rw.status)).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	})
}

// Обертка для ResponseWriter, чтобы перехватить код статуса
type responseWriterWrapper struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
