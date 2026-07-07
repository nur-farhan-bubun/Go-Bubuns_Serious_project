package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ride-sharing/location-service/internal/domain"
	redisrepo "github.com/ride-sharing/location-service/internal/repository/redis"
	"github.com/ride-sharing/location-service/internal/repository/postgres"
)

// LocationUpdateCallback is called when a user's location is updated.
// Used by the WebSocket hub to broadcast location changes in real-time.
type LocationUpdateCallback func(loc *domain.Location)

// Service handles location business logic.
type Service struct {
	locationRepo    *redisrepo.LocationRepository
	mapPostRepo     *postgres.MapPostRepository
	updateCallbacks []LocationUpdateCallback
}

// New creates a new location service.
func New(locationRepo *redisrepo.LocationRepository, mapPostRepo *postgres.MapPostRepository) *Service {
	return &Service{
		locationRepo: locationRepo,
		mapPostRepo:  mapPostRepo,
	}
}

// OnLocationUpdate registers a callback for location updates.
func (s *Service) OnLocationUpdate(cb LocationUpdateCallback) {
	s.updateCallbacks = append(s.updateCallbacks, cb)
}

// notifyUpdate triggers all registered callbacks.
func (s *Service) notifyUpdate(loc *domain.Location) {
	for _, cb := range s.updateCallbacks {
		cb(loc)
	}
}

// ─── Real-time Location ────────────────────────────────────────────────

// UpdateLocation updates a user's location and notifies subscribers.
func (s *Service) UpdateLocation(ctx context.Context, loc *domain.Location) error {
	if err := s.locationRepo.SetLocation(ctx, loc); err != nil {
		return err
	}
	// Notify WebSocket subscribers in a goroutine so the REST call returns immediately
	go s.notifyUpdate(loc)
	return nil
}

// GetLocation retrieves a user's current location.
func (s *Service) GetLocation(ctx context.Context, userID string) (*domain.Location, error) {
	loc, err := s.locationRepo.GetLocation(ctx, userID)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		loc = &domain.Location{
			UserID:    userID,
			Latitude:  0,
			Longitude: 0,
			UpdatedAt: time.Now(),
		}
	}
	return loc, nil
}

// ─── Map Posts (PostGIS) ───────────────────────────────────────────────

// CreatePost creates a new map post.
func (s *Service) CreatePost(ctx context.Context, post *domain.MapPost) (*domain.MapPost, error) {
	post.ID = uuid.New().String()
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()
	return s.mapPostRepo.Create(ctx, post)
}

// GetPost retrieves a map post by ID.
func (s *Service) GetPost(ctx context.Context, id string) (*domain.MapPost, error) {
	return s.mapPostRepo.GetByID(ctx, id)
}

// ListPosts returns map posts, optionally filtered by location.
func (s *Service) ListPosts(ctx context.Context, lat, lng, radius float64, category string) ([]*domain.MapPost, error) {
	if lat != 0 && lng != 0 && radius > 0 {
		return s.mapPostRepo.ListByLocation(ctx, lat, lng, radius, category)
	}
	if category != "" {
		// Filter by category without location — fetch all then filter
		all, err := s.mapPostRepo.ListAll(ctx)
		if err != nil {
			return nil, err
		}
		filtered := make([]*domain.MapPost, 0)
		for _, p := range all {
			if p.Category == category {
				filtered = append(filtered, p)
			}
		}
		return filtered, nil
	}
	return s.mapPostRepo.ListAll(ctx)
}

// UpdatePost updates a map post.
func (s *Service) UpdatePost(ctx context.Context, post *domain.MapPost) (*domain.MapPost, error) {
	post.UpdatedAt = time.Now()
	return s.mapPostRepo.Update(ctx, post)
}

// DeletePost deletes a map post.
func (s *Service) DeletePost(ctx context.Context, id, userID string) error {
	return s.mapPostRepo.Delete(ctx, id, userID)
}
