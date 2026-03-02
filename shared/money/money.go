package money

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func (m Money) Value() (driver.Value, error) {
	return fmt.Sprintf("%d|%s", m.Amount, m.Currency), nil
}

func (m *Money) Scan(value interface{}) error {
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return errors.New("invalid type for Money: expected string or []byte")
	}

	parts := strings.Split(str, "|")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format for Money: %q (expected amount|currency)", str)
	}

	amount, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount in Money: %w", err)
	}

	m.Amount = amount
	m.Currency = parts[1]
	return nil
}
