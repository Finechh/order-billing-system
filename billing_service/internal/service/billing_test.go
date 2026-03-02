package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-billing-system/billing_service/internal/billing_domain/events"
	"order-billing-system/billing_service/internal/billing_domain/invoice"
	"order-billing-system/billing_service/internal/service"
	r "order-billing-system/billing_service/internal/service/mocks"
	errorsx "order-billing-system/shared/errors"
	"order-billing-system/shared/money"
	"order-billing-system/shared/observability"
)

func newTestMetrics() *observability.Metrics {
	return observability.NewMetrics("test_billing", "billing")
}

func orderCreatedEvent() events.OrderCreatedEvent {
	return events.OrderCreatedEvent{OrderID: "order-abc", Total: money.Money{Amount: 2000, Currency: "USD"}, CreatedAt: time.Now()}
}

func TestHandleOrderCreated_Success(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	repo.On("Get", mock.Anything, "order-abc").Return(nil, errors.New("not found"))
	repo.On("Create", mock.Anything, mock.MatchedBy(func(inv *invoice.Invoice) bool {
		return inv.OrderID == "order-abc" &&
			inv.Status == invoice.InvoiceStatusPending &&
			inv.Amount.Amount == 2000
	})).Return(nil)

	err := svc.HandleOrderCreated(context.Background(), orderCreatedEvent())

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandleOrderCreated_Idempotent_AlreadyExists(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	existing := &invoice.Invoice{ID: "inv-1", OrderID: "order-abc", Status: invoice.InvoiceStatusPending}
	repo.On("Get", mock.Anything, "order-abc").Return(existing, nil)

	err := svc.HandleOrderCreated(context.Background(), orderCreatedEvent())

	require.NoError(t, err)
	repo.AssertNumberOfCalls(t, "Create", 0)
}

func TestHandleOrderCreated_RepoCreateError(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	repo.On("Get", mock.Anything, "order-abc").Return(nil, errors.New("not found"))
	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := svc.HandleOrderCreated(context.Background(), orderCreatedEvent())

	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INTERNAL", appErr.Code)
}
func TestHandleOrderPaid_Success(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	existing := &invoice.Invoice{ID: "inv-1", OrderID: "order-abc", Status: invoice.InvoiceStatusPending}
	repo.On("Get", mock.Anything, "order-abc").Return(existing, nil)
	repo.On("UpdateStatus", mock.Anything, "order-abc", invoice.InvoiceStatusPaid, mock.AnythingOfType("*time.Time")).
		Return(nil)

	event := events.OrderPaidEvent{OrderID: "order-abc", PaidAt: time.Now().Unix()}
	err := svc.HandleOrderPaid(context.Background(), event)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandleOrderPaid_InvoiceNotFound(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	repo.On("Get", mock.Anything, "order-abc").Return(nil, errors.New("not found"))

	event := events.OrderPaidEvent{OrderID: "order-abc", PaidAt: time.Now().Unix()}
	err := svc.HandleOrderPaid(context.Background(), event)

	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

func TestHandleOrderPaid_NotPendingStatus(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	existing := &invoice.Invoice{ID: "inv-1", OrderID: "order-abc", Status: invoice.InvoiceStatusPaid}
	repo.On("Get", mock.Anything, "order-abc").Return(existing, nil)

	event := events.OrderPaidEvent{OrderID: "order-abc", PaidAt: time.Now().Unix()}
	err := svc.HandleOrderPaid(context.Background(), event)

	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_ORDER_STATE", appErr.Code)
}

func TestHandleOrderCancelled_Success(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	existing := &invoice.Invoice{ID: "inv-1", OrderID: "order-abc", Status: invoice.InvoiceStatusPending}
	repo.On("Get", mock.Anything, "order-abc").Return(existing, nil)
	repo.On("UpdateStatus", mock.Anything, "order-abc", invoice.InvoiceStatusCancelled, (*time.Time)(nil)).
		Return(nil)

	event := events.OrderCancelledEvent{OrderID: "order-abc", CancelledAt: time.Now().Unix()}
	err := svc.HandleOrderCancelled(context.Background(), event)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandleOrderCancelled_AlreadyCancelled_Idempotent(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	existing := &invoice.Invoice{ID: "inv-1", OrderID: "order-abc", Status: invoice.InvoiceStatusCancelled}
	repo.On("Get", mock.Anything, "order-abc").Return(existing, nil)

	event := events.OrderCancelledEvent{OrderID: "order-abc", CancelledAt: time.Now().Unix()}
	err := svc.HandleOrderCancelled(context.Background(), event)

	require.NoError(t, err)
	repo.AssertNumberOfCalls(t, "UpdateStatus", 0)
}

func TestHandleOrderCancelled_PaidInvoice(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	existing := &invoice.Invoice{ID: "inv-1", OrderID: "order-abc", Status: invoice.InvoiceStatusPaid}
	repo.On("Get", mock.Anything, "order-abc").Return(existing, nil)

	event := events.OrderCancelledEvent{OrderID: "order-abc", CancelledAt: time.Now().Unix()}
	err := svc.HandleOrderCancelled(context.Background(), event)

	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_ORDER_STATE", appErr.Code)
}

func TestHandleOrderCancelled_InvoiceNotFound(t *testing.T) {
	repo := new(r.MockBillingRepo)
	svc := service.NewBillingService(repo, newTestMetrics())

	repo.On("Get", mock.Anything, "order-abc").Return(nil, errors.New("not found"))

	event := events.OrderCancelledEvent{OrderID: "order-abc", CancelledAt: time.Now().Unix()}
	err := svc.HandleOrderCancelled(context.Background(), event)

	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}
