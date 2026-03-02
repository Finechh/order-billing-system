package events

import (
	"order-billing-system/shared/money"
	"time"
)

type OrderItem struct {
	ProductID string      `json:"product_id"`
	Quantity  int         `json:"quantity"`
	Price     money.Money `json:"price"`
}

type OrderCreatedEvent struct {
	OrderID   string      `json:"order_id"`
	Total     money.Money `json:"total"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderPaidEvent struct {
	OrderID string    `json:"order_id"`
	PaidAt  int64  `json:"paid_at"`
}

type OrderCancelledEvent struct {
	OrderID     string `json:"order_id"`
	CancelledAt  int64  `json:"cancelled_at"`
}

type ProcessedEvent struct {
	EventID     string `json:"event_id" gorm:"primaryKey;size:255"`
	Topic       string `json:"topic" gorm:"size:255;not null"`
	ProcessedAt int64  `json:"processed_at" gorm:"not null"`
}
