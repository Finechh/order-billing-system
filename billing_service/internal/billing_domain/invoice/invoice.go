package invoice

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type InvoiceStatus string

const (
	InvoiceStatusPending   InvoiceStatus = "PENDING"
	InvoiceStatusPaid      InvoiceStatus = "PAID"
	InvoiceStatusCancelled InvoiceStatus = "CANCELLED"
)

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func (m Money) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *Money) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan Money")
	}
	return json.Unmarshal(bytes, m)
}

type Invoice struct {
	ID        string        `gorm:"primaryKey"`
	OrderID   string        `gorm:"uniqueIndex;not null"`
	Amount    Money         `gorm:"type:jsonb"` 
	Status    InvoiceStatus `gorm:"type:varchar(20);index"`
	CreatedAt time.Time
	PaidAt    *time.Time    
}