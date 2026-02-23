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
	query := `
		SELECT id, story_id, user_id, type, comment,
		       user_lat, user_lng, status, resolved_at, created_at
		FROM report
		WHERE id = $1`

	var rep domain.Report
	err := r.pool.QueryRow(ctx, query, id).Scan(
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

// GetAll returns reports with optional status filter and pagination.
func (r *ReportRepo) GetAll(ctx context.Context, status string, page, perPage int) ([]domain.Report, int, error) {
	countQuery := `SELECT COUNT(*) FROM report`
	dataQuery := `
		SELECT id, story_id, user_id, type, comment,
		       user_lat, user_lng, status, resolved_at, created_at
		FROM report`

	var args []interface{}
	argIdx := 1

	if status != "" {
		filter := fmt.Sprintf(" WHERE status = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, status)
		argIdx++
	}

	dataQuery += " ORDER BY created_at DESC"
	dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("report_repo: count: %w", err)
	}

	offset := (page - 1) * perPage
	dataArgs := append(args, perPage, offset) //nolint:gocritic

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("report_repo: get all: %w", err)
	}
	defer rows.Close()

	var reports []domain.Report
	for rows.Next() {
		var rep domain.Report
		if err := rows.Scan(
			&rep.ID, &rep.StoryID, &rep.UserID, &rep.Type, &rep.Comment,
			&rep.UserLat, &rep.UserLng, &rep.Status, &rep.ResolvedAt, &rep.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("report_repo: scan: %w", err)
		}
		reports = append(reports, rep)
	}

	return reports, total, rows.Err()
}

// UpdateStatus updates a report's status and sets resolved_at when applicable.
func (r *ReportRepo) UpdateStatus(ctx context.Context, id int, status domain.ReportStatus) (*domain.Report, error) {
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
	err := r.pool.QueryRow(ctx, query, status, resolvedAt, id).Scan(
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
