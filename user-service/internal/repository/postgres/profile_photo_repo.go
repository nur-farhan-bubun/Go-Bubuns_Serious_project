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

// ProfilePhotoRepository handles profile photo data access via PostgreSQL.
type ProfilePhotoRepository struct {
	pool *pgxpool.Pool
}

// NewProfilePhotoRepository creates a new profile photo repository.
func NewProfilePhotoRepository(pool *pgxpool.Pool) *ProfilePhotoRepository {
	return &ProfilePhotoRepository{pool: pool}
}

// ─── Queries ────────────────────────────────────────────────────────────────

const photoColumns = `id, user_id, s3_url, sort_order, is_primary, created_at`

func scanPhoto(row pgx.Row) (*domain.ProfilePhoto, error) {
	var p domain.ProfilePhoto
	err := row.Scan(
		&p.ID, &p.UserID, &p.S3URL, &p.SortOrder, &p.IsPrimary, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPhotos(rows pgx.Rows) ([]*domain.ProfilePhoto, error) {
	var photos []*domain.ProfilePhoto
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan photo row: %w", err)
		}
		photos = append(photos, photo)
	}
	return photos, nil
}

// ListByUserID retrieves all profile photos for a user, ordered by sort_order.
func (r *ProfilePhotoRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.ProfilePhoto, error) {
	query := `SELECT ` + photoColumns + ` FROM profile_photos WHERE user_id = $1 ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}
	defer rows.Close()

	return scanPhotos(rows)
}

// GetByID retrieves a single profile photo by ID.
func (r *ProfilePhotoRepository) GetByID(ctx context.Context, photoID string) (*domain.ProfilePhoto, error) {
	query := `SELECT ` + photoColumns + ` FROM profile_photos WHERE id = $1`

	photo, err := scanPhoto(r.pool.QueryRow(ctx, query, photoID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("photo not found: %s", photoID)
		}
		return nil, fmt.Errorf("failed to get photo: %w", err)
	}
	return photo, nil
}

// Create inserts a new profile photo.
func (r *ProfilePhotoRepository) Create(ctx context.Context, photo *domain.ProfilePhoto) error {
	if photo.ID == "" {
		photo.ID = uuid.New().String()
	}
	now := time.Now().UTC()

	// If this photo is set as primary, unset any existing primary for the user
	if photo.IsPrimary {
		if err := r.unsetPrimary(ctx, photo.UserID); err != nil {
			return err
		}
	}

	// Determine next sort order
	if photo.SortOrder == 0 {
		nextOrder, err := r.nextSortOrder(ctx, photo.UserID)
		if err != nil {
			return err
		}
		photo.SortOrder = nextOrder
	}

	query := `
		INSERT INTO profile_photos (id, user_id, s3_url, sort_order, is_primary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		photo.ID, photo.UserID, photo.S3URL,
		photo.SortOrder, photo.IsPrimary, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create photo: %w", err)
	}
	photo.CreatedAt = now
	return nil
}

// Delete removes a profile photo by ID.
func (r *ProfilePhotoRepository) Delete(ctx context.Context, photoID string) error {
	query := `DELETE FROM profile_photos WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, photoID)
	if err != nil {
		return fmt.Errorf("failed to delete photo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("photo not found: %s", photoID)
	}
	return nil
}

// SetPrimary sets a specific photo as the primary photo for a user.
// All other photos for the user are set to non-primary.
func (r *ProfilePhotoRepository) SetPrimary(ctx context.Context, photoID, userID string) error {
	if err := r.unsetPrimary(ctx, userID); err != nil {
		return err
	}

	query := `UPDATE profile_photos SET is_primary = TRUE WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, photoID, userID)
	if err != nil {
		return fmt.Errorf("failed to set primary photo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("photo not found: %s", photoID)
	}
	return nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (r *ProfilePhotoRepository) unsetPrimary(ctx context.Context, userID string) error {
	query := `UPDATE profile_photos SET is_primary = FALSE WHERE user_id = $1 AND is_primary = TRUE`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unset primary photo: %w", err)
	}
	return nil
}

func (r *ProfilePhotoRepository) nextSortOrder(ctx context.Context, userID string) (int, error) {
	query := `SELECT COALESCE(MAX(sort_order), 0) + 1 FROM profile_photos WHERE user_id = $1`
	var next int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("failed to compute next sort order: %w", err)
	}
	return next, nil
}
