package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ride-sharing/user-service/internal/domain"
)

// DatingProfileRepository handles dating profile data access via PostgreSQL.
type DatingProfileRepository struct {
	pool *pgxpool.Pool
}

// NewDatingProfileRepository creates a new dating profile repository.
func NewDatingProfileRepository(pool *pgxpool.Pool) *DatingProfileRepository {
	return &DatingProfileRepository{pool: pool}
}

// ─── Queries ────────────────────────────────────────────────────────────────

const datingProfileColumns = `user_id, gender, interested_in, birth_date, COALESCE(height_cm, 0)::INT, COALESCE(relationship_goal, ''), updated_at`

func scanDatingProfile(row pgx.Row) (*domain.DatingProfile, error) {
	var p domain.DatingProfile
	var heightCm int
	err := row.Scan(
		&p.UserID, &p.Gender, &p.InterestedIn, &p.BirthDate,
		&heightCm, &p.RelationshipGoal, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if heightCm > 0 {
		p.HeightCm = &heightCm
	}
	return &p, nil
}

// GetByUserID retrieves a dating profile by user ID.
func (r *DatingProfileRepository) GetByUserID(ctx context.Context, userID string) (*domain.DatingProfile, error) {
	query := `SELECT ` + datingProfileColumns + ` FROM dating_profiles WHERE user_id = $1`

	profile, err := scanDatingProfile(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("dating profile not found for user: %s", userID)
		}
		return nil, fmt.Errorf("failed to get dating profile: %w", err)
	}
	return profile, nil
}

// Upsert creates or replaces a dating profile (upsert by user_id).
func (r *DatingProfileRepository) Upsert(ctx context.Context, profile *domain.DatingProfile) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO dating_profiles (user_id, gender, interested_in, birth_date, height_cm, relationship_goal, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id)
		DO UPDATE SET
			gender = EXCLUDED.gender,
			interested_in = EXCLUDED.interested_in,
			birth_date = EXCLUDED.birth_date,
			height_cm = EXCLUDED.height_cm,
			relationship_goal = EXCLUDED.relationship_goal,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(ctx, query,
		profile.UserID, profile.Gender, profile.InterestedIn, profile.BirthDate,
		profile.HeightCm, profile.RelationshipGoal, now,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert dating profile: %w", err)
	}
	profile.UpdatedAt = now
	return nil
}

// Delete removes a dating profile by user ID.
func (r *DatingProfileRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM dating_profiles WHERE user_id = $1`

	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete dating profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("dating profile not found for user: %s", userID)
	}
	return nil
}
