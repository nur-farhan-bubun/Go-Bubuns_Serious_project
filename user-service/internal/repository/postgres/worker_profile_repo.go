package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ride-sharing/user-service/internal/domain"
)

// WorkerProfileRepository handles worker/job-seeker profile data access via PostgreSQL.
type WorkerProfileRepository struct {
	pool *pgxpool.Pool
}

// NewWorkerProfileRepository creates a new worker profile repository.
func NewWorkerProfileRepository(pool *pgxpool.Pool) *WorkerProfileRepository {
	return &WorkerProfileRepository{pool: pool}
}

// ─── Queries ────────────────────────────────────────────────────────────────

const workerProfileColumns = `user_id, skills, hourly_rate, is_available, completed_jobs_count, rating_avg, updated_at`

func scanWorkerProfile(row pgx.Row) (*domain.WorkerProfile, error) {
	var p domain.WorkerProfile
	err := row.Scan(
		&p.UserID, &p.Skills, &p.HourlyRate,
		&p.IsAvailable, &p.CompletedJobsCount, &p.RatingAvg,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByUserID retrieves a worker profile by user ID.
func (r *WorkerProfileRepository) GetByUserID(ctx context.Context, userID string) (*domain.WorkerProfile, error) {
	query := `SELECT ` + workerProfileColumns + ` FROM worker_profiles WHERE user_id = $1`

	profile, err := scanWorkerProfile(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("worker profile not found for user: %s", userID)
		}
		return nil, fmt.Errorf("failed to get worker profile: %w", err)
	}
	return profile, nil
}

// Upsert creates or replaces a worker profile (upsert by user_id).
func (r *WorkerProfileRepository) Upsert(ctx context.Context, profile *domain.WorkerProfile) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO worker_profiles (user_id, skills, hourly_rate, is_available, completed_jobs_count, rating_avg, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id)
		DO UPDATE SET
			skills = EXCLUDED.skills,
			hourly_rate = EXCLUDED.hourly_rate,
			is_available = EXCLUDED.is_available,
			completed_jobs_count = EXCLUDED.completed_jobs_count,
			rating_avg = EXCLUDED.rating_avg,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(ctx, query,
		profile.UserID, profile.Skills, profile.HourlyRate,
		profile.IsAvailable, profile.CompletedJobsCount, profile.RatingAvg,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert worker profile: %w", err)
	}
	profile.UpdatedAt = now
	return nil
}

// Delete removes a worker profile by user ID.
func (r *WorkerProfileRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM worker_profiles WHERE user_id = $1`

	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete worker profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("worker profile not found for user: %s", userID)
	}
	return nil
}
