package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"order-billing-system/shared/money"
	"time"
)

type OrderItem struct {
	ProductID string      `json:"product_id"`
	Quantity  int         `json:"quantity"`
	Price     money.Money `json:"price"`
}

type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "CREATED"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type JSONItems []OrderItem

func (j JSONItems) Value() (driver.Value, error) {
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONItems) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONItems")
	}
	return json.Unmarshal(bytes, j)
}

type Order struct {
	ID        string      `gorm:"primaryKey"`
	Items     JSONItems   `gorm:"type:json"`
	Total     money.Money `gorm:"serializer:json" json:"total"`
	Status    OrderStatus `gorm:"index"`
	CreatedAt time.Time   `gorm:"index"`
}
