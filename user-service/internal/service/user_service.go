package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ride-sharing/user-service/internal/domain"
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

// Service handles user business logic.
type Service struct {
	repo      UserRepository
	oauth2    *oauth2.Config
	jwtSecret []byte
}

// New creates a new user service.
func New(repo UserRepository, googleClientID, googleClientSecret, googleRedirectURL, jwtSecret string) *Service {
	return &Service{
		repo: repo,
		oauth2: &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
		jwtSecret: []byte(jwtSecret),
	}
}

// ─── User CRUD ──────────────────────────────────────────────────────────────

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

// Create creates a new user.
func (s *Service) Create(ctx context.Context, user *domain.User) (*domain.UserResponse, error) {
	return s.repo.Create(ctx, user)
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

	return domain.NewAuthResponse(user, jwtToken), nil
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

func (s *Service) findOrCreateUser(ctx context.Context, googleUser *domain.GoogleUserInfo) (*domain.User, error) {
	// Check if user already exists by email
	existing, err := s.repo.FindByEmail(ctx, googleUser.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Sync Google profile changes (name, avatar)
		needsUpdate := false
		if existing.Name != googleUser.Name {
			existing.Name = googleUser.Name
			needsUpdate = true
		}
		if existing.AvatarURL != googleUser.Picture {
			existing.AvatarURL = googleUser.Picture
			needsUpdate = true
		}
		if needsUpdate {
			existing.UpdatedAt = time.Now().UTC()
			if err := s.repo.Update(ctx, existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	// Create new user
	now := time.Now().UTC()
	user := &domain.User{
		ID:        uuid.New().String(),
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		AvatarURL: googleUser.Picture,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) generateJWT(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"iat":   now.Unix(),
		"exp":   now.Add(72 * time.Hour).Unix(), // 72 hour expiry
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func generateID() string {
	return uuid.New().String()
}
