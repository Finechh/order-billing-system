package billing_repo

import (
	"context"
	"errors"
	i "order-billing-system/billing_service/internal/billing_domain/invoice"
	errorsx "order-billing-system/shared/errors"

	"gorm.io/gorm"
)

type BillingReadRepository interface {
	List(ctx context.Context, status *i.InvoiceStatus, limit int, offset int) ([]i.Invoice, error)
	GetStats(ctx context.Context) (*BillingStats, error)
	GetByOrderID(ctx context.Context, orderID string) (*i.Invoice, error)
}

type BillingStats struct {
	TotalInvoices     int64
	PaidInvoices      int64
	PendingInvoices   int64
	CancelledInvoices int64
	TotalRevenue      i.Money
}

type InvoiceReadRepository struct {
	db *gorm.DB
}

func NewInvoiceReadRepository(db *gorm.DB) *InvoiceReadRepository {
	return &InvoiceReadRepository{db: db}
}

func (r *InvoiceReadRepository) GetByOrderID(ctx context.Context, orderID string) (*i.Invoice, error) {
	var invoice i.Invoice
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&invoice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ErrNotFound("invoice not found for order " + orderID)
		}
		return nil, err
	}
	return &invoice, nil
}

func (r *InvoiceReadRepository) List(ctx context.Context, status *i.InvoiceStatus, limit int, offset int) ([]i.Invoice, error) {
	var invoices []i.Invoice

	query := r.db.WithContext(ctx).Model(&i.Invoice{}).Order("created_at DESC").Limit(limit).Offset(offset)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	err := query.Find(&invoices).Error
	if err != nil {
		return nil, err
	}

	return invoices, nil
}

func (r *InvoiceReadRepository) GetStats(ctx context.Context) (*BillingStats, error) {
	stats := &BillingStats{}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var result struct {
			Count int64
			Sum   int64
		}

		if err := tx.Model(&i.Invoice{}).
			Select("COUNT(*) as count, COALESCE(SUM((amount->>'amount')::bigint), 0) as sum").
			Where("status = ?", i.InvoiceStatusPaid).
			Scan(&result).Error; err != nil {
			return err
		}
		stats.PaidInvoices = result.Count
		stats.TotalRevenue = i.Money{Amount: result.Sum, Currency: "USD"}

		if err := tx.Model(&i.Invoice{}).Count(&stats.TotalInvoices).Error; err != nil {
			return err
		}

		if err := tx.Model(&i.Invoice{}).
			Where("status = ?", i.InvoiceStatusPending).
			Count(&stats.PendingInvoices).Error; err != nil {
			return err
		}

		if err := tx.Model(&i.Invoice{}).
			Where("status = ?", i.InvoiceStatusCancelled).
			Count(&stats.CancelledInvoices).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return stats, nil
}
