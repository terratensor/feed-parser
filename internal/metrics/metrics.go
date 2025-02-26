package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	SuccessRequests  *prometheus.CounterVec
	ErrorRequests    *prometheus.CounterVec
	EntitiesInserted *prometheus.CounterVec
	EntitiesUpdated  *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		SuccessRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rss_parser_success_requests_total",
				Help: "Total number of successful requests.",
			},
			[]string{"url", "attempt"},
		),
		// Метрика для подсчета ошибок
		ErrorRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rss_parser_error_requests_total",
				Help: "Total number of failed requests.",
			},
			[]string{"url", "error_type", "attempt"}, // Метки для URL, типа ошибки и номера попытки
		),
		// Метрика для подсчета новых новостей, записанных в БД
		EntitiesInserted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rss_parser_entities_inserted_total",
				Help: "Total number of entities items inserted into the database.",
			},
			[]string{"url", "chunks"}, // Метка для URL, кол-во фрагментов
		),
		// Метрика для подсчета обновленных новостей в БД
		EntitiesUpdated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rss_parser_entities_updated_total",
				Help: "Total number of entities items updated in the database.",
			},
			[]string{"url", "chunks"}, // Метка для URL, кол-во фрагментов
		),
	}
}

func (m *Metrics) Register() {
	prometheus.MustRegister(m.SuccessRequests)
	prometheus.MustRegister(m.ErrorRequests)
	prometheus.MustRegister(m.EntitiesInserted)
	prometheus.MustRegister(m.EntitiesUpdated)
}
