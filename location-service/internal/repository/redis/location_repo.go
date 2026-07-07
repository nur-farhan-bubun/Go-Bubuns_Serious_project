package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ride-sharing/location-service/internal/domain"
)

// LocationRepository handles user location data in Redis GEO.
type LocationRepository struct {
	client *redis.Client
}

// NewLocationRepository creates a new location repository.
func NewLocationRepository(redisURL string) (*LocationRepository, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fall back to simple host:port format
		opts = &redis.Options{
			Addr: redisURL,
		}
	}

	client := redis.NewClient(opts)

	// Verify connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &LocationRepository{client: client}, nil
}

// Close closes the Redis connection.
func (r *LocationRepository) Close() error {
	return r.client.Close()
}

// SetLocation updates a user's location.
func (r *LocationRepository) SetLocation(ctx context.Context, loc *domain.Location) error {
	return r.client.GeoAdd(ctx, "user:locations", &redis.GeoLocation{
		Name:      loc.UserID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}).Err()
}

// GetLocation retrieves a user's current location.
func (r *LocationRepository) GetLocation(ctx context.Context, userID string) (*domain.Location, error) {
	pos, err := r.client.GeoPos(ctx, "user:locations", userID).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
	}
	if len(pos) == 0 || pos[0] == nil {
		return nil, nil
	}

	return &domain.Location{
		UserID:    userID,
		Latitude:  pos[0].Latitude,
		Longitude: pos[0].Longitude,
		UpdatedAt: time.Now(),
	}, nil
}
