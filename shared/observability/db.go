package observability

import(
	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) initDBMetrics(service string) {
    m.DBOpenConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Namespace: service,
            Subsystem: "db",
            Name:      "open_connections",
            Help:      "Number of open DB connections",
        },
    )

    m.DBInUseConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Namespace: service,
            Subsystem: "db",
            Name:      "in_use_connections",
            Help:      "Number of in-use DB connections",
        },
    )

    m.DBIdleConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Namespace: service,
            Subsystem: "db",
            Name:      "idle_connections",
            Help:      "Number of idle DB connections",
        },
    )

    m.registry.MustRegister(
        m.DBOpenConnections,
        m.DBInUseConnections,
        m.DBIdleConnections,
    )
}
