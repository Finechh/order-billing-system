package events

import (
	"order-billing-system/shared/money"
	"order-billing-system/order_service/internal/domain/model"
	"time"
)

type EventType string

const (
	EventOrderCreated   EventType = "OrderCreated"
	EventOrderPaid      EventType = "OrderPaid"
	EventOrderCancelled EventType = "OrderCancelled"
)

type OrderCreatedEvent struct {
	OrderID   string             `json:"order_id"`
	Total     money.Money       `json:"total"`
	Items     []models.OrderItem `json:"items"`
	CreatedAt time.Time          `json:"created_at"`
}

type OrderPaidEvent struct {
	OrderID string `json:"order_id"`
	PaidAt  int64  `json:"paid_at"`
}

type OrderCancelledEvent struct {
	OrderID     string `json:"order_id"`
	CancelledAt int64  `json:"cancelled_at"`
}
