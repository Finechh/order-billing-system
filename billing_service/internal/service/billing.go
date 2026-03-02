package service

import (
	"context"
	"fmt"
	"time"

	"order-billing-system/billing_service/internal/billing_domain/events"
	i "order-billing-system/billing_service/internal/billing_domain/invoice"
	"order-billing-system/shared/observability"
	r "order-billing-system/billing_service/internal/repository"
	errorsx "order-billing-system/shared/errors"
	"order-billing-system/shared/logger"

	"github.com/google/uuid"
)

type BillingServiceInterface interface {
	HandleOrderCreated(ctx context.Context, event events.OrderCreatedEvent) error
	HandleOrderPaid(ctx context.Context, event events.OrderPaidEvent) error
	HandleOrderCancelled(ctx context.Context, event events.OrderCancelledEvent) error
}

type BillingService struct {
	repo    r.BillingRepositoryInterface
	metrics *observability.Metrics
}

func NewBillingService(repo r.BillingRepositoryInterface, metrics *observability.Metrics) *BillingService {
	return &BillingService{repo: repo, metrics: metrics}
}

func (s *BillingService) HandleOrderCreated(ctx context.Context, event events.OrderCreatedEvent) error {
	logger.InfoCtx(ctx, fmt.Sprintf("handling OrderCreated for order %s", event.OrderID))

	existing, err := s.repo.Get(ctx, event.OrderID)
	if err == nil && existing != nil {
		logger.InfoCtx(ctx, fmt.Sprintf("invoice already exists for order %s, skipping", event.OrderID))
		return nil 
	}

	invoice := &i.Invoice{
		ID:        uuid.NewString(),
		OrderID:   event.OrderID,
		Amount:    i.Money(event.Total),
		Status:    i.InvoiceStatusPending,
		CreatedAt: event.CreatedAt,
	}

	if err := s.repo.Create(ctx, invoice); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to create invoice for order %s", event.OrderID), err)
		return errorsx.ErrInternalError("failed to create invoice")
	}

	s.metrics.InvoicesCreatedTotal.Inc()
	s.metrics.InvoiceStateTotal.WithLabelValues("pending").Inc()

	logger.InfoCtx(ctx, fmt.Sprintf("invoice %s created for order %s, amount: %d %s",
		invoice.ID, invoice.OrderID, invoice.Amount.Amount, invoice.Amount.Currency))
	return nil
}

func (s *BillingService) HandleOrderPaid(ctx context.Context, event events.OrderPaidEvent) error {
	logger.InfoCtx(ctx, fmt.Sprintf("handling OrderPaid for order %s", event.OrderID))

	invoice, err := s.repo.Get(ctx, event.OrderID)
	if err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("invoice not found for order %s", event.OrderID), err)
		return errorsx.ErrNotFound("invoice not found")
	}

	if invoice.Status != i.InvoiceStatusPending {
		logger.InfoCtx(ctx, fmt.Sprintf("invoice already in status %s for order %s, cannot mark paid", invoice.Status, event.OrderID))
		return errorsx.ErrInvalidOrderState("invoice cannot be paid: not in PENDING status")
	}

	paidAt := time.Unix(event.PaidAt, 0)

	if err := s.repo.UpdateStatus(ctx, event.OrderID, i.InvoiceStatusPaid, &paidAt); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to update invoice to PAID for order %s", event.OrderID), err)
		return errorsx.ErrInternalError("failed to update invoice status")
	}

	s.metrics.InvoicesPaidTotal.Inc()
	s.metrics.InvoiceStateTotal.WithLabelValues("pending").Dec()
	s.metrics.InvoiceStateTotal.WithLabelValues("paid").Inc()

	logger.InfoCtx(ctx, fmt.Sprintf("invoice %s marked PAID, order %s, paidAt %s",
		invoice.ID, event.OrderID, paidAt.Format(time.RFC3339)))
	return nil
}

func (s *BillingService) HandleOrderCancelled(ctx context.Context, event events.OrderCancelledEvent) error {
	logger.InfoCtx(ctx, fmt.Sprintf("handling OrderCancelled for order %s", event.OrderID))

	invoice, err := s.repo.Get(ctx, event.OrderID)
	if err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("invoice not found for order %s", event.OrderID), err)
		return errorsx.ErrNotFound("invoice not found")
	}

	if invoice.Status == i.InvoiceStatusPaid {
		logger.InfoCtx(ctx, fmt.Sprintf("cannot cancel PAID invoice for order %s", event.OrderID))
		return errorsx.ErrInvalidOrderState("paid invoice cannot be cancelled")
	}

	if invoice.Status == i.InvoiceStatusCancelled {
		return nil
	}

	if invoice.Status != i.InvoiceStatusPending {
		logger.InfoCtx(ctx, fmt.Sprintf("unexpected invoice status %s for order %s, skipping", invoice.Status, event.OrderID))
		return errorsx.ErrInvalidOrderState(fmt.Sprintf("cannot cancel invoice in status %s", invoice.Status))
	}

	if err := s.repo.UpdateStatus(ctx, event.OrderID, i.InvoiceStatusCancelled, nil); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to update invoice to CANCELLED for order %s", event.OrderID), err)
		return errorsx.ErrInternalError("failed to update invoice status")
	}

	s.metrics.InvoicesCancelledTotal.Inc()
	s.metrics.InvoiceStateTotal.WithLabelValues("pending").Dec()
	s.metrics.InvoiceStateTotal.WithLabelValues("cancelled").Inc()

	logger.InfoCtx(ctx, fmt.Sprintf("invoice %s cancelled for order %s", invoice.ID, event.OrderID))
	return nil
}