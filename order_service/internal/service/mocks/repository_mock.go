package repository_test

import (
	"context"

	"github.com/stretchr/testify/mock"

	"order-billing-system/order_service/internal/domain/model"
)

type MockOrderRepo struct {
	mock.Mock
}

func (m *MockOrderRepo) GetOrder(ctx context.Context, id string) (models.Order, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.Order), args.Error(1)
}
