package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	"order-billing-system/order_service/internal/config"
	"order-billing-system/order_service/internal/database"
	"order-billing-system/order_service/internal/domain/events"
	"order-billing-system/order_service/internal/grpc"
	"order-billing-system/order_service/internal/outbox"
	"order-billing-system/order_service/internal/proto/pb"
	"order-billing-system/order_service/internal/repository"
	"order-billing-system/order_service/internal/service"
	"order-billing-system/shared/currency"
	"order-billing-system/shared/logger"
	"order-billing-system/shared/middleware"
	"order-billing-system/shared/middleware_grpc"
	"order-billing-system/shared/observability"
)

func Run() error {
	logger.Init("order_service")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.InfoCtx(ctx, "starting order service")

	cfg := config.LoadConfig()

	db, err := database.InitDB(cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := db.AutoMigrate(&outbox.OutboxEvent{}); err != nil {
		return fmt.Errorf("failed to migrate outbox table: %w", err)
	}
	logger.InfoCtx(ctx, "database connected and migrated")

	metrics := observability.NewMetrics("order_service", "order")
	if err := metrics.StartServer(ctx, ":9094"); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	go collectDBMetrics(ctx, db, metrics)

	topicMap := map[events.EventType]string{
		events.EventOrderCreated:   cfg.OrderCreatedTopic,
		events.EventOrderPaid:      cfg.OrderPaidTopic,
		events.EventOrderCancelled: cfg.OrderCancelledTopic,
	}

	orderRepo := repository.NewOrderRepository(db)
	outboxRepo := outbox.NewOutboxRepository(db)
	converter := currency.NewStaticConverter()
	orderService := service.NewOrderService(db, orderRepo, outboxRepo, topicMap, converter, metrics)

	relay := outbox.NewRelay(outboxRepo, cfg.KafkaBrokers, 2*time.Second, cfg.OutboxBatchSize)
	go relay.Start(ctx)
	logger.InfoCtx(ctx, "outbox relay started (polling every 2s)")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("failed to listen on :50051: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RequestIDInterceptor(),
			middleware.ValidationInterceptor(),
			middleware.RecoveryInterceptor(),
			middlewaregrpc.GRPCMetricsInterceptor(metrics),
		),
	)

	pbServer := grpcorder.NewOrderGRPCServer(orderService)
	pb.RegisterOrderServiceServer(grpcServer, pbServer)
	reflection.Register(grpcServer)

	go func() {
		logger.InfoCtx(ctx, "order gRPC server started on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			logger.ErrorCtx(ctx, "gRPC server failed", err)
		}
	}()

	<-ctx.Done()
	logger.InfoCtx(context.Background(), "shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

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

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	logger.InfoCtx(context.Background(), "order service stopped")
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
				logger.ErrorCtx(ctx, "failed to get sql.DB for metrics", err)
				continue
			}
			stats := sqlDB.Stats()
			metrics.DBOpenConnections.Set(float64(stats.OpenConnections))
			metrics.DBInUseConnections.Set(float64(stats.InUse))
			metrics.DBIdleConnections.Set(float64(stats.Idle))
		}
	}
}
