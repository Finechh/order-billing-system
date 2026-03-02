package middleware

import (
	"context"

	"order-billing-system/shared/requestid"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,req interface{},info *grpc.UnaryServerInfo,handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		var reqID string
		if ok {
			values := md.Get("x-request-id")
			if len(values) > 0 {
				reqID = values[0]
			}
		}
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx = requestid.Set(ctx, reqID)
		return handler(ctx, req)
	}
}
