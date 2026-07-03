package redis

import (
	"context"

	"github.com/ride-sharing/location-service/internal/domain"
)

// LocationRepository handles user location data in Redis GEO.
type LocationRepository struct {
	// TODO: add *redis.Client
}

// NewLocationRepository creates a new location repository.
func NewLocationRepository() *LocationRepository {
	return &LocationRepository{}
}

// SetLocation updates a user's location.
func (r *LocationRepository) SetLocation(ctx context.Context, loc *domain.Location) error {
	return nil
}

// GetNearby returns users within a given radius.
func (r *LocationRepository) GetNearby(ctx context.Context, lat, lng, radius float64) ([]*domain.NearbyUser, error) {
	return nil, nil
}

// GetLocation retrieves a user's current location.
func (r *LocationRepository) GetLocation(ctx context.Context, userID string) (*domain.Location, error) {
	return nil, nil
}
