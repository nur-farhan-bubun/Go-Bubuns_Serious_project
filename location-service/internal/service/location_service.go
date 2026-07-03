package service

import (
	"context"

	"github.com/ride-sharing/location-service/internal/domain"
	redisrepo "github.com/ride-sharing/location-service/internal/repository/redis"
)

// Service handles location business logic.
type Service struct {
	locationRepo *redisrepo.LocationRepository
	presenceRepo *redisrepo.PresenceRepository
}

// New creates a new location service.
func New(locationRepo *redisrepo.LocationRepository, presenceRepo *redisrepo.PresenceRepository) *Service {
	return &Service{locationRepo: locationRepo, presenceRepo: presenceRepo}
}

// UpdateLocation updates a user's location.
func (s *Service) UpdateLocation(ctx context.Context, loc *domain.Location) error {
	return s.locationRepo.SetLocation(ctx, loc)
}

// GetNearby returns nearby users.
func (s *Service) GetNearby(ctx context.Context, lat, lng, radius float64) ([]*domain.NearbyUser, error) {
	return s.locationRepo.GetNearby(ctx, lat, lng, radius)
}

// SetOnline marks a user as online.
func (s *Service) SetOnline(ctx context.Context, userID string) error {
	return s.presenceRepo.SetOnline(ctx, userID)
}

// SetOffline marks a user as offline.
func (s *Service) SetOffline(ctx context.Context, userID string) error {
	return s.presenceRepo.SetOffline(ctx, userID)
}
