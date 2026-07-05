package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/ride-sharing/chat-service/internal/domain"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

// Consumer reads chat events from Kafka and fans them out to locally-connected
// WebSocket clients via the Hub. Each instance uses a unique consumer group ID
// so that every pod receives every message (fan-out / broadcast pattern).
type Consumer struct {
	reader *kafka.Reader
	hub    *ws.Hub
	logger *slog.Logger
}

// NewConsumer creates a Kafka consumer with a dynamic, short-lived consumer
// group ID (UUID per instance). This guarantees every horizontally-scaled pod
// receives every message from the topic.
func NewConsumer(brokers string, hub *ws.Hub, logger *slog.Logger) *Consumer {
	addrs := parseBrokers(brokers)
	groupID := fmt.Sprintf("chat-service-%s", uuid.NewString())

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     addrs,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset, // only new messages
		MinBytes:    1,
		MaxBytes:    10e6, // 10 MB
		MaxWait:     100 * time.Millisecond,
	})

	return &Consumer{
		reader: r,
		hub:    hub,
		logger: logger.With(
			slog.String("component", "kafka_consumer"),
			slog.String("group_id", groupID),
		),
	}
}

// Run starts the blocking consumption loop. It should be called in a
// goroutine. When ctx is cancelled (graceful shutdown) the loop exits.
//
// The consumer follows the non-blocking hand-off pattern from Phase 4.3:
// as soon as a message arrives it is unmarshalled, marshalled exactly once
// into a WSEnvelope, and handed to the Hub's SendToRoom — no database
// writes or heavy processing happen inside this loop.
func (c *Consumer) Run(ctx context.Context) {
	c.logger.Info("kafka consumer started")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.logger.Info("kafka consumer exiting")
				return
			}
			c.logger.Error("kafka read error, retrying in 1s", slog.String("error", err.Error()))
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
			continue
		}

		var msg Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			c.logger.Error("failed to unmarshal kafka message",
				slog.String("error", err.Error()),
				slog.Int64("offset", m.Offset),
			)
			continue
		}

		// Reconstruct the WSEnvelope and marshal it **once** for fan-out.
		envelope := &domain.WSEnvelope{
			Type:   msg.Type,
			RoomID: msg.RoomID,
			Data:   msg.Data,
		}

		payload, err := json.Marshal(envelope)
		if err != nil {
			c.logger.Error("failed to marshal fan-out envelope",
				slog.String("error", err.Error()),
			)
			continue
		}

		// Non-blocking hand-off to the Hub's sharded room map.
		c.hub.SendToRoom(msg.RoomID, payload)
	}
}

// Close shuts down the reader, committing the final offset and leaving the
// consumer group so the group rebalance completes quickly.
func (c *Consumer) Close() error {
	c.logger.Info("closing kafka consumer")
	return c.reader.Close()
}
