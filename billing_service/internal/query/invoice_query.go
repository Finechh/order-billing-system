package query

import (
	"context"
	i "order-billing-system/billing_service/internal/billing_domain/invoice"
	r "order-billing-system/billing_service/internal/repository"
)

type InvoiceQueryService struct {
	repo r.BillingReadRepository
}

func NewInvoiceQueryService(repo r.BillingReadRepository) *InvoiceQueryService {
	return &InvoiceQueryService{repo: repo}
}

func (q *InvoiceQueryService) GetByOrderID(ctx context.Context, orderID string) (*i.Invoice, error) {
	return q.repo.GetByOrderID(ctx, orderID)
}

func (q *InvoiceQueryService) List(ctx context.Context, status *i.InvoiceStatus, limit int, offset int) ([]i.Invoice, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return q.repo.List(ctx, status, limit, offset)
}

func (q *InvoiceQueryService) GetStats(ctx context.Context) (*r.BillingStats, error) {
	return q.repo.GetStats(ctx)
}
