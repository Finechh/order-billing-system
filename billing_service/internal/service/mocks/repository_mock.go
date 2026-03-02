package repository_test

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"order-billing-system/billing_service/internal/billing_domain/invoice"
)

type MockBillingRepo struct {
	mock.Mock
}

func (m *MockBillingRepo) Create(ctx context.Context, inv *invoice.Invoice) error {
	return m.Called(ctx, inv).Error(0)
}

func (m *MockBillingRepo) Get(ctx context.Context, orderID string) (*invoice.Invoice, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*invoice.Invoice), args.Error(1)
}

func (m *MockBillingRepo) UpdateStatus(ctx context.Context, orderID string, status invoice.InvoiceStatus, paidAt *time.Time) error {
	return m.Called(ctx, orderID, status, paidAt).Error(0)
}
