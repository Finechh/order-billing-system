package middlewaregrpc

import (
	"context"
	"time"

	"order-billing-system/shared/observability"
	"order-billing-system/shared/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func GRPCMetricsInterceptor(metrics *observability.Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()

		statusCode := "ok"
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code().String()
		}

		metrics.GRPCRequestsTotal.WithLabelValues(info.FullMethod, statusCode).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		if err != nil {
			logger.ErrorCtx(ctx, "gRPC request failed: "+info.FullMethod, err)
		} else {
			logger.InfoCtx(ctx, "gRPC request succeeded: "+info.FullMethod)
		}
		return resp, err
	}
}
