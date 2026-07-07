package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ride-sharing/location-service/internal/domain"
)

// MapPostRepository handles map_posts data access via PostgreSQL + PostGIS.
type MapPostRepository struct {
	pool *pgxpool.Pool
}

// NewMapPostRepository creates a new map post repository.
func NewMapPostRepository(ctx context.Context, databaseURL string) (*MapPostRepository, error) {
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

	return &MapPostRepository{pool: pool}, nil
}

// Close shuts down the connection pool.
func (r *MapPostRepository) Close() {
	r.pool.Close()
}

// scanPost scans a row into a MapPost domain model.
func scanPost(row pgx.Row) (*domain.MapPost, error) {
	var post domain.MapPost
	err := row.Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content,
		&post.Category, &post.ImageURLs,
		&post.Latitude, &post.Longitude,
		&post.CreatedAt, &post.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

// Create inserts a new map post.
func (r *MapPostRepository) Create(ctx context.Context, post *domain.MapPost) (*domain.MapPost, error) {
	query := `
		INSERT INTO map_posts (id, user_id, title, content, category, image_urls, geom, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10)
		RETURNING id, user_id, title, content, category, image_urls,
		          ST_X(geom::geometry) AS longitude, ST_Y(geom::geometry) AS latitude,
		          created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		post.ID, post.UserID, post.Title, post.Content,
		post.Category, post.ImageURLs,
		post.Longitude, post.Latitude,
		post.CreatedAt, post.UpdatedAt,
	)
	return scanPost(row)
}

// GetByID retrieves a map post by ID.
func (r *MapPostRepository) GetByID(ctx context.Context, id string) (*domain.MapPost, error) {
	query := `
		SELECT id, user_id, title, content, category, image_urls,
		       ST_X(geom::geometry) AS longitude, ST_Y(geom::geometry) AS latitude,
		       created_at, updated_at
		FROM map_posts WHERE id = $1`

	post, err := scanPost(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("post not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	return post, nil
}

// ListByLocation returns map posts within a radius (meters) of a point, ordered by distance.
// If lat/lng are both nil, returns all posts ordered by newest first.
func (r *MapPostRepository) ListByLocation(ctx context.Context, lat, lng, radius float64, category string) ([]*domain.MapPost, error) {
	var query string
	var rows pgx.Rows
	var err error

	if category != "" {
		query = `
			SELECT id, user_id, title, content, category, image_urls,
			       ST_X(geom::geometry) AS longitude, ST_Y(geom::geometry) AS latitude,
			       created_at, updated_at
			FROM map_posts
			WHERE ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
			  AND category = $4
			ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, query, lng, lat, radius, category)
	} else {
		query = `
			SELECT id, user_id, title, content, category, image_urls,
			       ST_X(geom::geometry) AS longitude, ST_Y(geom::geometry) AS latitude,
			       created_at, updated_at
			FROM map_posts
			WHERE ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
			ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, query, lng, lat, radius)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	posts := make([]*domain.MapPost, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post row: %w", err)
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// ListAll returns all map posts, newest first.
func (r *MapPostRepository) ListAll(ctx context.Context) ([]*domain.MapPost, error) {
	query := `
		SELECT id, user_id, title, content, category, image_urls,
		       ST_X(geom::geometry) AS longitude, ST_Y(geom::geometry) AS latitude,
		       created_at, updated_at
		FROM map_posts
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	posts := make([]*domain.MapPost, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post row: %w", err)
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// Update updates an existing map post.
func (r *MapPostRepository) Update(ctx context.Context, post *domain.MapPost) (*domain.MapPost, error) {
	query := `
		UPDATE map_posts
		SET title = COALESCE($2, title),
		    content = COALESCE($3, content),
		    category = COALESCE($4, category),
		    image_urls = COALESCE($5, image_urls),
		    updated_at = $6
		WHERE id = $1
		RETURNING id, user_id, title, content, category, image_urls,
		          ST_X(geom::geometry) AS longitude, ST_Y(geom::geometry) AS latitude,
		          created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		post.ID, post.Title, post.Content, post.Category,
		post.ImageURLs, post.UpdatedAt,
	)
	return scanPost(row)
}

// Delete removes a map post by ID.
func (r *MapPostRepository) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM map_posts WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("post not found or not owned by user: %s", id)
	}
	return nil
}
