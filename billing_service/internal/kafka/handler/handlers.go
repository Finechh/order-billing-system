package handlers

import (
	"context"
	"encoding/json"
	"time"

	"order-billing-system/billing_service/internal/billing_domain/events"
	"order-billing-system/billing_service/internal/kafka/reliability"
	"order-billing-system/billing_service/internal/service"
	"order-billing-system/shared/logger"

	"github.com/segmentio/kafka-go"
)

type TopicHandlerFunc func(ctx context.Context, msg kafka.Message) error

type OrderHandler struct {
	service    service.BillingServiceInterface
	dlq        *reliability.DLQProducer
	topicHandlers map[string]TopicHandlerFunc
}

func NewOrderHandler(s service.BillingServiceInterface, dlq *reliability.DLQProducer, topicCreated, topicPaid, topicCancelled string) *OrderHandler {
	h := &OrderHandler{service: s, dlq: dlq}
	h.topicHandlers = map[string]TopicHandlerFunc{
		topicCreated:   h.handleCreated,
		topicPaid:      h.handlePaid,
		topicCancelled: h.handleCancelled,
	}
	return h
}

func (h *OrderHandler) Handle(ctx context.Context, msg kafka.Message) error {
	logger.InfoCtx(ctx, "received kafka message on topic "+msg.Topic)
	fn, ok := h.topicHandlers[msg.Topic]
	if !ok {
		logger.InfoCtx(ctx, "unknown topic: "+msg.Topic+", skipping")
		return nil
	}
	return fn(ctx, msg)
}

func unmarshal[T any](ctx context.Context, h *OrderHandler, msg kafka.Message) (T, bool) {
	var event T
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		logger.ErrorCtx(ctx, "failed to unmarshal kafka message, sending to DLQ", err)
		h.sendToDLQ(ctx, msg, err)
		return event, false
	}
	return event, true
}

func (h *OrderHandler) handleCreated(ctx context.Context, msg kafka.Message) error {
	event, ok := unmarshal[events.OrderCreatedEvent](ctx, h, msg)
	if !ok {
		return nil
	}
	if event.OrderID == "" {
		logger.InfoCtx(ctx, "received OrderCreated with empty order_id, sending to DLQ")
		h.sendToDLQ(ctx, msg, nil)
		return nil
	}
	return h.service.HandleOrderCreated(ctx, event)
}

func (h *OrderHandler) handlePaid(ctx context.Context, msg kafka.Message) error {
	event, ok := unmarshal[events.OrderPaidEvent](ctx, h, msg)
	if !ok {
		return nil
	}
	if event.OrderID == "" {
		logger.InfoCtx(ctx, "received OrderPaid with empty order_id, sending to DLQ")
		h.sendToDLQ(ctx, msg, nil)
		return nil
	}
	return h.service.HandleOrderPaid(ctx, event)
}

func (h *OrderHandler) handleCancelled(ctx context.Context, msg kafka.Message) error {
	event, ok := unmarshal[events.OrderCancelledEvent](ctx, h, msg)
	if !ok {
		return nil
	}
	if event.OrderID == "" {
		logger.InfoCtx(ctx, "received OrderCancelled with empty order_id, sending to DLQ")
		h.sendToDLQ(ctx, msg, nil)
		return nil
	}
	return h.service.HandleOrderCancelled(ctx, event)
}

func (h *OrderHandler) sendToDLQ(ctx context.Context, msg kafka.Message, err error) {
	dlqMsg := reliability.DLQMessage{
		OriginalTopic: msg.Topic,
		Payload:       string(msg.Value),
		FailedAt:      time.Now().Unix(),
	}
	if err != nil {
		dlqMsg.Error = err.Error()
	}
	h.dlq.Send(ctx, dlqMsg)
}
