package app

import (
	"context"
	"fmt"
	"net"
	"order-billing-system/billing_service/internal/config"
	database "order-billing-system/billing_service/internal/database"
	billinggrpc "order-billing-system/billing_service/internal/grpc"
	"order-billing-system/billing_service/internal/kafka/consumer"
	handlers "order-billing-system/billing_service/internal/kafka/handler"
	"order-billing-system/billing_service/internal/kafka/reliability"
	"order-billing-system/billing_service/internal/proto/pb"
	query "order-billing-system/billing_service/internal/query"
	r "order-billing-system/billing_service/internal/repository"
	"order-billing-system/billing_service/internal/service"
	"order-billing-system/shared/logger"
	"order-billing-system/shared/middleware"
	middlewaregrpc "order-billing-system/shared/middleware_grpc"
	"order-billing-system/shared/observability"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

func Run() error {
	logger.Init("billing_service")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.InfoCtx(ctx, "Starting Billing Service...")

	cfg := config.LoadConfig()
	db, err := database.InitDB(cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.InfoCtx(ctx, "Database connected successfully")

	metrics := observability.NewMetrics("billing_service", "billing")
	if err := metrics.StartServer(ctx, ":9093"); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	go collectDBMetrics(ctx, db, metrics)

	billingRepo := r.NewBillingRepository(db)
	invoiceReadRepo := r.NewInvoiceReadRepository(db)
	processedRepo := r.NewProcessedEventRepo(db)

	billingService := service.NewBillingService(billingRepo, metrics)
	queryService := query.NewInvoiceQueryService(invoiceReadRepo)
	grpcHandler := billinggrpc.NewBillingQueryHandler(queryService)

	dlqProducer := reliability.NewDLQProducer(cfg.KafkaBrokers, "billing-dlq-topic")
	eventHandler := handlers.NewOrderHandler(billingService, dlqProducer, cfg.OrderCreatedTopic, cfg.OrderPaidTopic, cfg.OrderCancelledTopic)

	createdConsumer := consumer.NewConsumer(cfg.OrderCreatedTopic, cfg.KafkaBrokers, "billing-group-created", eventHandler, dlqProducer, processedRepo, metrics)
	paidConsumer := consumer.NewConsumer(cfg.OrderPaidTopic, cfg.KafkaBrokers, "billing-group-paid", eventHandler, dlqProducer, processedRepo, metrics)
	cancelledConsumer := consumer.NewConsumer(cfg.OrderCancelledTopic, cfg.KafkaBrokers, "billing-group-cancelled", eventHandler, dlqProducer, processedRepo, metrics)

	go createdConsumer.Start(ctx)
	go paidConsumer.Start(ctx)
	go cancelledConsumer.Start(ctx)
	logger.InfoCtx(ctx, "Kafka consumers started")

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		return fmt.Errorf("failed to listen on :50052: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RequestIDInterceptor(),
			middleware.ValidationInterceptor(),
			middleware.RecoveryInterceptor(),
			middlewaregrpc.GRPCMetricsInterceptor(metrics),
		))

	pb.RegisterBillingQueryServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	go func() {
		logger.InfoCtx(ctx, "Billing gRPC server started on :50052")
		if err := grpcServer.Serve(lis); err != nil {
			logger.ErrorCtx(ctx, "gRPC server failed", err)
		}
	}()

	<-ctx.Done()
	logger.InfoCtx(context.Background(), "Shutdown signal received, starting graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := createdConsumer.Close(); err != nil {
		logger.ErrorCtx(context.Background(), "failed to close created consumer", err)
	}
	if err := paidConsumer.Close(); err != nil {
		logger.ErrorCtx(context.Background(), "failed to close paid consumer", err)
	}
	if err := cancelledConsumer.Close(); err != nil {
		logger.ErrorCtx(context.Background(), "failed to close cancelled consumer", err)
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.InfoCtx(context.Background(), "gRPC server stopped gracefully")
	case <-shutdownCtx.Done():
		logger.InfoCtx(context.Background(), "shutdown timeout, forcing stop")
		grpcServer.Stop()
	}

	if err := dlqProducer.Close(); err != nil {
		logger.ErrorCtx(context.Background(), "failed to close DLQ producer", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	logger.InfoCtx(context.Background(), "Billing service stopped cleanly")
	return nil
}

func collectDBMetrics(ctx context.Context, db *gorm.DB, metrics *observability.Metrics) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sqlDB, err := db.DB()
			if err != nil {
				logger.ErrorCtx(ctx, "Failed to get sql.DB for metrics", err)
				continue
			}
			stats := sqlDB.Stats()
			metrics.DBOpenConnections.Set(float64(stats.OpenConnections))
			metrics.DBInUseConnections.Set(float64(stats.InUse))
			metrics.DBIdleConnections.Set(float64(stats.Idle))
		}
	}
}
