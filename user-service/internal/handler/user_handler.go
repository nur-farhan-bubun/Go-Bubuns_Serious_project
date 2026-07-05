package handler

import (
	"context"

	"github.com/ride-sharing/user-service/internal/domain"
)

// UserService defines the service contract used by the application handlers.
type UserService interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.UserResponse, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) (*domain.DeleteUserResponse, error)
	List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error)
}
