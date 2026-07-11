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
)

// UserStore defines the contract for storing/retrieving cached user info.
type UserStore interface {
	Upsert(ctx context.Context, user *domain.UserInfo) error
	GetByID(ctx context.Context, userID string) (*domain.UserInfo, error)
	List(ctx context.Context) ([]*domain.UserInfo, error)
	Delete(ctx context.Context, userID string) error
}

// UserConsumer reads user lifecycle events from Kafka and keeps the local
// user cache eventually consistent. Each instance uses a unique consumer
// group ID so every pod receives every message (fan-out pattern).
type UserConsumer struct {
	reader    *kafka.Reader
	userStore UserStore
	logger    *slog.Logger
}

// NewUserConsumer creates a Kafka consumer for the user-events topic.
// Each instance gets a unique consumer group ID for fan-out delivery.
func NewUserConsumer(brokers string, userStore UserStore, logger *slog.Logger) *UserConsumer {
	addrs := parseBrokers(brokers)
	groupID := fmt.Sprintf("chat-service-users-%s", uuid.NewString())

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     addrs,
		Topic:       UserTopic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset, // only new messages
		MinBytes:    1,
		MaxBytes:    10e6, // 10 MB
		MaxWait:     100 * time.Millisecond,
	})

	logger.Info("user kafka consumer created",
		slog.String("brokers", brokers),
		slog.Any("parsed_addrs", addrs),
		slog.String("group_id", groupID),
	)

	return &UserConsumer{
		reader:    r,
		userStore: userStore,
		logger:    logger.With(slog.String("component", "user_kafka_consumer"), slog.String("topic", UserTopic)),
	}
}

// Run starts the blocking consumption loop. It should be called in a
// goroutine. When ctx is cancelled (graceful shutdown) the loop exits.
func (c *UserConsumer) Run(ctx context.Context) {
	c.logger.Info("user kafka consumer started")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.logger.Info("user kafka consumer exiting")
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

		var event UserEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			c.logger.Error("failed to unmarshal user event",
				slog.String("error", err.Error()),
				slog.Int64("offset", m.Offset),
			)
			continue
		}

		c.processEvent(ctx, &event)
	}
}

func (c *UserConsumer) processEvent(ctx context.Context, event *UserEvent) {
	switch event.Type {
	case UserCreated, UserUpdated:
		user := &domain.UserInfo{
			UserID:      event.UserID,
			Email:       event.Email,
			DisplayName: event.DisplayName,
			AvatarURL:   event.AvatarURL,
		}
		if err := c.userStore.Upsert(ctx, user); err != nil {
			c.logger.Error("failed to upsert user in cache",
				slog.String("user_id", event.UserID),
				slog.String("error", err.Error()),
			)
			return
		}
		c.logger.Info("user cached",
			slog.String("user_id", event.UserID),
			slog.String("display_name", event.DisplayName),
			slog.String("event_type", string(event.Type)),
		)

	case UserDeleted:
		if err := c.userStore.Delete(ctx, event.UserID); err != nil {
			c.logger.Error("failed to delete user from cache",
				slog.String("user_id", event.UserID),
				slog.String("error", err.Error()),
			)
			return
		}
		c.logger.Info("user removed from cache",
			slog.String("user_id", event.UserID),
		)

	default:
		c.logger.Warn("unknown user event type",
			slog.String("type", string(event.Type)),
		)
	}
}

// Close shuts down the reader.
func (c *UserConsumer) Close() error {
	c.logger.Info("closing user kafka consumer")
	return c.reader.Close()
}
