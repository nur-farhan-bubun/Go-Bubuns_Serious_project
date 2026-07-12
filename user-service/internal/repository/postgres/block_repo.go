package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BlockRepository struct {
	pool *pgxpool.Pool
}

func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{pool: pool}
}

func (r *BlockRepository) BlockUser(ctx context.Context, blockerID, blockedID string) error {
	query := `
		INSERT INTO blocked_users (blocker_id, blocked_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("failed to block user: %w", err)
	}
	return nil
}

func (r *BlockRepository) UnblockUser(ctx context.Context, blockerID, blockedID string) error {
	query := `DELETE FROM blocked_users WHERE blocker_id = $1 AND blocked_id = $2`
	tag, err := r.pool.Exec(ctx, query, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("failed to unblock user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("block record not found")
	}
	return nil
}

func (r *BlockRepository) IsBlocked(ctx context.Context, userID1, userID2 string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM blocked_users
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID1, userID2).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check block status: %w", err)
	}
	return exists, nil
}
