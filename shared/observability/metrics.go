package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"order-billing-system/shared/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry

	serviceName string

	GRPCRequestsTotal   *prometheus.CounterVec
	GRPCRequestDuration *prometheus.HistogramVec

	KafkaMessagesConsumed   *prometheus.CounterVec
	KafkaMessagesProduced   *prometheus.CounterVec
	KafkaMessagesFailed     *prometheus.CounterVec
	KafkaProcessingDuration *prometheus.HistogramVec

	DBOpenConnections  prometheus.Gauge
	DBInUseConnections prometheus.Gauge
	DBIdleConnections  prometheus.Gauge

	InvoicesCreatedTotal   prometheus.Counter
	InvoicesPaidTotal      prometheus.Counter
	InvoicesCancelledTotal prometheus.Counter
	InvoiceStateTotal      *prometheus.GaugeVec

	OrdersCreatedTotal   prometheus.Counter
	OrdersPaidTotal      prometheus.Counter
	OrdersCancelledTotal prometheus.Counter
}

func NewMetrics(serviceName, service string) *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry:    registry,
		serviceName: serviceName,
	}

	m.initGRPCMetrics(service)
	m.initKafkaMetrics(service)
	m.initDBMetrics(service)
	m.initBusinessMetrics(service)
	return m
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) StartServer(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCtx(ctx, "http server failed", err)
		}
	}()
	logger.InfoCtx(ctx, fmt.Sprintf("Metrics server started on %s", addr))
	return nil
}
