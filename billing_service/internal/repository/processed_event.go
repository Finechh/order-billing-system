package billing_repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"order-billing-system/billing_service/internal/billing_domain/events"
)

type ProcessedEventRepository interface {
	Exists(ctx context.Context, eventID string) (bool, error)
	Save(ctx context.Context, eventID string, topic string) error
}

type eventRepository struct {
	db *gorm.DB
}

func NewProcessedEventRepo(db *gorm.DB) *eventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Exists(ctx context.Context, eventID string) (bool, error) {
	var event events.ProcessedEvent

	err := r.db.WithContext(ctx).First(&event, "event_id = ?", eventID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *eventRepository) Save(ctx context.Context, eventID string, topic string) error {
	event := events.ProcessedEvent{
		EventID:     eventID,
		Topic:       topic,
		ProcessedAt: time.Now().Unix(),
	}
	return r.db.WithContext(ctx).Create(&event).Error
}
