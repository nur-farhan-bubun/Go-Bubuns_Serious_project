package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ride-sharing/user-service/internal/domain"
	userevent "github.com/ride-sharing/user-service/internal/kafka"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// UserRepository defines the persistence contract for user data.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.UserResponse, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) (*domain.DeleteUserResponse, error)
	List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error)
}

// ProfileRepository defines the persistence contract for universal profile data.
type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	Upsert(ctx context.Context, profile *domain.Profile) error
	Delete(ctx context.Context, userID string) error
}

// DatingProfileRepository defines the persistence contract for dating profile data.
type DatingProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.DatingProfile, error)
	Upsert(ctx context.Context, profile *domain.DatingProfile) error
	Delete(ctx context.Context, userID string) error
}

// WorkerProfileRepository defines the persistence contract for worker profile data.
type WorkerProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.WorkerProfile, error)
	Upsert(ctx context.Context, profile *domain.WorkerProfile) error
	Delete(ctx context.Context, userID string) error
}

// PhotoRepository defines the persistence contract for profile photos.
type PhotoRepository interface {
	ListByUserID(ctx context.Context, userID string) ([]*domain.ProfilePhoto, error)
	GetByID(ctx context.Context, photoID string) (*domain.ProfilePhoto, error)
	Create(ctx context.Context, photo *domain.ProfilePhoto) error
	Delete(ctx context.Context, photoID string) error
	SetPrimary(ctx context.Context, photoID, userID string) error
}

// EventPublisher defines the contract for publishing user lifecycle events.
type EventPublisher interface {
	Publish(ctx context.Context, event *userevent.UserEvent) error
	Close() error
}

// Service handles user business logic.
type Service struct {
	repo             UserRepository
	profileRepo      ProfileRepository
	datingRepo       DatingProfileRepository
	workerRepo       WorkerProfileRepository
	photoRepo        PhotoRepository
	oauth2           *oauth2.Config
	jwtSecret        []byte
	eventPublisher   EventPublisher
}

// New creates a new user service.
func New(repo UserRepository, profileRepo ProfileRepository, datingRepo DatingProfileRepository, workerRepo WorkerProfileRepository, photoRepo PhotoRepository, googleClientID, googleClientSecret, googleRedirectURL, jwtSecret string, eventPublisher EventPublisher) *Service {
	return &Service{
		repo:        repo,
		profileRepo: profileRepo,
		datingRepo:  datingRepo,
		workerRepo:  workerRepo,
		photoRepo:   photoRepo,
		oauth2: &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
		jwtSecret:      []byte(jwtSecret),
		eventPublisher: eventPublisher,
	}
}

// ─── User CRUD ──────────────────────────────────────────────────────────────

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

// Create creates a new user, optionally creates a profile, and publishes
// a user.created event for eventual consistency with downstream services.
func (s *Service) Create(ctx context.Context, user *domain.User, displayName, avatarURL string) (*domain.UserResponse, error) {
	resp, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// Create profile if display info was provided (e.g. from email/password registration)
	if displayName != "" || avatarURL != "" {
		_ = s.profileRepo.Upsert(ctx, &domain.Profile{
			UserID:      user.ID,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			UpdatedAt:   time.Now().UTC(),
		})
	}

	// Publish user.created event for eventual consistency
	s.publishUserCreated(ctx, user.ID, user.Email, displayName, avatarURL)

	return resp, nil
}

// Update updates a user.
func (s *Service) Update(ctx context.Context, user *domain.User) error {
	return s.repo.Update(ctx, user)
}

// Delete deletes a user and returns the deleted user's data.
func (s *Service) Delete(ctx context.Context, id string) (*domain.DeleteUserResponse, error) {
	return s.repo.Delete(ctx, id)
}

// List returns a paginated list of users.
func (s *Service) List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error) {
	return s.repo.List(ctx, page, pageSize)
}

// ─── Google OAuth ───────────────────────────────────────────────────────────

