package grpcorder

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"order-billing-system/order_service/internal/domain/model"
	pb "order-billing-system/order_service/internal/proto/pb"
	"order-billing-system/shared/money"
)

func mapProtoItemsDomain(items []*pb.OrderItem) []models.OrderItem {
	result := make([]models.OrderItem, 0, len(items))

	for _, item := range items {
		result = append(result, models.OrderItem{
			ProductID: item.ProductId,
			Quantity:  int(item.Quantity),
			Price: money.Money{
				Amount:   item.Price.Amount,
				Currency: item.Price.Currency,
			},
		})
	}
	return result
}
func mapProtoItemsProto(order models.Order) *pb.Order {
	items := make([]*pb.OrderItem, 0, len(order.Items))

	for _, item := range order.Items {
		items = append(items, &pb.OrderItem{
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			Price: &pb.Money{
				Amount:   item.Price.Amount,
				Currency: item.Price.Currency,
			},
		})
	}

	return &pb.Order{
		Id:    order.ID,
		Items: items,
		TotalPrice: &pb.Money{
			Amount:   order.Total.Amount,
			Currency: order.Total.Currency,
		},
		Status: mapOrderStatusToProto(order.Status),
		CreatedAt: timestamppb.New(order.CreatedAt),
	}
}

func mapOrderStatusToProto(status models.OrderStatus) pb.OrderStatus {
	switch status {
	case models.OrderStatusCreated:
		return pb.OrderStatus_ORDER_STATUS_CREATED
	case models.OrderStatusPaid:
		return pb.OrderStatus_ORDER_STATUS_PAID
	case models.OrderStatusCancelled:
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}
