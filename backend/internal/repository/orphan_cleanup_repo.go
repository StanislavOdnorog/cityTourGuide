package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// OrphanCleanupRepo handles database operations for orphan cleanup jobs.
type OrphanCleanupRepo struct {
	pool *pgxpool.Pool
}

// NewOrphanCleanupRepo creates a new OrphanCleanupRepo.
func NewOrphanCleanupRepo(pool *pgxpool.Pool) *OrphanCleanupRepo {
	return &OrphanCleanupRepo{pool: pool}
}

// Enqueue inserts one or more pending cleanup jobs for the given object keys.
// Duplicate keys are accepted; the worker handles idempotent deletion.
func (r *OrphanCleanupRepo) Enqueue(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, key := range objectKeys {
		batch.Queue(`INSERT INTO orphan_cleanup_job (object_key) VALUES ($1)`, key)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range objectKeys {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("orphan_cleanup_repo: enqueue: %w", err)
		}
	}

	return nil
}

// EnqueueTx inserts cleanup jobs inside an existing transaction.
func (r *OrphanCleanupRepo) EnqueueTx(ctx context.Context, tx pgx.Tx, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	for _, key := range objectKeys {
		if _, err := tx.Exec(ctx, `INSERT INTO orphan_cleanup_job (object_key) VALUES ($1)`, key); err != nil {
			return fmt.Errorf("orphan_cleanup_repo: enqueue tx: %w", err)
		}
	}

	return nil
}

// ClaimBatch atomically claims up to limit pending jobs for processing.
// Jobs are locked with FOR UPDATE SKIP LOCKED to allow concurrent workers.
func (r *OrphanCleanupRepo) ClaimBatch(ctx context.Context, limit int) ([]domain.OrphanCleanupJob, error) {
	query := `
		UPDATE orphan_cleanup_job
		SET status = 'pending', attempts = attempts + 1, updated_at = NOW()
		WHERE id IN (
			SELECT id FROM orphan_cleanup_job
			WHERE status = 'pending'
			ORDER BY id
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, object_key, status, attempts, last_error, created_at, updated_at`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("orphan_cleanup_repo: claim batch: %w", err)
	}
	defer rows.Close()

	var jobs []domain.OrphanCleanupJob
	for rows.Next() {
		var j domain.OrphanCleanupJob
		if err := rows.Scan(&j.ID, &j.ObjectKey, &j.Status, &j.Attempts, &j.LastError, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("orphan_cleanup_repo: claim batch scan: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// MarkCompleted marks a job as completed.
func (r *OrphanCleanupRepo) MarkCompleted(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orphan_cleanup_job SET status = 'completed', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("orphan_cleanup_repo: mark completed: %w", err)
	}
	return nil
}

// MarkFailed marks a job as failed with an error message.
// Jobs with fewer than maxAttempts will be reset to pending on the next reclaim cycle.
func (r *OrphanCleanupRepo) MarkFailed(ctx context.Context, id int, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orphan_cleanup_job SET status = 'failed', last_error = $2, updated_at = NOW() WHERE id = $1`,
		id, errMsg)
	if err != nil {
		return fmt.Errorf("orphan_cleanup_repo: mark failed: %w", err)
	}
	return nil
}

// ReclaimFailed resets failed jobs with fewer than maxAttempts back to pending.
// Returns the number of reclaimed jobs.
func (r *OrphanCleanupRepo) ReclaimFailed(ctx context.Context, maxAttempts int) (int, error) {
	result, err := r.pool.Exec(ctx,
		`UPDATE orphan_cleanup_job SET status = 'pending', updated_at = NOW()
		 WHERE status = 'failed' AND attempts < $1`, maxAttempts)
	if err != nil {
		return 0, fmt.Errorf("orphan_cleanup_repo: reclaim failed: %w", err)
	}
	return int(result.RowsAffected()), nil
}
