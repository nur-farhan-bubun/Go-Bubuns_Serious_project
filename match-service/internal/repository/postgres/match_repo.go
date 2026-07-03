package postgres

import (
	"context"

	"github.com/ride-sharing/match-service/internal/domain"
)

// Repository handles match data access.
type Repository struct {
	// TODO: add *pgxpool.Pool
}

// New creates a new match repository.
func New() *Repository {
	return &Repository{}
}

// CreateSwipe records a swipe action.
func (r *Repository) CreateSwipe(ctx context.Context, swipe *domain.Swipe) error {
	return nil
}

// CheckMatch checks if target_id has also swiped right on user_id.
func (r *Repository) CheckMatch(ctx context.Context, userID, targetID string) (bool, error) {
	return false, nil
}

// CreateMatch records a mutual match.
func (r *Repository) CreateMatch(ctx context.Context, match *domain.Match) error {
	return nil
}

// GetMatchesByUserID returns all matches for a user.
func (r *Repository) GetMatchesByUserID(ctx context.Context, userID string) ([]*domain.Match, error) {
	return nil, nil
}

// GetDiscoverProfiles returns profiles available for discovery.
func (r *Repository) GetDiscoverProfiles(ctx context.Context, userID string, page, pageSize int) ([]string, int, error) {
	return nil, 0, nil
}
