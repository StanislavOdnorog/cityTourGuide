package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// StoryRepo handles database operations for stories.
type StoryRepo struct {
	pool *pgxpool.Pool
}

// NewStoryRepo creates a new StoryRepo.
func NewStoryRepo(pool *pgxpool.Pool) *StoryRepo {
	return &StoryRepo{pool: pool}
}

// Create inserts a new story and returns it with generated fields.
func (r *StoryRepo) Create(ctx context.Context, story *domain.Story) (*domain.Story, error) {
	query := `
		INSERT INTO story (poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at`

	var s domain.Story
	err := r.pool.QueryRow(ctx, query,
		story.POIID,
		story.Language,
		story.Text,
		story.AudioURL,
		story.DurationSec,
		story.LayerType,
		story.OrderIndex,
		story.IsInflation,
		story.Confidence,
		story.Sources,
		story.Status,
	).Scan(
		&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
		&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("story_repo: create: %w", err)
	}

	return &s, nil
}

// GetByID returns a story by its ID.
func (r *StoryRepo) GetByID(ctx context.Context, id int) (*domain.Story, error) {
	query := `
		SELECT id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at
		FROM story
		WHERE id = $1`

	var s domain.Story
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
		&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("story_repo: get by id: %w", err)
	}

	return &s, nil
}

// GetByPOIID returns stories for a given POI, filtered by language and status.
func (r *StoryRepo) GetByPOIID(ctx context.Context, poiID int, language string, status *domain.StoryStatus) ([]domain.Story, error) {
	query := `
		SELECT id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at
		FROM story
		WHERE poi_id = $1 AND language = $2`

	args := []interface{}{poiID, language}
	argIdx := 3

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
	}

	query += " ORDER BY order_index, created_at"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("story_repo: get by poi id: %w", err)
	}
	defer rows.Close()

	var stories []domain.Story
	for rows.Next() {
		var s domain.Story
		if err := rows.Scan(
			&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
			&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
			&s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("story_repo: get by poi id scan: %w", err)
		}
		stories = append(stories, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("story_repo: get by poi id rows: %w", err)
	}

	return stories, nil
}

// ListByPOIID returns stories with cursor-based pagination, ordered by id ASC.
func (r *StoryRepo) ListByPOIID(ctx context.Context, poiID int, language string, status *domain.StoryStatus, page domain.PageRequest) (*domain.PageResponse[domain.Story], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("story_repo: list: %w", err)
	}

	query := `
		SELECT id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at
		FROM story
		WHERE poi_id = $1 AND language = $2`

	args := []interface{}{poiID, language}
	argIdx := 3

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	if page.Cursor != "" {
		cursorID, err := domain.DecodeCursor(page.Cursor)
		if err != nil {
			return nil, fmt.Errorf("story_repo: list: %w", err)
		}
		query += fmt.Sprintf(" AND id > $%d", argIdx)
		args = append(args, cursorID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("story_repo: list: %w", err)
	}
	defer rows.Close()

	var stories []domain.Story
	for rows.Next() {
		var s domain.Story
		if err := rows.Scan(
			&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
			&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
			&s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("story_repo: list scan: %w", err)
		}
		stories = append(stories, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("story_repo: list rows: %w", err)
	}

	hasMore := len(stories) > page.Limit
	if hasMore {
		stories = stories[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(stories) > 0 {
		nextCursor = domain.EncodeCursor(stories[len(stories)-1].ID)
	}

	return &domain.PageResponse[domain.Story]{
		Items:      stories,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// Update modifies an existing story and returns the updated record.
func (r *StoryRepo) Update(ctx context.Context, story *domain.Story) (*domain.Story, error) {
	query := `
		UPDATE story
		SET poi_id = $2, language = $3, text = $4, audio_url = $5, duration_sec = $6,
		    layer_type = $7, order_index = $8, is_inflation = $9, confidence = $10,
		    sources = $11, status = $12, updated_at = NOW()
		WHERE id = $1
		RETURNING id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at`

	var s domain.Story
	err := r.pool.QueryRow(ctx, query,
		story.ID,
		story.POIID,
		story.Language,
		story.Text,
		story.AudioURL,
		story.DurationSec,
		story.LayerType,
		story.OrderIndex,
		story.IsInflation,
		story.Confidence,
		story.Sources,
		story.Status,
	).Scan(
		&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
		&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("story_repo: update: %w", err)
	}

	return &s, nil
}

// Delete removes a story by its ID.
func (r *StoryRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM story WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("story_repo: delete: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// GetDownloadManifest returns stories for a city with POI name, for building download manifests.
func (r *StoryRepo) GetDownloadManifest(ctx context.Context, cityID int, language string) ([]domain.DownloadManifestItem, error) {
	query := `
		SELECT s.id, s.poi_id, p.name, s.audio_url, s.duration_sec
		FROM story s
		INNER JOIN poi p ON s.poi_id = p.id
		WHERE p.city_id = $1 AND s.language = $2 AND s.status = 'active' AND s.audio_url IS NOT NULL
		ORDER BY p.interest_score DESC, s.order_index`

	rows, err := r.pool.Query(ctx, query, cityID, language)
	if err != nil {
		return nil, fmt.Errorf("story_repo: get download manifest: %w", err)
	}
	defer rows.Close()

	var items []domain.DownloadManifestItem
	for rows.Next() {
		var item domain.DownloadManifestItem
		if err := rows.Scan(
			&item.StoryID, &item.POIID, &item.POIName, &item.AudioURL, &item.DurationSec,
		); err != nil {
			return nil, fmt.Errorf("story_repo: get download manifest scan: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("story_repo: get download manifest rows: %w", err)
	}

	return items, nil
}

// CountByPOI returns the number of stories for a given POI.
func (r *StoryRepo) CountByPOI(ctx context.Context, poiID int) (int, error) {
	query := `SELECT COUNT(*) FROM story WHERE poi_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, poiID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("story_repo: count by poi: %w", err)
	}

	return count, nil
}
