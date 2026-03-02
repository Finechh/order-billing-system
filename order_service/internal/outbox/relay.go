package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"order-billing-system/shared/logger"
)

type Relay struct {
	repo      *OutboxRepository
	writer    *kafka.Writer
	interval  time.Duration
	batchSize int
}

func NewRelay(repo *OutboxRepository, brokers []string, interval time.Duration, batchSize int) *Relay {
	if batchSize <= 0 {
		batchSize = 50
	}
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}
	return &Relay{
		repo:      repo,
		writer:    writer,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (r *Relay) Start(ctx context.Context) {
	logger.InfoCtx(ctx, "outbox relay started")
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.InfoCtx(ctx, "outbox relay stopping")
			if err := r.writer.Close(); err != nil {
				logger.ErrorCtx(context.Background(), "outbox relay: failed to close writer", err)
			}
			return
		case <-ticker.C:
			if err := r.processOnce(ctx); err != nil {
				logger.ErrorCtx(ctx, "outbox relay cycle error", err)
			}
		}
	}
}

func (r *Relay) processOnce(ctx context.Context) error {
	events, err := r.repo.FetchPending(ctx, r.batchSize)
	if err != nil {
		return fmt.Errorf("fetch pending: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	logger.InfoCtx(ctx, fmt.Sprintf("outbox relay: processing %d events", len(events)))

	for _, ev := range events {
		msg := kafka.Message{
			Topic: ev.Topic,
			Key:   []byte(ev.OrderID),
			Value: []byte(ev.Payload),
			Headers: []kafka.Header{
				{Key: "event-type", Value: []byte(ev.EventType)},
				{Key: "order-id", Value: []byte(ev.OrderID)},
			},
		}

		if err := r.writer.WriteMessages(ctx, msg); err != nil {
			logger.ErrorCtx(ctx, fmt.Sprintf("outbox relay: failed to publish event %s, will retry next tick", ev.ID), err)
			continue
		}

		if err := r.repo.MarkSent(ctx, ev.ID); err != nil {
			logger.ErrorCtx(ctx, fmt.Sprintf("outbox relay: failed to mark event %s as sent", ev.ID), err)
		}
	}
	return nil
}

func BuildPayload(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
