package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"order-billing-system/shared/logger"

	"github.com/segmentio/kafka-go"
)

func SafeHandle(ctx context.Context, msg kafka.Message, handler func(context.Context, kafka.Message) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			panicErr := fmt.Errorf("panic recovered in kafka handler: %v\n%s", r, stack)
			logger.ErrorCtx(ctx, "panic recovered in kafka handler", panicErr)
			err = panicErr
		}
	}()
	return handler(ctx, msg)
}
