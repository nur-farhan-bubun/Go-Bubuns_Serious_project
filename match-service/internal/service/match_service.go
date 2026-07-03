package service

import (
	"context"

	"github.com/ride-sharing/match-service/internal/domain"
	"github.com/ride-sharing/match-service/internal/repository/postgres"
)

// Service handles match business logic.
type Service struct {
	repo *postgres.Repository
}

// New creates a new match service.
func New(repo *postgres.Repository) *Service {
	return &Service{repo: repo}
}

// Swipe handles a swipe action and returns a match if mutual.
func (s *Service) Swipe(ctx context.Context, userID, targetID, direction string) (*domain.Match, error) {
	return nil, nil
}

// GetMatches returns all matches for a user.
func (s *Service) GetMatches(ctx context.Context, userID string) ([]*domain.Match, error) {
	return s.repo.GetMatchesByUserID(ctx, userID)
}

// GetDiscover returns discoverable profiles.
func (s *Service) GetDiscover(ctx context.Context, userID string, page, pageSize int) ([]string, int, error) {
	return s.repo.GetDiscoverProfiles(ctx, userID, page, pageSize)
}
