package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// ListeningRepo handles database operations for user listening records.
type ListeningRepo struct {
	pool *pgxpool.Pool
}

// NewListeningRepo creates a new ListeningRepo.
func NewListeningRepo(pool *pgxpool.Pool) *ListeningRepo {
	return &ListeningRepo{pool: pool}
}

// Create inserts a new listening record. Lat/Lng are optional (nil = no location).
func (r *ListeningRepo) Create(ctx context.Context, userID string, storyID int, completed bool, lat, lng *float64) (*domain.UserListening, error) {
	var query string
	var args []interface{}

	if lat != nil && lng != nil {
		query = `
			INSERT INTO user_listening (user_id, story_id, completed, location)
			VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography)
			RETURNING id, user_id, story_id, listened_at, completed,
				ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng`
		args = []interface{}{userID, storyID, completed, *lng, *lat}
	} else {
		query = `
			INSERT INTO user_listening (user_id, story_id, completed)
			VALUES ($1, $2, $3)
			RETURNING id, user_id, story_id, listened_at, completed,
				NULL::double precision AS lat, NULL::double precision AS lng`
		args = []interface{}{userID, storyID, completed}
	}

	var l domain.UserListening
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&l.ID, &l.UserID, &l.StoryID, &l.ListenedAt, &l.Completed,
		&l.Lat, &l.Lng,
	)
	if err != nil {
		return nil, fmt.Errorf("listening_repo: create: %w", err)
	}

	return &l, nil
}

// CreateOrUpdate inserts a new listening record or updates an existing one (UPSERT).
// If a record for the same user_id+story_id exists, updates completed, listened_at, and location.
func (r *ListeningRepo) CreateOrUpdate(ctx context.Context, userID string, storyID int, completed bool, lat, lng *float64) (*domain.UserListening, error) {
	var query string
	var args []interface{}

	if lat != nil && lng != nil {
		query = `
			INSERT INTO user_listening (user_id, story_id, completed, location)
			VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography)
			ON CONFLICT (user_id, story_id) DO UPDATE SET
				completed = EXCLUDED.completed,
				listened_at = NOW(),
				location = EXCLUDED.location
			RETURNING id, user_id, story_id, listened_at, completed,
				ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng`
		args = []interface{}{userID, storyID, completed, *lng, *lat}
	} else {
		query = `
			INSERT INTO user_listening (user_id, story_id, completed)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, story_id) DO UPDATE SET
				completed = EXCLUDED.completed,
				listened_at = NOW()
			RETURNING id, user_id, story_id, listened_at, completed,
				NULL::double precision AS lat, NULL::double precision AS lng`
		args = []interface{}{userID, storyID, completed}
	}

	var l domain.UserListening
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&l.ID, &l.UserID, &l.StoryID, &l.ListenedAt, &l.Completed,
		&l.Lat, &l.Lng,
	)
	if err != nil {
		return nil, fmt.Errorf("listening_repo: create or update: %w", err)
	}

	return &l, nil
}

// GetListenedStoryIDs returns a list of story IDs that a user has listened to.
// Used for deduplication when selecting nearby stories.
func (r *ListeningRepo) GetListenedStoryIDs(ctx context.Context, userID string) ([]int, error) {
	query := `
		SELECT DISTINCT story_id
		FROM user_listening
		WHERE user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listening_repo: get listened story ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("listening_repo: get listened story ids scan: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listening_repo: get listened story ids rows: %w", err)
	}

	return ids, nil
}

// HasListened checks whether a user has listened to a specific story.
func (r *ListeningRepo) HasListened(ctx context.Context, userID string, storyID int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_listening
			WHERE user_id = $1 AND story_id = $2
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, storyID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("listening_repo: has listened: %w", err)
	}

	return exists, nil
}
