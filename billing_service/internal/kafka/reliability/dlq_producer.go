package reliability

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"order-billing-system/shared/logger"
)

type DLQMessage struct {
	OriginalTopic string `json:"original_topic"`
	EventType     string `json:"event_type,omitempty"`
	Payload       string `json:"payload"`
	Error         string `json:"error,omitempty"`
	FailedAt      int64  `json:"failed_at"`
}

type DLQProducer struct {
	writer *kafka.Writer
	topic  string
}

func NewDLQProducer(brokers []string, topic string) *DLQProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &DLQProducer{writer: writer, topic: topic}
}

func (p *DLQProducer) Send(ctx context.Context, msg DLQMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.ErrorCtx(ctx, "DLQ: failed to marshal message", err)
		return
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		logger.ErrorCtx(ctx, fmt.Sprintf("DLQ: failed to send message to topic %s", p.topic), err)
		return
	}
	logger.InfoCtx(ctx, fmt.Sprintf("DLQ: message sent to %s, original_topic=%s", p.topic, msg.OriginalTopic))
}

func (p *DLQProducer) Close() error {
	logger.InfoCtx(context.Background(), fmt.Sprintf("DLQ: closing producer for topic %s", p.topic))
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close DLQ writer: %w", err)
	}
	return nil
}
