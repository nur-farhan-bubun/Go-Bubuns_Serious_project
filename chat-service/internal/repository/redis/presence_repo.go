package redis

import (
	"context"

	"github.com/ride-sharing/chat-service/internal/domain"
)

// PresenceRepository handles user presence data in Redis.
type PresenceRepository struct {
	// TODO: add *redis.Client
}

// NewPresenceRepository creates a new presence repository.
func NewPresenceRepository() *PresenceRepository {
	return &PresenceRepository{}
}

// SetPresence updates a user's presence status.
func (r *PresenceRepository) SetPresence(ctx context.Context, presence *domain.Presence) error {
	return nil
}

// GetPresence retrieves a user's presence status.
func (r *PresenceRepository) GetPresence(ctx context.Context, userID string) (*domain.Presence, error) {
	return nil, nil
}

// GetOnlineUsers returns user IDs who are currently online.
func (r *PresenceRepository) GetOnlineUsers(ctx context.Context) ([]string, error) {
	return nil, nil
}
