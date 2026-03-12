package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// ListByUserID returns listening records for a user with cursor-based pagination, ordered by id ASC.
func (r *ListeningRepo) ListByUserID(ctx context.Context, userID string, page domain.PageRequest) (*domain.PageResponse[domain.UserListening], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("listening_repo: list: %w", err)
	}

	query := `
		SELECT id, user_id, story_id, listened_at, completed,
			ST_Y(location::geometry) AS lat,
			ST_X(location::geometry) AS lng
		FROM user_listening
		WHERE user_id = $1`

	args := []interface{}{userID}
	argIdx := 2

	if page.Cursor != "" {
		cursorID, err := domain.DecodeCursor(page.Cursor)
		if err != nil {
			return nil, fmt.Errorf("listening_repo: list: %w", err)
		}
		query += fmt.Sprintf(" AND id > $%d", argIdx)
		args = append(args, cursorID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listening_repo: list: %w", err)
	}
	defer rows.Close()

	var listenings []domain.UserListening
	for rows.Next() {
		var l domain.UserListening
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.StoryID, &l.ListenedAt, &l.Completed,
			&l.Lat, &l.Lng,
		); err != nil {
			return nil, fmt.Errorf("listening_repo: list scan: %w", err)
		}
		listenings = append(listenings, l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listening_repo: list rows: %w", err)
	}

	hasMore := len(listenings) > page.Limit
	if hasMore {
		listenings = listenings[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(listenings) > 0 {
		nextCursor = domain.EncodeCursor(listenings[len(listenings)-1].ID)
	}

	return &domain.PageResponse[domain.UserListening]{
		Items:      listenings,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

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
