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
	topic           = "chat-messages"
	LifecycleTopic  = "chat-lifecycle"
	batchSize       = 100
	batchTimeout    = 10 * time.Millisecond
)

// Producer is an asynchronous Kafka writer that batches messages for
// high-throughput chat event publishing.
type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

// NewProducer creates a new asynchronous Kafka producer with the batching
// parameters prescribed by Phase 4.1 of the architecture spec.
//
// The writer runs in async mode so WriteMessages returns immediately after
// enqueuing; actual network writes are batched in the background.
func NewProducer(brokers string, logger *slog.Logger) *Producer {
	addrs := parseBrokers(brokers)
	// Do NOT set a default Topic on the Writer — the topic is set on each
	// individual Message instead (see Publish vs PublishToTopic).
	// This avoids "Topic must not be specified for both Writer and Message".
	w := &kafka.Writer{
		Addr:         kafka.TCP(addrs...),
		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
		Async:        true, // non-blocking writes
		RequiredAcks: kafka.RequireNone,
	}
	return &Producer{writer: w, logger: logger.With(slog.String("component", "kafka_producer"))}
}

// Publish enqueues a chat event for asynchronous delivery to the default
// chat-messages topic. The room ID is used as the message key so all
// messages from the same room land on the same partition.
func (p *Producer) Publish(ctx context.Context, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(msg.RoomID),
		Value: data,
	})
}

// PublishToTopic enqueues a chat event for asynchronous delivery to a
// specific Kafka topic. The room ID is used as the message key so all
// messages from the same room land on the same partition (preserving order).
func (p *Producer) PublishToTopic(ctx context.Context, topic string, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(msg.RoomID),
		Value: data,
	})
}

// Close flushes pending messages and shuts down the writer.
func (p *Producer) Close() error {
	p.logger.Info("closing kafka producer")
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
