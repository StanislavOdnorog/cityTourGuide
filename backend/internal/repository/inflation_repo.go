package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// InflationRepo handles database operations for inflation jobs.
type InflationRepo struct {
	pool *pgxpool.Pool
}

// NewInflationRepo creates a new InflationRepo.
func NewInflationRepo(pool *pgxpool.Pool) *InflationRepo {
	return &InflationRepo{pool: pool}
}

// Create inserts a new inflation job.
func (r *InflationRepo) Create(ctx context.Context, job *domain.InflationJob) (*domain.InflationJob, error) {
	query := `
		INSERT INTO inflation_job (poi_id, status, trigger_type, segments_count, max_segments)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, poi_id, status, trigger_type, segments_count, max_segments,
		          started_at, completed_at, error_log, created_at`

	created := &domain.InflationJob{}
	err := r.pool.QueryRow(ctx, query,
		job.POIID, job.Status, job.TriggerType, job.SegmentsCount, job.MaxSegments,
	).Scan(
		&created.ID, &created.POIID, &created.Status, &created.TriggerType,
		&created.SegmentsCount, &created.MaxSegments,
		&created.StartedAt, &created.CompletedAt, &created.ErrorLog, &created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: create: %w", err)
	}

	return created, nil
}

// GetByPOIID returns all inflation jobs for a specific POI.
func (r *InflationRepo) GetByPOIID(ctx context.Context, poiID int) ([]domain.InflationJob, error) {
	query := `
		SELECT id, poi_id, status, trigger_type, segments_count, max_segments,
		       started_at, completed_at, error_log, created_at
		FROM inflation_job
		WHERE poi_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, poiID)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: get by poi_id: %w", err)
	}
	defer rows.Close()

	var jobs []domain.InflationJob
	for rows.Next() {
		var j domain.InflationJob
		if err := rows.Scan(
			&j.ID, &j.POIID, &j.Status, &j.TriggerType,
			&j.SegmentsCount, &j.MaxSegments,
			&j.StartedAt, &j.CompletedAt, &j.ErrorLog, &j.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("inflation_repo: scan: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// CountActiveByPOIID returns the number of inflation segments already generated for a POI.
func (r *InflationRepo) CountActiveByPOIID(ctx context.Context, poiID int) (int, error) {
	query := `SELECT COUNT(*) FROM inflation_job WHERE poi_id = $1 AND status IN ('pending', 'running', 'completed')`

	var count int
	err := r.pool.QueryRow(ctx, query, poiID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("inflation_repo: count active: %w", err)
	}

	return count, nil
}
