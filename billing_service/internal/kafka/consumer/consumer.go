package consumer

import (
	"context"
	"fmt"
	"time"

	"order-billing-system/billing_service/internal/kafka/reliability"
	billing_repo "order-billing-system/billing_service/internal/repository"
	"order-billing-system/shared/logger"
	"order-billing-system/shared/middleware"
	"order-billing-system/shared/observability"
	"order-billing-system/shared/requestid"

	"github.com/segmentio/kafka-go"
)

type MessageHandler interface {
	Handle(ctx context.Context, msg kafka.Message) error
}

type Consumer struct {
	reader        *kafka.Reader
	handler       MessageHandler
	dlq           *reliability.DLQProducer
	processedRepo billing_repo.ProcessedEventRepository
	metrics       *observability.Metrics
}

func NewConsumer(topic string, brokers []string, groupID string, handler MessageHandler, dlq *reliability.DLQProducer, processedRepo billing_repo.ProcessedEventRepository, metrics *observability.Metrics) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  time.Second * 1,
	})
	return &Consumer{reader: reader, handler: handler, dlq: dlq, processedRepo: processedRepo, metrics: metrics}
}

func (c *Consumer) Start(ctx context.Context) {
	topic := c.reader.Config().Topic
	logger.InfoCtx(ctx, fmt.Sprintf("starting kafka consumer for topic: %s", topic))

	for {
		select {
		case <-ctx.Done():
			logger.InfoCtx(ctx, fmt.Sprintf("consumer for topic %s shutting down", topic))
			return
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.ErrorCtx(ctx, fmt.Sprintf("kafka fetch error on topic %s", topic), err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		c.processMessage(ctx, msg)
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
	start := time.Now()

	orderID := extractHeader(msg, "order-id")
	eventType := extractHeader(msg, "event-type")

	var eventID string
	if orderID != "" && eventType != "" {
		eventID = fmt.Sprintf("%s-%s", eventType, orderID)
	} else {
		eventID = fmt.Sprintf("%s-%d-%d", msg.Topic, msg.Partition, msg.Offset)
		logger.InfoCtx(ctx, fmt.Sprintf("message missing headers, using offset-based key: %s", eventID))
	}

	exists, err := c.processedRepo.Exists(ctx, eventID)
	if err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("idempotency check failed for %s, processing anyway", eventID), err)
	}
	if exists {
		logger.InfoCtx(ctx, fmt.Sprintf("event already processed: %s, skipping", eventID))
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			logger.ErrorCtx(ctx, "failed to commit already-processed message", err)
		}
		return
	}

	reqID := extractHeader(msg, "x-request-id")
	msgCtx := requestid.Set(ctx, reqID)

	err = reliability.Retry(msgCtx, 3, func() error {
		return middleware.SafeHandle(msgCtx, msg, c.handler.Handle)
	})

	duration := time.Since(start).Seconds()

	if err != nil {
		c.metrics.KafkaMessagesFailed.WithLabelValues(msg.Topic, eventType).Inc()
		c.metrics.KafkaProcessingDuration.WithLabelValues(msg.Topic, eventType).Observe(duration)

		logger.ErrorCtx(msgCtx, fmt.Sprintf(
			"failed to process message after retries, sending to DLQ. topic=%s event_type=%s event_id=%s",
			msg.Topic, eventType, eventID,
		), err)

		c.dlq.Send(msgCtx, reliability.DLQMessage{
			OriginalTopic: msg.Topic,
			EventType:     eventType,
			Payload:       string(msg.Value),
			Error:         err.Error(),
			FailedAt:      time.Now().Unix(),
		})

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			logger.ErrorCtx(msgCtx, "failed to commit failed message after DLQ", commitErr)
		}
		return
	}

	if err := c.processedRepo.Save(msgCtx, eventID, msg.Topic); err != nil {
		logger.ErrorCtx(msgCtx, fmt.Sprintf("failed to save processed event %s", eventID), err)
		return
	}

	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		logger.ErrorCtx(msgCtx, "failed to commit kafka message", err)
	}

	c.metrics.KafkaMessagesConsumed.WithLabelValues(msg.Topic, eventType).Inc()
	c.metrics.KafkaProcessingDuration.WithLabelValues(msg.Topic, eventType).Observe(duration)

	logger.InfoCtx(msgCtx, fmt.Sprintf(
		"message processed successfully. topic=%s event_type=%s event_id=%s duration=%.3fs",
		msg.Topic, eventType, eventID, duration,
	))
}

func extractHeader(msg kafka.Message, key string) string {
	for _, h := range msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *Consumer) Close() error {
	topic := c.reader.Config().Topic
	logger.InfoCtx(context.Background(), fmt.Sprintf("closing consumer for topic: %s", topic))
	return c.reader.Close()
}
