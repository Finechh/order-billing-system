package service

import (
	"context"
	"fmt"

	"order-billing-system/order_service/internal/domain/model"
	"order-billing-system/shared/currency"
	errorsx "order-billing-system/shared/errors"
)

func calcTotalUSD(ctx context.Context, items []models.OrderItem, conv currency.Converter) (int64, error) {
	var total int64
	for i, item := range items {
		lineAmount := item.Price.Amount * int64(item.Quantity)
		usd, err := conv.ToUSD(ctx, lineAmount, item.Price.Currency)
		if err != nil {
			return 0, errorsx.ErrInvalidInput(
				fmt.Sprintf("item %d: cannot convert %s to USD: %s", i, item.Price.Currency, err.Error()),
			)
		}
		total += usd
	}
	return total, nil
}
