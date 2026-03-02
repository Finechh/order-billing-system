package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) initBusinessMetrics(service string) {
	m.InvoicesCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "invoices_created_total",
			Help:      "Total created invoices",
		},
	)

	m.InvoicesPaidTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "invoices_paid_total",
			Help:      "Total paid invoices",
		},
	)

	m.InvoicesCancelledTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "invoices_cancelled_total",
			Help:      "Total cancelled invoices",
		},
	)

	m.InvoiceStateTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "invoice_state_total",
			Help:      "Current number of invoices per state",
		},
		[]string{"state"},
	)

	m.OrdersCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "orders_created_total",
			Help:      "Total created orders",
		},
	)
	m.OrdersPaidTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "orders_paid_total",
			Help:      "Total paid orders",
		},
	)
	m.OrdersCancelledTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: service,
			Subsystem: "business",
			Name:      "orders_cancelled_total",
			Help:      "Total cancelled orders",
		},
	)
	m.registry.MustRegister(
		m.InvoicesCreatedTotal,
		m.InvoicesPaidTotal,
		m.InvoicesCancelledTotal,
		m.InvoiceStateTotal,
		m.OrdersCreatedTotal,
		m.OrdersPaidTotal,
		m.OrdersCancelledTotal,
	)
}