// GenerateGoogleLoginURL returns the URL to redirect the user to Google's consent screen.
func (s *Service) GenerateGoogleLoginURL(state string) string {
	return s.oauth2.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// HandleGoogleCallback exchanges the auth code for a token, fetches user info,
// creates or finds the user, and returns a signed JWT.
func (s *Service) HandleGoogleCallback(ctx context.Context, code string) (*domain.AuthResponse, error) {
	// 1. Exchange auth code for Google token
	token, err := s.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// 2. Fetch user info from Google
	googleUser, err := s.fetchGoogleUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// 3. Find or create user in our database
	user, err := s.findOrCreateUser(ctx, googleUser)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// 4. Generate JWT session token
	jwtToken, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return domain.NewAuthResponse(user, googleUser.Name, googleUser.Picture, jwtToken), nil
}

// ValidateJWT validates a JWT token and returns the user ID.
func (s *Service) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid subject claim")
	}
	return userID, nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (s *Service) fetchGoogleUserInfo(ctx context.Context, accessToken string) (*domain.GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned %d: %s", resp.StatusCode, string(body))
	}

	var info domain.GoogleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ─── Profile CRUD ───────────────────────────────────────────────────────────

// GetProfile retrieves the universal profile for a user.
func (s *Service) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	return s.profileRepo.GetByUserID(ctx, userID)
}

// UpdateProfile creates or updates a user's universal profile.
func (s *Service) UpdateProfile(ctx context.Context, profile *domain.Profile) error {
	return s.profileRepo.Upsert(ctx, profile)
}

// ─── Dating Profile CRUD ───────────────────────────────────────────────────

// GetDatingProfile retrieves the dating profile for a user.
func (s *Service) GetDatingProfile(ctx context.Context, userID string) (*domain.DatingProfile, error) {
	return s.datingRepo.GetByUserID(ctx, userID)
}

// UpdateDatingProfile creates or updates a dating profile.
func (s *Service) UpdateDatingProfile(ctx context.Context, profile *domain.DatingProfile) error {
	return s.datingRepo.Upsert(ctx, profile)
}

// DeleteDatingProfile removes a dating profile.
func (s *Service) DeleteDatingProfile(ctx context.Context, userID string) error {
	return s.datingRepo.Delete(ctx, userID)
}

// ─── Worker Profile CRUD ────────────────────────────────────────────────────

// GetWorkerProfile retrieves the worker profile for a user.
func (s *Service) GetWorkerProfile(ctx context.Context, userID string) (*domain.WorkerProfile, error) {
	return s.workerRepo.GetByUserID(ctx, userID)
}

// UpdateWorkerProfile creates or updates a worker profile.
func (s *Service) UpdateWorkerProfile(ctx context.Context, profile *domain.WorkerProfile) error {
	return s.workerRepo.Upsert(ctx, profile)
}

// DeleteWorkerProfile removes a worker profile.
func (s *Service) DeleteWorkerProfile(ctx context.Context, userID string) error {
	return s.workerRepo.Delete(ctx, userID)
}

// ─── Profile Photo CRUD ─────────────────────────────────────────────────────

// ListPhotos retrieves all profile photos for a user.
func (s *Service) ListPhotos(ctx context.Context, userID string) ([]*domain.ProfilePhoto, error) {
	return s.photoRepo.ListByUserID(ctx, userID)
}

// AddPhoto adds a new profile photo.
func (s *Service) AddPhoto(ctx context.Context, photo *domain.ProfilePhoto) error {
	return s.photoRepo.Create(ctx, photo)
}

// DeletePhoto removes a profile photo.
func (s *Service) DeletePhoto(ctx context.Context, photoID string) error {
	return s.photoRepo.Delete(ctx, photoID)
}

