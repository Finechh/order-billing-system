package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) initKafkaMetrics(service string) {
	m.KafkaMessagesConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "kafka",
			Name:      "messages_consumed_total",
			Help:      "Total consumed Kafka messages",
		},
		[]string{"topic", "event_type"},
	)
	m.KafkaMessagesProduced = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "kafka",
			Name:      "messages_produced_total",
			Help:      "Total Produced Kafka messages",
		},
		[]string{"topic", "event_type"},
	)

	m.KafkaMessagesFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "kafka",
			Name:      "messages_failed_total",
			Help:      "Total failed Kafka message processing",
		},
		[]string{"topic", "event_type"},
	)

	m.KafkaProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: service,
			Subsystem: "kafka",
			Name:      "processing_duration_seconds",
			Help:      "Kafka message processing duration",
			Buckets:   []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"topic", "event_type"},
	)

	m.registry.MustRegister(
		m.KafkaMessagesConsumed,
		m.KafkaMessagesFailed,
		m.KafkaProcessingDuration,
		m.KafkaMessagesProduced,
	)
}
