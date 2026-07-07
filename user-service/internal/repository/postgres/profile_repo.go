package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ride-sharing/user-service/internal/domain"
)

// ProfileRepository handles universal profile data access via PostgreSQL.
type ProfileRepository struct {
	pool *pgxpool.Pool
}

// NewProfileRepository creates a new profile repository.
func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

// ─── Queries ────────────────────────────────────────────────────────────────

const profileColumns = `id, user_id, display_name, COALESCE(avatar_url, ''), COALESCE(bio, ''), updated_at`

func scanProfile(row pgx.Row) (*domain.Profile, error) {
	var p domain.Profile
	var avatarURL, bio string
	err := row.Scan(
		&p.ID, &p.UserID, &p.DisplayName, &avatarURL, &bio, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if avatarURL != "" {
		p.AvatarURL = avatarURL
	}
	if bio != "" {
		p.Bio = bio
	}
	return &p, nil
}

// GetByUserID retrieves a profile by user ID.
func (r *ProfileRepository) GetByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	query := `SELECT ` + profileColumns + ` FROM profiles WHERE user_id = $1`

	profile, err := scanProfile(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("profile not found for user: %s", userID)
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return profile, nil
}

// Upsert creates or replaces a profile (upsert by user_id).
func (r *ProfileRepository) Upsert(ctx context.Context, profile *domain.Profile) error {
	now := time.Now().UTC()
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}
	query := `
		INSERT INTO profiles (id, user_id, display_name, avatar_url, bio, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id)
		DO UPDATE SET
			display_name = EXCLUDED.display_name,
			avatar_url = EXCLUDED.avatar_url,
			bio = EXCLUDED.bio,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(ctx, query,
		profile.ID, profile.UserID, profile.DisplayName,
		profile.AvatarURL, profile.Bio, now,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert profile: %w", err)
	}
	profile.UpdatedAt = now
	return nil
}

// Delete removes a profile by user ID.
func (r *ProfileRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM profiles WHERE user_id = $1`

	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("profile not found for user: %s", userID)
	}
	return nil
}
