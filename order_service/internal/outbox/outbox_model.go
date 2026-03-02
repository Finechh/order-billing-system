package outbox

import "time"

type EventStatus string

const (
	EventStatusPending EventStatus = "PENDING"
	EventStatusSent    EventStatus = "SENT"
)

type OutboxEvent struct {
	ID          string      `gorm:"primaryKey"`
	OrderID     string      `gorm:"index;not null"`
	EventType   string      `gorm:"not null"`
	Topic       string      `gorm:"not null"`
	Payload     string      `gorm:"type:text;not null"`
	Status      EventStatus `gorm:"index;default:'PENDING'"`
	CreatedAt   time.Time
	ProcessedAt *time.Time
}
