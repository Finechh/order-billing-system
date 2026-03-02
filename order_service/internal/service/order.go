package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"order-billing-system/order_service/internal/domain/events"
	models "order-billing-system/order_service/internal/domain/model"
	"order-billing-system/order_service/internal/outbox"
	"order-billing-system/order_service/internal/repository"
	"order-billing-system/shared/currency"
	errorsx "order-billing-system/shared/errors"
	"order-billing-system/shared/logger"
	"order-billing-system/shared/money"
	"order-billing-system/shared/observability"
)

type OrderServiceInterface interface {
	CreateOrder(ctx context.Context, items []models.OrderItem) (models.Order, error)
	GetOrder(ctx context.Context, id string) (models.Order, error)
	MarkOrderPaid(ctx context.Context, id string) (models.Order, error)
	CancelOrder(ctx context.Context, id string) (models.Order, error)
}

type OrderService struct {
	db         *gorm.DB
	repo       repository.OrderRepositoryInterface
	outboxRepo *outbox.OutboxRepository
	converter  currency.Converter
	metrics    *observability.Metrics
	topicMap   map[events.EventType]string
}

func NewOrderService(db *gorm.DB, r repository.OrderRepositoryInterface, outboxRepo *outbox.OutboxRepository, topicMap map[events.EventType]string, converter currency.Converter, metrics *observability.Metrics) *OrderService {
	return &OrderService{db: db, repo: r, outboxRepo: outboxRepo, topicMap: topicMap, converter: converter, metrics: metrics}
}

var allowedCurrencies = map[string]bool{
	"USD": true,
	"EUR": true,
	"RUB": true,
}

func validateItems(items []models.OrderItem) error {
	if len(items) == 0 {
		return errorsx.ErrInvalidInput("order must have at least one item")
	}
	for i, item := range items {
		if item.ProductID == "" {
			return errorsx.ErrInvalidInput(fmt.Sprintf("item %d: product_id is required", i))
		}
		if item.Quantity <= 0 {
			return errorsx.ErrInvalidInput(fmt.Sprintf("item %d: quantity must be positive", i))
		}
		if item.Price.Amount <= 0 {
			return errorsx.ErrInvalidInput(fmt.Sprintf("item %d: price must be positive", i))
		}
		if !allowedCurrencies[item.Price.Currency] {
			return errorsx.ErrInvalidInput(
				fmt.Sprintf("item %d: unsupported currency %s (allowed: USD, EUR, RUB)", i, item.Price.Currency),
			)
		}
	}
	return nil
}

func (s *OrderService) writeWithOutbox(ctx context.Context, order models.Order, ev any, evType events.EventType) error {
	topic, ok := s.topicMap[evType]
	if !ok {
		return errorsx.ErrInternalError(fmt.Sprintf("unknown event type: %s", evType))
	}
	payload, err := outbox.BuildPayload(ev)
	if err != nil {
		return errorsx.ErrInternalError("failed to serialize event")
	}
	outboxEvent := &outbox.OutboxEvent{
		ID:        uuid.NewString(),
		OrderID:   order.ID,
		EventType: string(evType),
		Topic:     topic,
		Payload:   payload,
		Status:    outbox.EventStatusPending,
		CreatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return s.outboxRepo.SaveTx(ctx, tx, outboxEvent)
	})
}

func (s *OrderService) CreateOrder(ctx context.Context, items []models.OrderItem) (models.Order, error) {
	if err := validateItems(items); err != nil {
		return models.Order{}, err
	}

	totalUSD, err := calcTotalUSD(ctx, items, s.converter)
	if err != nil {
		return models.Order{}, err
	}

	order := models.Order{
		ID:        uuid.NewString(),
		Items:     items,
		Total:     money.Money{Amount: totalUSD, Currency: "USD"},
		Status:    models.OrderStatusCreated,
		CreatedAt: time.Now(),
	}

	event := events.OrderCreatedEvent{
		OrderID:   order.ID,
		Total:     order.Total,
		Items:     order.Items,
		CreatedAt: order.CreatedAt,
	}

	if err := s.writeWithOutbox(ctx, order, event, events.EventOrderCreated); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to persist order %s with outbox", order.ID), err)
		return models.Order{}, errorsx.ErrInternalError("failed to create order")
	}

	logger.InfoCtx(ctx, fmt.Sprintf("order %s created, total %d USD, outbox event queued", order.ID, totalUSD))
	s.metrics.OrdersCreatedTotal.Inc()
	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (models.Order, error) {
	if id == "" {
		return models.Order{}, errorsx.ErrInvalidInput("order_id is required")
	}
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return models.Order{}, errorsx.ErrNotFound("order not found")
	}
	return order, nil
}

func (s *OrderService) MarkOrderPaid(ctx context.Context, id string) (models.Order, error) {
	if id == "" {
		return models.Order{}, errorsx.ErrInvalidInput("order_id is required")
	}
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return models.Order{}, errorsx.ErrNotFound("order not found")
	}
	if order.Status != models.OrderStatusCreated {
		return models.Order{}, errorsx.ErrInvalidOrderState(
			fmt.Sprintf("cannot pay order with status %s, only CREATED orders can be paid", order.Status),
		)
	}
	order.Status = models.OrderStatusPaid
	event := events.OrderPaidEvent{OrderID: order.ID, PaidAt: time.Now().Unix()}
	if err := s.writeWithOutbox(ctx, order, event, events.EventOrderPaid); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to update order %s to PAID", id), err)
		return models.Order{}, errorsx.ErrInternalError("failed to update order")
	}
	logger.InfoCtx(ctx, fmt.Sprintf("order %s marked PAID, outbox event queued", id))
	s.metrics.OrdersPaidTotal.Inc()
	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id string) (models.Order, error) {
	if id == "" {
		return models.Order{}, errorsx.ErrInvalidInput("order_id is required")
	}
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return models.Order{}, errorsx.ErrNotFound("order not found")
	}
	if order.Status == models.OrderStatusPaid {
		return models.Order{}, errorsx.ErrInvalidOrderState("cannot cancel a paid order")
	}
	if order.Status == models.OrderStatusCancelled {
		return order, nil
	}
	order.Status = models.OrderStatusCancelled
	event := events.OrderCancelledEvent{OrderID: order.ID, CancelledAt: time.Now().Unix()}
	if err := s.writeWithOutbox(ctx, order, event, events.EventOrderCancelled); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to update order %s to CANCELLED", id), err)
		return models.Order{}, errorsx.ErrInternalError("failed to update order")
	}
	logger.InfoCtx(ctx, fmt.Sprintf("order %s cancelled, outbox event queued", id))
	s.metrics.OrdersCancelledTotal.Inc()
	return order, nil
}
