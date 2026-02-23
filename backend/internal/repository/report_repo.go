package repository

import (
	"context"
	"fmt"

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
