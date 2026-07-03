package redis

import (
	"context"
)

// PresenceRepository handles user presence data in Redis.
type PresenceRepository struct {
	// TODO: add *redis.Client
}

// NewPresenceRepository creates a new presence repository.
func NewPresenceRepository() *PresenceRepository {
	return &PresenceRepository{}
}

// SetOnline marks a user as online.
func (r *PresenceRepository) SetOnline(ctx context.Context, userID string) error {
	return nil
}

// SetOffline marks a user as offline.
func (r *PresenceRepository) SetOffline(ctx context.Context, userID string) error {
	return nil
}

// IsOnline checks if a user is currently online.
func (r *PresenceRepository) IsOnline(ctx context.Context, userID string) (bool, error) {
	return false, nil
}
