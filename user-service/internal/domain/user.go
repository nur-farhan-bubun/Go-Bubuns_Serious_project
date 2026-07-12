package domain

import "time"

// ─── Enum types ─────────────────────────────────────────────────────────────

// UserRole represents the role enum from the migration schema.
type UserRole string

const (
	UserRoleJobSeeker  UserRole = "JOB_SEEKER"
	UserRoleDatingOnly UserRole = "DATING_ONLY"
	UserRoleSocialOnly UserRole = "SOCIAL_ONLY"
	UserRolePowerUser  UserRole = "POWER_USER"
)

// ─── Core Domain Models (1:1 with migration tables) ─────────────────────────

// User represents the core identity/auth record (users table).
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	IsOnboarded  bool      `json:"is_onboarded"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Profile represents the universal display profile (profiles table, 1:1 with users).
type Profile struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Bio         string    `json:"bio,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DatingProfile represents dating-specific profile (dating_profiles table, 1:1 with users).
type DatingProfile struct {
	UserID           string    `json:"user_id"`
	Gender           string    `json:"gender"`
	InterestedIn     string    `json:"interested_in"`
	BirthDate        time.Time `json:"birth_date"`
	HeightCm         *int      `json:"height_cm,omitempty"`
	RelationshipGoal string    `json:"relationship_goal,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// WorkerProfile represents worker/job-seeker profile (worker_profiles table, 1:1 with users).
type WorkerProfile struct {
	UserID             string    `json:"user_id"`
	Skills             []string  `json:"skills,omitempty"`
	HourlyRate         *float64  `json:"hourly_rate,omitempty"`
	IsAvailable        bool      `json:"is_available"`
	CompletedJobsCount int       `json:"completed_jobs_count"`
	RatingAvg          float64   `json:"rating_avg"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ProfilePhoto represents a user's gallery photo (profile_photos table, 1:N with users).
type ProfilePhoto struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	S3URL     string    `json:"s3_url"`
	SortOrder int       `json:"sort_order"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── API DTOs (aggregate views) ─────────────────────────────────────────────

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Name      string   `json:"name" validate:"required,min=1,max=100"`
	Bio       string   `json:"bio" validate:"max=500"`
	PhotoURLs []string `json:"photo_urls"`
	Phone     string   `json:"phone,omitempty"`
}

// UserResponse is a consistent response body for user operations.
type UserResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Bio       string   `json:"bio"`
	AvatarURL string   `json:"avatar_url"`
	PhotoURLs []string `json:"photo_urls"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// DeleteUserResponse is returned after deleting a user.
type DeleteUserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Message   string `json:"message"`
}

// ListUsersResponse is returned by the list endpoint.
type ListUsersResponse struct {
	Users []*UserResponse `json:"users"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

// ToUser converts a CreateUserRequest to a User domain model (identity only).
// Profile data (name, bio, avatar) must be handled separately.
func (r *CreateUserRequest) ToUser(id string, now time.Time) *User {
	return &User{
		ID:         id,
		Email:      r.Email,
		Phone:      r.Phone,
		Role:       UserRoleSocialOnly,
		IsOnboarded: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// ToResponse converts a User to a UserResponse.
// Profile fields (name, bio, avatar, photo_urls) are left empty
// and should be populated by the service layer from Profile/ProfilePhoto tables.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

type BlockedUser struct {
	BlockerID string    `json:"blocker_id"`
	BlockedID string    `json:"blocked_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Auth types ─────────────────────────────────────────────────────────────

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
	Token string               `json:"token"`
	User  *UserProfileResponse `json:"user"`
}

// UserProfileResponse is a public-facing user profile.
type UserProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
}

// NewAuthResponse creates an AuthResponse from a User, display name, avatar, and JWT token.
func NewAuthResponse(user *User, displayName, avatarURL, token string) *AuthResponse {
	return &AuthResponse{
		Token: token,
		User: &UserProfileResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      displayName,
			AvatarURL: avatarURL,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}
}
