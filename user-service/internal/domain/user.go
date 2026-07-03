package domain

import "time"

// User represents a user profile.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Bio       string    `json:"bio"`
	AvatarURL string    `json:"avatar_url"`
	PhotoURLs []string  `json:"photo_urls"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Name      string   `json:"name" validate:"required,min=1,max=100"`
	Bio       string   `json:"bio" validate:"max=500"`
	PhotoURLs []string `json:"photo_urls"`
}

// CreateUserResponse is the response body after creating a user.
type CreateUserResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Bio       string   `json:"bio"`
	PhotoURLs []string `json:"photo_urls"`
	CreatedAt string   `json:"created_at"`
}

// ToUser converts a CreateUserRequest to a User domain model.
func (r *CreateUserRequest) ToUser(id string, now time.Time) *User {
	return &User{
		ID:        id,
		Email:     r.Email,
		Name:      r.Name,
		Bio:       r.Bio,
		PhotoURLs: r.PhotoURLs,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ToCreateResponse converts a User to a CreateUserResponse.
func (u *User) ToCreateResponse() *CreateUserResponse {
	return &CreateUserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Bio:       u.Bio,
		PhotoURLs: u.PhotoURLs,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

// ─── Auth types ────────────────────────────────────────────────────────────

// GoogleUserInfo represents the user info returned by Google's OAuth API.
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// AuthResponse is returned after a successful login.
type AuthResponse struct {
	Token       string              `json:"token"`
	User        *UserProfileResponse `json:"user"`
}

// UserProfileResponse is a public-facing user profile.
type UserProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
}

// NewAuthResponse creates an AuthResponse from a User and JWT token.
func NewAuthResponse(user *User, token string) *AuthResponse {
	return &AuthResponse{
		Token: token,
		User: &UserProfileResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			AvatarURL: user.AvatarURL,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}
}
