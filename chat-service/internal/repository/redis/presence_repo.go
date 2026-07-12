package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ride-sharing/chat-service/internal/domain"
)

const (
	presencePrefix     = "presence:"
	presenceOnlineKey  = "presence:online"
	presenceTTL        = 2 * time.Minute
)

// PresenceRepository handles user presence data in Redis.
type PresenceRepository struct {
	client *redis.Client
}

// NewPresenceRepository creates a new presence repository and connects to Redis.
func NewPresenceRepository(ctx context.Context, redisURL string) (*PresenceRepository, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// If no protocol, treat as host:port with no password
		opts = &redis.Options{
			Addr:     redisURL,
			DB:       0,
			Password: "",
		}
	}

	client := redis.NewClient(opts)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &PresenceRepository{client: client}, nil
}

// Close shuts down the Redis client.
func (r *PresenceRepository) Close() error {
	return r.client.Close()
}

// SetPresence updates a user's presence status.
// Uses both:
//  1. HSET presence:online userID timestamp — for fast online user lookups
//  2. SET presence:userID {json} — for full presence data retrieval
func (r *PresenceRepository) SetPresence(ctx context.Context, presence *domain.Presence) error {
	// HSET for the online hash (fast lookup of who's online)
	now := time.Now().Unix()
	if err := r.client.HSet(ctx, presenceOnlineKey, presence.UserID, now).Err(); err != nil {
		return fmt.Errorf("failed to hset presence online: %w", err)
	}

	// Set expiry on the hash so stale entries are cleaned up
	if err := r.client.Expire(ctx, presenceOnlineKey, presenceTTL).Err(); err != nil {
		return fmt.Errorf("failed to set expiry on presence:online: %w", err)
	}

	// Full presence JSON for detailed retrieval
	key := presencePrefix + presence.UserID
	data, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("failed to marshal presence: %w", err)
	}

	return r.client.Set(ctx, key, data, presenceTTL).Err()
}

// GetPresence retrieves a user's presence status.
func (r *PresenceRepository) GetPresence(ctx context.Context, userID string) (*domain.Presence, error) {
	key := presencePrefix + userID

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return &domain.Presence{
				UserID:   userID,
				Status:   "offline",
				LastSeen: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get presence: %w", err)
	}

	var presence domain.Presence
	if err := json.Unmarshal(data, &presence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal presence: %w", err)
	}

	return &presence, nil
}

// GetOnlineUsers returns user IDs who are currently online.
func (r *PresenceRepository) GetOnlineUsers(ctx context.Context) ([]string, error) {
	iter := r.client.Scan(ctx, 0, presencePrefix+"*", 1000).Iterator()

	var users []string
	for iter.Next(ctx) {
		key := iter.Val()
		// Strip prefix to get user ID
		if len(key) > len(presencePrefix) {
			userID := key[len(presencePrefix):]
			users = append(users, userID)
		}
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan online users: %w", err)
	}

	return users, nil
}
