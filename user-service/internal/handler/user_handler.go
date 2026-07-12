package handler

import (
	"context"

	"github.com/ride-sharing/user-service/internal/domain"
)

// UserService defines the service contract used by the application handlers.
type UserService interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User, displayName, avatarURL string) (*domain.UserResponse, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) (*domain.DeleteUserResponse, error)
	List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error)

	// Universal profile
	GetProfile(ctx context.Context, userID string) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, profile *domain.Profile) error

	// Dating profile
	GetDatingProfile(ctx context.Context, userID string) (*domain.DatingProfile, error)
	UpdateDatingProfile(ctx context.Context, profile *domain.DatingProfile) error
	DeleteDatingProfile(ctx context.Context, userID string) error

	// Worker profile
	GetWorkerProfile(ctx context.Context, userID string) (*domain.WorkerProfile, error)
	UpdateWorkerProfile(ctx context.Context, profile *domain.WorkerProfile) error
	DeleteWorkerProfile(ctx context.Context, userID string) error

	// Profile photos
	ListPhotos(ctx context.Context, userID string) ([]*domain.ProfilePhoto, error)
	AddPhoto(ctx context.Context, photo *domain.ProfilePhoto) error
	DeletePhoto(ctx context.Context, photoID string) error
	SetPrimaryPhoto(ctx context.Context, photoID, userID string) (*domain.ProfilePhoto, error)

	// Search
	SearchUsers(ctx context.Context, searchTerm string, excludeUserID string, limit int) ([]*domain.Profile, error)

	// Block / Unblock
	BlockUser(ctx context.Context, blockerID, blockedID string) error
	UnblockUser(ctx context.Context, blockerID, blockedID string) error
}
