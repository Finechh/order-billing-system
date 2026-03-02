package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"order-billing-system/order_service/internal/domain/events"
	models "order-billing-system/order_service/internal/domain/model"
	r "order-billing-system/order_service/internal/service/mocks"
	"order-billing-system/shared/currency"
	errorsx "order-billing-system/shared/errors"
	"order-billing-system/shared/money"
	"order-billing-system/shared/observability"
)

func newTestMetrics() *observability.Metrics {
	return observability.NewMetrics("test_order", "order")
}

func newSvc(repo *r.MockOrderRepo) *OrderService {
	return &OrderService{
		db:         nil,
		repo:       repo,
		outboxRepo: nil,
		topicMap:   nil,
		converter:  currency.NewStaticConverter(),
		metrics:    newTestMetrics(),
	}
}

func validItemsUSD() []models.OrderItem {
	return []models.OrderItem{
		{ProductID: "prod-1", Quantity: 2, Price: money.Money{Amount: 500, Currency: "USD"}},
		{ProductID: "prod-2", Quantity: 1, Price: money.Money{Amount: 1000, Currency: "USD"}},
	}
}

func TestCalcTotalUSD_SingleCurrency(t *testing.T) {
	items := []models.OrderItem{
		{ProductID: "p1", Quantity: 2, Price: money.Money{Amount: 500, Currency: "USD"}},
		{ProductID: "p2", Quantity: 1, Price: money.Money{Amount: 1000, Currency: "USD"}},
	}
	total, err := calcTotalUSD(context.Background(), items, currency.NewStaticConverter())
	require.NoError(t, err)
	assert.Equal(t, int64(2000), total)
}

func TestCalcTotalUSD_MixedCurrencies(t *testing.T) {
	items := []models.OrderItem{
		{ProductID: "p1", Quantity: 1, Price: money.Money{Amount: 1000, Currency: "RUB"}},
		{ProductID: "p2", Quantity: 2, Price: money.Money{Amount: 100, Currency: "EUR"}},
	}
	total, err := calcTotalUSD(context.Background(), items, currency.NewStaticConverter())
	require.NoError(t, err)
	assert.Equal(t, int64(229), total)
}

func TestCalcTotalUSD_UnknownCurrency(t *testing.T) {
	items := []models.OrderItem{
		{ProductID: "p1", Quantity: 1, Price: money.Money{Amount: 100, Currency: "GBP"}},
	}
	_, err := calcTotalUSD(context.Background(), items, currency.NewStaticConverter())
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestCreateOrder_EmptyItems(t *testing.T) {
	svc := newSvc(nil)
	_, err := svc.CreateOrder(context.Background(), []models.OrderItem{})
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestCreateOrder_MissingProductID(t *testing.T) {
	svc := newSvc(nil)
	items := []models.OrderItem{
		{ProductID: "", Quantity: 1, Price: money.Money{Amount: 100, Currency: "USD"}},
	}
	_, err := svc.CreateOrder(context.Background(), items)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestCreateOrder_ZeroQuantity(t *testing.T) {
	svc := newSvc(nil)
	items := []models.OrderItem{
		{ProductID: "p1", Quantity: 0, Price: money.Money{Amount: 100, Currency: "USD"}},
	}
	_, err := svc.CreateOrder(context.Background(), items)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestCreateOrder_UnsupportedCurrency(t *testing.T) {
	svc := newSvc(nil)
	items := []models.OrderItem{
		{ProductID: "p1", Quantity: 1, Price: money.Money{Amount: 100, Currency: "GBP"}},
	}
	_, err := svc.CreateOrder(context.Background(), items)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestCreateOrder_MixedCurrencies_FailsOnWrite(t *testing.T) {
	svc := newSvc(nil)
	items := []models.OrderItem{
		{ProductID: "p1", Quantity: 1, Price: money.Money{Amount: 1000, Currency: "RUB"}},
		{ProductID: "p2", Quantity: 1, Price: money.Money{Amount: 100, Currency: "EUR"}},
	}

	defer func() {
		recover()
	}()

	_, err := svc.CreateOrder(context.Background(), items)
	if err != nil {
		var appErr *errorsx.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, "ERR_INTERNAL", appErr.Code)
	}
}

func TestGetOrder_Success(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	expected := models.Order{ID: "order-123", Status: models.OrderStatusCreated}
	repo.On("GetOrder", mock.Anything, "order-123").Return(expected, nil)

	order, err := svc.GetOrder(context.Background(), "order-123")
	require.NoError(t, err)
	assert.Equal(t, "order-123", order.ID)
	repo.AssertExpectations(t)
}

func TestGetOrder_EmptyID(t *testing.T) {
	svc := newSvc(nil)
	_, err := svc.GetOrder(context.Background(), "")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestGetOrder_NotFound(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	repo.On("GetOrder", mock.Anything, "missing").Return(models.Order{}, errors.New("record not found"))

	_, err := svc.GetOrder(context.Background(), "missing")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

func TestMarkOrderPaid_AlreadyPaid(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	existing := models.Order{ID: "order-1", Status: models.OrderStatusPaid}
	repo.On("GetOrder", mock.Anything, "order-1").Return(existing, nil)

	_, err := svc.MarkOrderPaid(context.Background(), "order-1")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_ORDER_STATE", appErr.Code)
}

func TestMarkOrderPaid_CancelledOrder(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	existing := models.Order{ID: "order-1", Status: models.OrderStatusCancelled}
	repo.On("GetOrder", mock.Anything, "order-1").Return(existing, nil)

	_, err := svc.MarkOrderPaid(context.Background(), "order-1")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_ORDER_STATE", appErr.Code)
}

func TestMarkOrderPaid_EmptyID(t *testing.T) {
	svc := newSvc(nil)
	_, err := svc.MarkOrderPaid(context.Background(), "")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestMarkOrderPaid_NotFound(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	repo.On("GetOrder", mock.Anything, "x").Return(models.Order{}, errors.New("not found"))

	_, err := svc.MarkOrderPaid(context.Background(), "x")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

func TestCancelOrder_PaidOrder(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	existing := models.Order{ID: "order-1", Status: models.OrderStatusPaid}
	repo.On("GetOrder", mock.Anything, "order-1").Return(existing, nil)

	_, err := svc.CancelOrder(context.Background(), "order-1")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_ORDER_STATE", appErr.Code)
}

func TestCancelOrder_AlreadyCancelled_Idempotent(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	existing := models.Order{ID: "order-1", Status: models.OrderStatusCancelled}
	repo.On("GetOrder", mock.Anything, "order-1").Return(existing, nil)

	order, err := svc.CancelOrder(context.Background(), "order-1")
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusCancelled, order.Status)

	repo.AssertNumberOfCalls(t, "GetOrder", 1) 
}

func TestCancelOrder_EmptyID(t *testing.T) {
	svc := newSvc(nil)
	_, err := svc.CancelOrder(context.Background(), "")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestCancelOrder_NotFound(t *testing.T) {
	repo := new(r.MockOrderRepo)
	svc := newSvc(repo)

	repo.On("GetOrder", mock.Anything, "gone").Return(models.Order{}, errors.New("not found"))

	_, err := svc.CancelOrder(context.Background(), "gone")
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

var _ events.EventType = events.EventOrderCreated
