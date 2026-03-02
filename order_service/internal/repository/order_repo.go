package repository

import (
	"context"
	"order-billing-system/order_service/internal/domain/model"

	"gorm.io/gorm"
)

type OrderRepositoryInterface interface {
	GetOrder(ctx context.Context, id string) (models.Order, error)
}

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) GetOrder(ctx context.Context, id string) (models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error
	return order, err
}
