package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// ReportRepo handles database operations for reports.
type ReportRepo struct {
	pool *pgxpool.Pool
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// NewReportRepo creates a new ReportRepo.
func NewReportRepo(pool *pgxpool.Pool) *ReportRepo {
	return &ReportRepo{pool: pool}
}

// Create inserts a new report into the database.
func (r *ReportRepo) Create(ctx context.Context, storyID int, userID string, reportType domain.ReportType, comment *string, lat, lng *float64) (*domain.Report, error) {
	query := `
		INSERT INTO report (story_id, user_id, type, comment, user_lat, user_lng)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, story_id, user_id, type, comment, user_lat, user_lng, status, resolved_at, created_at`

	var rep domain.Report
	err := r.pool.QueryRow(ctx, query, storyID, userID, reportType, comment, lat, lng).Scan(
		&rep.ID, &rep.StoryID, &rep.UserID, &rep.Type, &rep.Comment,
		&rep.UserLat, &rep.UserLng, &rep.Status, &rep.ResolvedAt, &rep.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("report_repo: create: %w", err)
	}

	return &rep, nil
}

// GetByID returns a single report by ID.
func (r *ReportRepo) GetByID(ctx context.Context, id int) (*domain.Report, error) {
	return r.getByID(ctx, r.pool, id, "")
}

func (r *ReportRepo) getByID(ctx context.Context, q queryRower, id int, suffix string) (*domain.Report, error) {
	query := `
		SELECT id, story_id, user_id, type, comment,
		       user_lat, user_lng, status, resolved_at, created_at
		FROM report
		WHERE id = $1 ` + suffix

	var rep domain.Report
	err := q.QueryRow(ctx, query, id).Scan(
		&rep.ID, &rep.StoryID, &rep.UserID, &rep.Type, &rep.Comment,
		&rep.UserLat, &rep.UserLng, &rep.Status, &rep.ResolvedAt, &rep.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("report_repo: get by id: %w", err)
	}

	return &rep, nil
}

// GetByIDForUpdateTx returns a single report by ID and locks it for update.
func (r *ReportRepo) GetByIDForUpdateTx(ctx context.Context, tx pgx.Tx, id int) (*domain.Report, error) {
	return r.getByID(ctx, tx, id, "FOR UPDATE")
}

// List returns reports with cursor-based pagination, ordered by id ASC.
func (r *ReportRepo) List(ctx context.Context, status string, page domain.PageRequest) (*domain.PageResponse[domain.Report], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("report_repo: list: %w", err)
	}

	query := `
		SELECT id, story_id, user_id, type, comment,
		       user_lat, user_lng, status, resolved_at, created_at
		FROM report`

	args := []interface{}{}
	argIdx := 1
	conditions := []string{}

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if page.Cursor != "" {
		cursorID, err := domain.DecodeCursor(page.Cursor)
		if err != nil {
			return nil, fmt.Errorf("report_repo: list: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("id > $%d", argIdx))
		args = append(args, cursorID)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, cond := range conditions[1:] {
			query += " AND " + cond
		}
	}

	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("report_repo: list: %w", err)
	}
	defer rows.Close()

	var reports []domain.Report
	for rows.Next() {
		var rep domain.Report
		if err := rows.Scan(
			&rep.ID, &rep.StoryID, &rep.UserID, &rep.Type, &rep.Comment,
			&rep.UserLat, &rep.UserLng, &rep.Status, &rep.ResolvedAt, &rep.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("report_repo: list scan: %w", err)
		}
		reports = append(reports, rep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("report_repo: list rows: %w", err)
	}

	hasMore := len(reports) > page.Limit
	if hasMore {
		reports = reports[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(reports) > 0 {
		nextCursor = domain.EncodeCursor(reports[len(reports)-1].ID)
	}

	return &domain.PageResponse[domain.Report]{
		Items:      reports,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// ListAdmin returns reports with joined POI/story context for admin listing.
func (r *ReportRepo) ListAdmin(ctx context.Context, status string, page domain.PageRequest, sort ListSort) (*domain.PageResponse[domain.AdminReportListItem], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("report_repo: list admin: %w", err)
	}

	resolvedSort, err := ResolveSort(sort, map[string]SortColumn{
		"id":         {Key: "id", Column: "r.id", Type: SortValueInt},
		"story_id":   {Key: "story_id", Column: "r.story_id", Type: SortValueInt},
		"type":       {Key: "type", Column: "r.type", Type: SortValueString},
		"status":     {Key: "status", Column: "r.status", Type: SortValueString},
		"created_at": {Key: "created_at", Column: "r.created_at", Type: SortValueTime},
	}, "id", SortDirAsc)
	if err != nil {
		return nil, fmt.Errorf("report_repo: list admin: %w", err)
	}

	query := `
		SELECT r.id, r.story_id, r.user_id, r.type, r.comment,
		       r.user_lat, r.user_lng, r.status, r.resolved_at, r.created_at,
		       s.poi_id, p.name, s.language, s.status
		FROM report r
		LEFT JOIN story s ON s.id = r.story_id
		LEFT JOIN poi p ON p.id = s.poi_id`

	args := []interface{}{}
	argIdx := 1
	conditions := []string{}

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	cursorCondition, cursorArgs, err := resolvedSort.CursorCondition(page.Cursor, argIdx)
	if err != nil {
		return nil, fmt.Errorf("report_repo: list admin: %w", err)
	}
	if cursorCondition != "" {
		conditions = append(conditions, cursorCondition)
		args = append(args, cursorArgs...)
		argIdx += len(cursorArgs)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, cond := range conditions[1:] {
			query += " AND " + cond
		}
	}

	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d", resolvedSort.OrderBy(), argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("report_repo: list admin: %w", err)
	}
	defer rows.Close()

	var items []domain.AdminReportListItem
	for rows.Next() {
		var item domain.AdminReportListItem
		if err := rows.Scan(
			&item.ID, &item.StoryID, &item.UserID, &item.Type, &item.Comment,
			&item.UserLat, &item.UserLng, &item.Status, &item.ResolvedAt, &item.CreatedAt,
			&item.POIID, &item.POIName, &item.StoryLanguage, &item.StoryStatus,
		); err != nil {
			return nil, fmt.Errorf("report_repo: list admin scan: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("report_repo: list admin rows: %w", err)
	}

	hasMore := len(items) > page.Limit
	if hasMore {
		items = items[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		nextCursor, err = EncodeOrderedCursor(resolvedSort, adminReportSortValue(items[len(items)-1], resolvedSort.Key), items[len(items)-1].ID)
		if err != nil {
			return nil, fmt.Errorf("report_repo: list admin: %w", err)
		}
	}

	return &domain.PageResponse[domain.AdminReportListItem]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func adminReportSortValue(item domain.AdminReportListItem, key string) interface{} {
	switch key {
	case "story_id":
		return item.StoryID
	case "type":
		return string(item.Type)
	case "status":
		return string(item.Status)
	case "created_at":
		return item.CreatedAt
	default:
		return item.ID
	}
}

// UpdateStatus updates a report's status and sets resolved_at when applicable.
func (r *ReportRepo) UpdateStatus(ctx context.Context, id int, status domain.ReportStatus) (*domain.Report, error) {
	return r.updateStatus(ctx, r.pool, id, status)
}

func (r *ReportRepo) updateStatus(ctx context.Context, q queryRower, id int, status domain.ReportStatus) (*domain.Report, error) {
	var resolvedAt *time.Time
	if status == domain.ReportStatusResolved || status == domain.ReportStatusDismissed {
		now := time.Now()
		resolvedAt = &now
	}

	query := `
		UPDATE report
		SET status = $1, resolved_at = $2
		WHERE id = $3
		RETURNING id, story_id, user_id, type, comment, user_lat, user_lng, status, resolved_at, created_at`

	var rep domain.Report
	err := q.QueryRow(ctx, query, status, resolvedAt, id).Scan(
		&rep.ID, &rep.StoryID, &rep.UserID, &rep.Type, &rep.Comment,
		&rep.UserLat, &rep.UserLng, &rep.Status, &rep.ResolvedAt, &rep.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("report_repo: update status: %w", err)
	}

	return &rep, nil
}

// UpdateStatusTx updates a report status inside an existing transaction.
func (r *ReportRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, status domain.ReportStatus) (*domain.Report, error) {
	return r.updateStatus(ctx, tx, id, status)
}

// ModerateDisableStory atomically disables the reported story and resolves the report.
// It retries once on transient PostgreSQL errors (serialization conflicts, deadlocks).
func (r *ReportRepo) ModerateDisableStory(ctx context.Context, id int) (*domain.ModeratedReportResult, error) {
	storyRepo := &StoryRepo{pool: r.pool}

	var result *domain.ModeratedReportResult

	err := RetryableTx(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		report, err := r.GetByIDForUpdateTx(ctx, tx, id)
		if err != nil {
			return err
		}

		story, err := storyRepo.GetByIDForUpdateTx(ctx, tx, report.StoryID)
		if err != nil {
			return err
		}

		if report.Status != domain.ReportStatusResolved && report.Status != domain.ReportStatusDismissed {
			if story.Status != domain.StoryStatusDisabled {
				story, err = storyRepo.UpdateStatusTx(ctx, tx, story.ID, domain.StoryStatusDisabled)
				if err != nil {
					return err
				}
			}

			report, err = r.UpdateStatusTx(ctx, tx, report.ID, domain.ReportStatusResolved)
			if err != nil {
				return err
			}
		}

		result = &domain.ModeratedReportResult{
			Report: *report,
			Story: domain.ModeratedStory{
				ID:       story.ID,
				POIID:    story.POIID,
				Language: story.Language,
				Status:   story.Status,
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetByPOIID returns all reports for stories belonging to a specific POI.
func (r *ReportRepo) GetByPOIID(ctx context.Context, poiID int) ([]domain.Report, error) {
	query := `
		SELECT r.id, r.story_id, r.user_id, r.type, r.comment,
		       r.user_lat, r.user_lng, r.status, r.resolved_at, r.created_at
		FROM report r
		INNER JOIN story s ON s.id = r.story_id
		WHERE s.poi_id = $1
		ORDER BY r.created_at DESC`

	rows, err := r.pool.Query(ctx, query, poiID)
	if err != nil {
		return nil, fmt.Errorf("report_repo: get by poi_id: %w", err)
	}
	defer rows.Close()

	var reports []domain.Report
	for rows.Next() {
		var rep domain.Report
		if err := rows.Scan(
			&rep.ID, &rep.StoryID, &rep.UserID, &rep.Type, &rep.Comment,
			&rep.UserLat, &rep.UserLng, &rep.Status, &rep.ResolvedAt, &rep.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("report_repo: scan: %w", err)
		}
		reports = append(reports, rep)
	}

	return reports, rows.Err()
}
