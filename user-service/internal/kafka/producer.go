package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	batchSize    = 10
	batchTimeout = 50 * time.Millisecond
)

// Producer publishes user lifecycle events to Kafka for eventual consistency
// with downstream services (chat-service, match-service, etc.).
type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

// NewProducer creates a new asynchronous Kafka producer for user events.
func NewProducer(brokers string, logger *slog.Logger) *Producer {
	addrs := parseBrokers(brokers)
	w := &kafka.Writer{
		Addr:         kafka.TCP(addrs...),
		Topic:        UserTopic,
		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
		Async:        true, // non-blocking writes
		RequiredAcks: kafka.RequireNone,
	}
	return &Producer{writer: w, logger: logger.With(slog.String("component", "user_kafka_producer"))}
}

// Publish enqueues a user event for asynchronous delivery to the user-events topic.
// The user ID is used as the message key so all events for the same user land
// on the same partition (preserving order).
func (p *Producer) Publish(ctx context.Context, event *UserEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.UserID),
		Value: data,
	})
}

// Close flushes pending messages and shuts down the writer.
func (p *Producer) Close() error {
	p.logger.Info("closing user kafka producer")
	return p.writer.Close()
}

func parseBrokers(s string) []string {
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}
