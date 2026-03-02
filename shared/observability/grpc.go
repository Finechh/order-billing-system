package observability

import(
	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) initGRPCMetrics(service string) {
    m.GRPCRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: service,
            Subsystem: "grpc",
            Name:      "requests_total",
            Help:      "Total number of gRPC requests",
        },
        []string{"method", "status"},
    )

    m.GRPCRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: service,
            Subsystem: "grpc",
            Name:      "request_duration_seconds",
            Help:      "gRPC request latency",
            Buckets:   prometheus.DefBuckets,
        },
        []string{"method"},
    )

    m.registry.MustRegister(
        m.GRPCRequestsTotal,
        m.GRPCRequestDuration,
    )
}
