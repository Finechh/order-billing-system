package grpcorder

import (
	"context"
	"errors"

	"order-billing-system/order_service/internal/proto/pb"
	"order-billing-system/order_service/internal/service"
	errorsx "order-billing-system/shared/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderGRPCServer struct {
	pb.UnimplementedOrderServiceServer
	service service.OrderServiceInterface
}

func NewOrderGRPCServer(s service.OrderServiceInterface) *OrderGRPCServer {
	return &OrderGRPCServer{service: s}
}

func (s *OrderGRPCServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	domainItems := mapProtoItemsDomain(req.Items)

	order, err := s.service.CreateOrder(ctx, domainItems)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.CreateOrderResponse{Order: mapProtoItemsProto(order)}, nil
}

func (s *OrderGRPCServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	order, err := s.service.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.GetOrderResponse{Order: mapProtoItemsProto(order)}, nil
}

func (s *OrderGRPCServer) MarkOrderPaid(ctx context.Context, req *pb.MarkOrderPaidRequest) (*pb.MarkOrderPaidResponse, error) {
	order, err := s.service.MarkOrderPaid(ctx, req.OrderId)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.MarkOrderPaidResponse{Order: mapProtoItemsProto(order)}, nil
}

func (s *OrderGRPCServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	order, err := s.service.CancelOrder(ctx, req.OrderId)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.CancelOrderResponse{Order: mapProtoItemsProto(order)}, nil
}

func mapError(err error) error {
	var appErr *errorsx.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case "ERR_NOT_FOUND":
			return status.Error(codes.NotFound, appErr.Message)
		case "ERR_INVALID_INPUT":
			return status.Error(codes.InvalidArgument, appErr.Message)
		case "ERR_INVALID_ORDER_STATE":
			return status.Error(codes.FailedPrecondition, appErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