// SetPrimaryPhoto sets a photo as the primary profile photo.
func (s *Service) SetPrimaryPhoto(ctx context.Context, photoID, userID string) (*domain.ProfilePhoto, error) {
	if err := s.photoRepo.SetPrimary(ctx, photoID, userID); err != nil {
		return nil, err
	}
	return s.photoRepo.GetByID(ctx, photoID)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (s *Service) findOrCreateUser(ctx context.Context, googleUser *domain.GoogleUserInfo) (*domain.User, error) {
	// Check if user already exists by email
	existing, err := s.repo.FindByEmail(ctx, googleUser.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Sync Google profile changes (name, avatar)
		if err := s.syncGoogleProfile(ctx, existing, googleUser); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Create new user
	now := time.Now().UTC()
	user := &domain.User{
		ID:        uuid.New().String(),
		Email:     googleUser.Email,
		Role:      domain.UserRoleSocialOnly,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Create profile with Google data
	profile := &domain.Profile{
		UserID:      user.ID,
		DisplayName: googleUser.Name,
		AvatarURL:   googleUser.Picture,
		UpdatedAt:   now,
	}
	if err := s.profileRepo.Upsert(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Publish user.created event for eventual consistency
	s.publishUserCreated(ctx, user.ID, user.Email, profile.DisplayName, profile.AvatarURL)

	return user, nil
}

func (s *Service) syncGoogleProfile(ctx context.Context, user *domain.User, googleUser *domain.GoogleUserInfo) error {
	profile, err := s.profileRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		// If not found, create a new profile with Google data
		if strings.Contains(err.Error(), "not found") {
			profile = &domain.Profile{
				UserID:      user.ID,
				DisplayName: googleUser.Name,
				AvatarURL:   googleUser.Picture,
			}
			return s.profileRepo.Upsert(ctx, profile)
		}
		return fmt.Errorf("failed to get profile for sync: %w", err)
	}

	// Sync Google profile changes
	needsUpdate := false
	if profile.DisplayName != googleUser.Name {
		profile.DisplayName = googleUser.Name
		needsUpdate = true
	}
	if profile.AvatarURL != googleUser.Picture {
		profile.AvatarURL = googleUser.Picture
		needsUpdate = true
	}
	if needsUpdate {
		if err := s.profileRepo.Upsert(ctx, profile); err != nil {
			return err
		}
	}

	// Update user timestamp
	user.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, user)
}

// publishUserCreated publishes a user.created event to Kafka (fire-and-forget).
// If the event publisher is not configured, this is a no-op.
func (s *Service) publishUserCreated(ctx context.Context, userID, email, displayName, avatarURL string) {
	if s.eventPublisher == nil {
		return
	}
	event := &userevent.UserEvent{
		Type:        userevent.UserCreated,
		UserID:      userID,
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
	if err := s.eventPublisher.Publish(ctx, event); err != nil {
		// Log but don't fail — eventual consistency means the chat-service
		// will pick up the user on its next sync.
		_ = err
	}
}

func (s *Service) generateJWT(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   now.Unix(),
		"exp":   now.Add(72 * time.Hour).Unix(), // 72 hour expiry
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// Backfill publishes user.created events for all existing users.
// This is called on startup so previously-registered users appear in the
// chat-service user cache (eventual consistency backfill).
func (s *Service) Backfill(ctx context.Context) error {
	if s.eventPublisher == nil {
		return nil
	}
	page := 1
	pageSize := 100
	for {
		users, total, err := s.repo.List(ctx, page, pageSize)
		if err != nil {
			return fmt.Errorf("backfill: list page %d: %w", page, err)
		}
		for _, user := range users {
			displayName := ""
			avatarURL := ""
			if profile, err := s.profileRepo.GetByUserID(ctx, user.ID); err == nil && profile != nil {
				displayName = profile.DisplayName
				avatarURL = profile.AvatarURL
			}
			s.publishUserCreated(ctx, user.ID, user.Email, displayName, avatarURL)
		}
		if page*pageSize >= total {
			break
		}
		page++
	}
	return nil
}

func generateID() string {
	return uuid.New().String()
}
