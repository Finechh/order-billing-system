package outbox

import (
	"context"

	"gorm.io/gorm"
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) SaveTx(ctx context.Context, tx *gorm.DB, event *OutboxEvent) error {
	return tx.WithContext(ctx).Create(event).Error
}

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]OutboxEvent, error) {
	var events []OutboxEvent
	err := r.db.WithContext(ctx).Where("status = ?", EventStatusPending).Order("created_at ASC").Limit(limit).Find(&events).Error
	return events, err
}

func (r *OutboxRepository) MarkSent(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&OutboxEvent{}).Where("id = ?", id).Updates(map[string]any{"status": EventStatusSent, "processed_at": gorm.Expr("NOW()")}).Error
}
