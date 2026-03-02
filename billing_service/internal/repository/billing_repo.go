package billing_repo

import (
	"context"
	"errors"
	"fmt"
	"order-billing-system/billing_service/internal/billing_domain/invoice"
	"order-billing-system/shared/logger"
	"time"

	"gorm.io/gorm"
)

type BillingRepositoryInterface interface {
	Create(ctx context.Context, inv *invoice.Invoice) error
	Get(ctx context.Context, invoiceID string) (*invoice.Invoice, error)
	UpdateStatus(ctx context.Context, orderID string, status invoice.InvoiceStatus, paidAt *time.Time) error
}

type BillingRepository struct {
	db *gorm.DB
}

func NewBillingRepository(db *gorm.DB) *BillingRepository {
	return &BillingRepository{db: db}
}

func (b *BillingRepository) Create(ctx context.Context, inv *invoice.Invoice) error {
	if err := b.db.WithContext(ctx).Create(inv).Error; err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("failed to create invoice for order_id=%s", inv.OrderID), err)
		return err
	}
	return nil
}

func (b *BillingRepository) Get(ctx context.Context, orderID string) (*invoice.Invoice, error) {
	var invoice invoice.Invoice
	err := b.db.WithContext(ctx).Where("order_id = ?", orderID).First(&invoice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	return &invoice, nil
}

func (b *BillingRepository) UpdateStatus(ctx context.Context, orderID string, status invoice.InvoiceStatus, paidAt *time.Time) error {
	updates := map[string]any{"status": status}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	result := b.db.WithContext(ctx).Model(&invoice.Invoice{}).Where("order_id = ?", orderID).Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update invoice: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("invoice not found")
	}
	return nil
}
