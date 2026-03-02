package currency

import (
	"context"
	"fmt"
)

type Converter interface {
	ToUSD(ctx context.Context, amount int64, from string) (int64, error)
}

var staticRates = map[string]float64{
	"USD": 1.00,
	"EUR": 1.09,
	"RUB": 0.011,
}

type StaticConverter struct{}

func NewStaticConverter() *StaticConverter {
	return &StaticConverter{}
}

func (c *StaticConverter) ToUSD(_ context.Context, amount int64, from string) (int64, error) {
	if from == "USD" {
		return amount, nil
	}
	rate, ok := staticRates[from]
	if !ok {
		return 0, fmt.Errorf("unsupported currency: %s", from)
	}
	converted := int64(float64(amount) * rate)
	return converted, nil
}
