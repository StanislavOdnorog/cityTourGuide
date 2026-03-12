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

// InflationRepo handles database operations for inflation jobs.
type InflationRepo struct {
	pool *pgxpool.Pool
}

// NewInflationRepo creates a new InflationRepo.
func NewInflationRepo(pool *pgxpool.Pool) *InflationRepo {
	return &InflationRepo{pool: pool}
}

// inflationJobColumns is the canonical column list for inflation_job queries.
const inflationJobColumns = `id, poi_id, status, trigger_type, segments_count, max_segments,
	started_at, completed_at, error_log, created_at, heartbeat_at, worker_id, attempts`

// scanInflationJob scans a row into an InflationJob struct.
func scanInflationJob(row pgx.Row) (domain.InflationJob, error) {
	var j domain.InflationJob
	err := row.Scan(
		&j.ID, &j.POIID, &j.Status, &j.TriggerType,
		&j.SegmentsCount, &j.MaxSegments,
		&j.StartedAt, &j.CompletedAt, &j.ErrorLog, &j.CreatedAt,
		&j.HeartbeatAt, &j.WorkerID, &j.Attempts,
	)
	return j, err
}

// scanInflationJobs scans multiple rows into InflationJob slices.
func scanInflationJobs(rows pgx.Rows) ([]domain.InflationJob, error) {
	var jobs []domain.InflationJob
	for rows.Next() {
		var j domain.InflationJob
		if err := rows.Scan(
			&j.ID, &j.POIID, &j.Status, &j.TriggerType,
			&j.SegmentsCount, &j.MaxSegments,
			&j.StartedAt, &j.CompletedAt, &j.ErrorLog, &j.CreatedAt,
			&j.HeartbeatAt, &j.WorkerID, &j.Attempts,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// Create inserts a new inflation job.
func (r *InflationRepo) Create(ctx context.Context, job *domain.InflationJob) (*domain.InflationJob, error) {
	query := `
		INSERT INTO inflation_job (poi_id, status, trigger_type, segments_count, max_segments)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + inflationJobColumns

	created := &domain.InflationJob{}
	err := r.pool.QueryRow(ctx, query,
		job.POIID, job.Status, job.TriggerType, job.SegmentsCount, job.MaxSegments,
	).Scan(
		&created.ID, &created.POIID, &created.Status, &created.TriggerType,
		&created.SegmentsCount, &created.MaxSegments,
		&created.StartedAt, &created.CompletedAt, &created.ErrorLog, &created.CreatedAt,
		&created.HeartbeatAt, &created.WorkerID, &created.Attempts,
	)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: create: %w", err)
	}

	return created, nil
}

// GetByPOIID returns all inflation jobs for a specific POI.
func (r *InflationRepo) GetByPOIID(ctx context.Context, poiID int) ([]domain.InflationJob, error) {
	query := `
		SELECT ` + inflationJobColumns + `
		FROM inflation_job
		WHERE poi_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, poiID)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: get by poi_id: %w", err)
	}
	defer rows.Close()

	jobs, err := scanInflationJobs(rows)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: scan: %w", err)
	}
	return jobs, nil
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

// GetByID returns a single inflation job by its ID.
func (r *InflationRepo) GetByID(ctx context.Context, id int) (*domain.InflationJob, error) {
	query := `
		SELECT ` + inflationJobColumns + `
		FROM inflation_job
		WHERE id = $1`

	j, err := scanInflationJob(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("inflation_repo: get by id: %w", err)
	}

	return &j, nil
}

// GetPendingJobs returns up to `limit` jobs with status='pending', ordered by created_at ASC.
func (r *InflationRepo) GetPendingJobs(ctx context.Context, limit int) ([]domain.InflationJob, error) {
	query := `
		SELECT ` + inflationJobColumns + `
		FROM inflation_job
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: get pending jobs: %w", err)
	}
	defer rows.Close()

	jobs, err := scanInflationJobs(rows)
	if err != nil {
		return nil, fmt.Errorf("inflation_repo: scan pending: %w", err)
	}
	return jobs, nil
}

// ClaimNextJob atomically finds and claims the next available job.
// A job is claimable if it is pending, or if it is running with a heartbeat
// older than the lease window (indicating a crashed/hung worker).
// Returns nil, nil when no claimable job exists.
func (r *InflationRepo) ClaimNextJob(ctx context.Context, workerID string, leaseWindow time.Duration) (*domain.InflationJob, error) {
	now := time.Now()
	leaseExpiry := now.Add(-leaseWindow)

	query := `
		UPDATE inflation_job
		SET status = 'running', started_at = $1, heartbeat_at = $1, worker_id = $2,
		    attempts = attempts + 1
		WHERE id = (
			SELECT id FROM inflation_job
			WHERE status = 'pending'
			   OR (status = 'running' AND heartbeat_at < $3)
			ORDER BY
				CASE WHEN status = 'pending' THEN 0 ELSE 1 END,
				created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING ` + inflationJobColumns

	j, err := scanInflationJob(r.pool.QueryRow(ctx, query, now, workerID, leaseExpiry))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("inflation_repo: claim next job: %w", err)
	}

	return &j, nil
}

// UpdateHeartbeat refreshes the heartbeat timestamp for a running job.
func (r *InflationRepo) UpdateHeartbeat(ctx context.Context, jobID int) error {
	query := `
		UPDATE inflation_job
		SET heartbeat_at = $2
		WHERE id = $1 AND status = 'running'`

	result, err := r.pool.Exec(ctx, query, jobID, time.Now())
	if err != nil {
		return fmt.Errorf("inflation_repo: update heartbeat: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// SetRunning atomically marks a pending job as running.
// Returns ErrNotFound if the job is no longer pending (another worker picked it up).
func (r *InflationRepo) SetRunning(ctx context.Context, jobID int) error {
	now := time.Now()
	query := `
		UPDATE inflation_job
		SET status = 'running', started_at = $2, heartbeat_at = $2,
		    attempts = attempts + 1
		WHERE id = $1 AND status = 'pending'`

	result, err := r.pool.Exec(ctx, query, jobID, now)
	if err != nil {
		return fmt.Errorf("inflation_repo: set running: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// SetCompleted marks a running job as completed and increments segments_count.
func (r *InflationRepo) SetCompleted(ctx context.Context, jobID int) error {
	query := `
		UPDATE inflation_job
		SET status = 'completed', completed_at = $2, segments_count = segments_count + 1,
		    heartbeat_at = NULL
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, jobID, time.Now())
	if err != nil {
		return fmt.Errorf("inflation_repo: set completed: %w", err)
	}

	return nil
}

// SetFailed marks a job as failed with an error message.
func (r *InflationRepo) SetFailed(ctx context.Context, jobID int, errMsg string) error {
	query := `
		UPDATE inflation_job
		SET status = 'failed', completed_at = $2, error_log = $3,
		    heartbeat_at = NULL
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, jobID, time.Now(), errMsg)
	if err != nil {
		return fmt.Errorf("inflation_repo: set failed: %w", err)
	}

	return nil
}

// ReclaimStaleJobs resets stale running jobs back to pending so they can be retried.
// Jobs whose heartbeat is older than staleThreshold are considered stale.
// Jobs that have already been attempted maxAttempts times are moved to 'dead' status.
// Returns the number of jobs reclaimed (set back to pending).
func (r *InflationRepo) ReclaimStaleJobs(ctx context.Context, staleThreshold time.Duration, maxAttempts int) (int, error) {
	now := time.Now()
	cutoff := now.Add(-staleThreshold)

	// First, mark jobs exceeding max attempts as dead.
	deadQuery := `
		UPDATE inflation_job
		SET status = 'dead', completed_at = $1, heartbeat_at = NULL,
		    error_log = COALESCE(error_log, '') || 'exceeded max attempts'
		WHERE id IN (
			SELECT id FROM inflation_job
			WHERE status = 'running'
			  AND heartbeat_at < $2
			  AND attempts >= $3
			FOR UPDATE SKIP LOCKED
		)`

	_, err := r.pool.Exec(ctx, deadQuery, now, cutoff, maxAttempts)
	if err != nil {
		return 0, fmt.Errorf("inflation_repo: reclaim stale (mark dead): %w", err)
	}

	// Then, reclaim remaining stale jobs (under max attempts) back to pending.
	reclaimQuery := `
		UPDATE inflation_job
		SET status = 'pending', heartbeat_at = NULL, worker_id = NULL
		WHERE id IN (
			SELECT id FROM inflation_job
			WHERE status = 'running'
			  AND heartbeat_at < $1
			  AND attempts < $2
			FOR UPDATE SKIP LOCKED
		)`

	result, err := r.pool.Exec(ctx, reclaimQuery, cutoff, maxAttempts)
	if err != nil {
		return 0, fmt.Errorf("inflation_repo: reclaim stale (reset pending): %w", err)
	}

	return int(result.RowsAffected()), nil
}
