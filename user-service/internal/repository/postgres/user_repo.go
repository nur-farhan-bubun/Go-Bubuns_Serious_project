package postgres

import (
	"context"
	"fmt"
"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ride-sharing/user-service/internal/domain"
)

// Repository handles user data access via PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new user repository and connects to PostgreSQL.
// The caller should call Close() on the returned repository when done.
func New(ctx context.Context, databaseURL string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{pool: pool}, nil
}

// Close shuts down the connection pool.
func (r *Repository) Close() {
	r.pool.Close()
}

// Pool returns the underlying connection pool for reuse by other repositories.
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// ─── Queries ────────────────────────────────────────────────────────────────

const selectColumns = `id, email, COALESCE(phone, ''), COALESCE(password_hash, ''), role, is_onboarded, created_at, updated_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var user domain.User
	var phone, passwordHash string
	err := row.Scan(
		&user.ID, &user.Email, &phone, &passwordHash,
		&user.Role, &user.IsOnboarded,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if phone != "" {
		user.Phone = phone
	}
	if passwordHash != "" {
		user.PasswordHash = passwordHash
	}
	return &user, nil
}

// GetByID retrieves a user by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT ` + selectColumns + ` FROM users WHERE id = $1`

	user, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// FindByEmail retrieves a user by email.
// Returns (nil, nil) if not found — that's not an error.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + selectColumns + ` FROM users WHERE email = $1`

	user, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // not found is not an error
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return user, nil
}

// Create inserts a new user and returns the created response.
func (r *Repository) Create(ctx context.Context, user *domain.User) (*domain.UserResponse, error) {
	query := `
		INSERT INTO users (id, email, phone, role, is_onboarded, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Phone,
		user.Role, user.IsOnboarded,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user.ToResponse(), nil
}

// Update updates an existing user.
func (r *Repository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, phone = $2, role = $3, is_onboarded = $4, updated_at = $5
		WHERE id = $6
	`

	tag, err := r.pool.Exec(ctx, query,
		user.Email, user.Phone, user.Role, user.IsOnboarded,
		user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", user.ID)
	}
	return nil
}

// Delete removes a user by ID.
func (r *Repository) Delete(ctx context.Context, id string) (*domain.DeleteUserResponse, error) {
	// SQL কোয়েরিতে RETURNING ক্লজ যোগ করা হয়েছে যাতে ডিলিট হওয়া রো-এর ডেটা পাওয়া যায়
	query := `
		DELETE FROM users 
		WHERE id = $1 
		RETURNING id, email`

	var response domain.DeleteUserResponse

	err := r.pool.QueryRow(ctx, query, id).Scan(&response.ID, &response.Email)
	if err != nil {
		// যদি ওই ID-র কোনো ইউজার না থাকে, তবে pgx সাধারণত pgx.ErrNoRows রিটার্ন করে
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	// সাকসেস মেসেজ সেট করা
	response.Message = "User deleted successfully"

	// সাকসেস রেসপন্স অবজেক্ট এবং nil এরর রিটার্ন
	return &response, nil
}

func (r *Repository) List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error) {
	// Total count
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if page < 1 {
		
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `SELECT ` + selectColumns + ` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}
