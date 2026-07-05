package handler

import (
	"context"

	"github.com/ride-sharing/user-service/internal/domain"
)

// AuthService defines the service contract used by the application handlers.
type AuthService interface {
	GenerateGoogleLoginURL(state string) string
	HandleGoogleCallback(ctx context.Context, code string) (*domain.AuthResponse, error)
}
