package reliability

import (
	"context"
	"math/rand"
	"time"
)

func Retry(ctx context.Context, attempts int, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			base := time.Duration(1<<uint(i)) * time.Second
			jitter := time.Duration(rand.Int63n(int64(base / 5)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(base + jitter):
			}
		}
	}
	return err
}
