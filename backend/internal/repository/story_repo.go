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

type storyQueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
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
	return r.getByID(ctx, r.pool, id, "")
}

func (r *StoryRepo) getByID(ctx context.Context, q storyQueryRower, id int, suffix string) (*domain.Story, error) {
	query := `
		SELECT id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at
		FROM story
		WHERE id = $1 ` + suffix

	var s domain.Story
	err := q.QueryRow(ctx, query, id).Scan(
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

// GetByIDForUpdateTx returns a story by its ID and locks it for update.
func (r *StoryRepo) GetByIDForUpdateTx(ctx context.Context, tx pgx.Tx, id int) (*domain.Story, error) {
	return r.getByID(ctx, tx, id, "FOR UPDATE")
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

// GetByPOIIDs returns stories for multiple POIs in a single query, grouped by POI ID.
// Stories are ordered by order_index, created_at within each POI group.
// POIs with no stories will have empty slices in the returned map.
// If excludeUserID is non-empty, stories the user has already listened to are
// excluded via a NOT EXISTS subquery, eliminating a separate round-trip.
func (r *StoryRepo) GetByPOIIDs(ctx context.Context, poiIDs []int, language string, status *domain.StoryStatus, excludeUserID string) (map[int][]domain.Story, error) {
	result := make(map[int][]domain.Story, len(poiIDs))
	if len(poiIDs) == 0 {
		return result, nil
	}

	// Pre-populate map so POIs with no stories get empty slices.
	for _, id := range poiIDs {
		result[id] = []domain.Story{}
	}

	query := `
		SELECT id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at
		FROM story
		WHERE poi_id = ANY($1) AND language = $2`

	args := []interface{}{poiIDs, language}
	argIdx := 3

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	if excludeUserID != "" {
		query += fmt.Sprintf(" AND NOT EXISTS (SELECT 1 FROM user_listening ul WHERE ul.story_id = story.id AND ul.user_id = $%d)", argIdx)
		args = append(args, excludeUserID)
	}

	query += " ORDER BY poi_id, order_index, created_at"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("story_repo: get by poi ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.Story
		if err := rows.Scan(
			&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
			&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
			&s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("story_repo: get by poi ids scan: %w", err)
		}
		result[s.POIID] = append(result[s.POIID], s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("story_repo: get by poi ids rows: %w", err)
	}

	return result, nil
}

// ListByPOIID returns stories with cursor-based pagination, ordered by id ASC.
func (r *StoryRepo) ListByPOIID(ctx context.Context, poiID int, language string, status *domain.StoryStatus, page domain.PageRequest, sort ListSort) (*domain.PageResponse[domain.Story], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("story_repo: list: %w", err)
	}

	resolvedSort, err := ResolveSort(sort, map[string]SortColumn{
		"id":          {Key: "id", Column: "id", Type: SortValueInt},
		"poi_id":      {Key: "poi_id", Column: "poi_id", Type: SortValueInt},
		"language":    {Key: "language", Column: "language", Type: SortValueString},
		"status":      {Key: "status", Column: "status", Type: SortValueString},
		"layer_type":  {Key: "layer_type", Column: "layer_type", Type: SortValueString},
		"order_index": {Key: "order_index", Column: "order_index", Type: SortValueInt16},
		"confidence":  {Key: "confidence", Column: "confidence", Type: SortValueInt16},
		"created_at":  {Key: "created_at", Column: "created_at", Type: SortValueTime},
		"updated_at":  {Key: "updated_at", Column: "updated_at", Type: SortValueTime},
	}, "id", SortDirAsc)
	if err != nil {
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

	cursorCondition, cursorArgs, err := resolvedSort.CursorCondition(page.Cursor, argIdx)
	if err != nil {
		return nil, fmt.Errorf("story_repo: list: %w", err)
	}
	if cursorCondition != "" {
		query += " AND " + cursorCondition
		args = append(args, cursorArgs...)
		argIdx += len(cursorArgs)
	}

	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d", resolvedSort.OrderBy(), argIdx)
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
		nextCursor, err = EncodeOrderedCursor(resolvedSort, storySortValue(stories[len(stories)-1], resolvedSort.Key), stories[len(stories)-1].ID)
		if err != nil {
			return nil, fmt.Errorf("story_repo: list: %w", err)
		}
	}

	return &domain.PageResponse[domain.Story]{
		Items:      stories,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func storySortValue(story domain.Story, key string) interface{} {
	switch key {
	case "poi_id":
		return story.POIID
	case "language":
		return story.Language
	case "status":
		return string(story.Status)
	case "layer_type":
		return string(story.LayerType)
	case "order_index":
		return story.OrderIndex
	case "confidence":
		return story.Confidence
	case "created_at":
		return story.CreatedAt
	case "updated_at":
		return story.UpdatedAt
	default:
		return story.ID
	}
}

// Update modifies an existing story and returns the updated record.
func (r *StoryRepo) Update(ctx context.Context, story *domain.Story) (*domain.Story, error) {
	return r.update(ctx, r.pool, story)
}

func (r *StoryRepo) update(ctx context.Context, q storyQueryRower, story *domain.Story) (*domain.Story, error) {
	query := `
		UPDATE story
		SET poi_id = $2, language = $3, text = $4, audio_url = $5, duration_sec = $6,
		    layer_type = $7, order_index = $8, is_inflation = $9, confidence = $10,
		    sources = $11, status = $12, updated_at = NOW()
		WHERE id = $1
		RETURNING id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at`

	var s domain.Story
	err := q.QueryRow(ctx, query,
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

// UpdateTx modifies an existing story inside an existing transaction.
func (r *StoryRepo) UpdateTx(ctx context.Context, tx pgx.Tx, story *domain.Story) (*domain.Story, error) {
	return r.update(ctx, tx, story)
}

// UpdateStatusTx updates only a story status inside an existing transaction.
func (r *StoryRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, status domain.StoryStatus) (*domain.Story, error) {
	query := `
		UPDATE story
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status, created_at, updated_at`

	var s domain.Story
	err := tx.QueryRow(ctx, query, id, status).Scan(
		&s.ID, &s.POIID, &s.Language, &s.Text, &s.AudioURL, &s.DurationSec,
		&s.LayerType, &s.OrderIndex, &s.IsInflation, &s.Confidence, &s.Sources,
		&s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("story_repo: update status tx: %w", err)
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
		SELECT s.id, s.poi_id,
			CASE WHEN $2 = 'ru' AND p.name_ru IS NOT NULL AND p.name_ru != '' THEN p.name_ru ELSE p.name END,
			s.audio_url, s.duration_sec
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
