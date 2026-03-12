package worker

import (
	"context"
	"log"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

const (
	defaultCleanupPollInterval    = 30 * time.Second
	defaultCleanupBatchSize       = 20
	defaultCleanupMaxAttempts     = 5
	defaultCleanupReclaimInterval = 5 * time.Minute
)

// CleanupRepo defines the repository interface used by the cleanup worker.
type CleanupRepo interface {
	ClaimBatch(ctx context.Context, limit int) ([]domain.OrphanCleanupJob, error)
	MarkCompleted(ctx context.Context, id int) error
	MarkFailed(ctx context.Context, id int, errMsg string) error
	ReclaimFailed(ctx context.Context, maxAttempts int) (int, error)
}

// ObjectDeleter can delete objects from storage.
type ObjectDeleter interface {
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// CleanupWorkerConfig holds optional configuration for the CleanupWorker.
type CleanupWorkerConfig struct {
	PollInterval    time.Duration
	BatchSize       int
	MaxAttempts     int
	ReclaimInterval time.Duration
}

// CleanupWorker polls for orphan cleanup jobs and deletes the corresponding
// objects from storage. Deletion is idempotent — already-missing objects are
// treated as successful cleanup.
type CleanupWorker struct {
	repo            CleanupRepo
	storage         ObjectDeleter
	pollInterval    time.Duration
	batchSize       int
	maxAttempts     int
	reclaimInterval time.Duration
}

// NewCleanupWorker creates a new CleanupWorker.
func NewCleanupWorker(repo CleanupRepo, storage ObjectDeleter, cfg *CleanupWorkerConfig) *CleanupWorker {
	pollInterval := defaultCleanupPollInterval
	batchSize := defaultCleanupBatchSize
	maxAttempts := defaultCleanupMaxAttempts
	reclaimInterval := defaultCleanupReclaimInterval

	if cfg != nil {
		if cfg.PollInterval > 0 {
			pollInterval = cfg.PollInterval
		}
		if cfg.BatchSize > 0 {
			batchSize = cfg.BatchSize
		}
		if cfg.MaxAttempts > 0 {
			maxAttempts = cfg.MaxAttempts
		}
		if cfg.ReclaimInterval > 0 {
			reclaimInterval = cfg.ReclaimInterval
		}
	}

	return &CleanupWorker{
		repo:            repo,
		storage:         storage,
		pollInterval:    pollInterval,
		batchSize:       batchSize,
		maxAttempts:     maxAttempts,
		reclaimInterval: reclaimInterval,
	}
}

// Start begins the polling loop. It blocks until the context is canceled.
func (w *CleanupWorker) Start(ctx context.Context) error {
	log.Printf("Cleanup worker started (poll: %s, batch: %d, max_attempts: %d)",
		w.pollInterval, w.batchSize, w.maxAttempts)

	w.reclaimFailed(ctx)

	pollTicker := time.NewTicker(w.pollInterval)
	defer pollTicker.Stop()

	reclaimTicker := time.NewTicker(w.reclaimInterval)
	defer reclaimTicker.Stop()

	w.pollAndProcess(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup worker stopping...")
			return nil
		case <-reclaimTicker.C:
			w.reclaimFailed(ctx)
		case <-pollTicker.C:
			w.pollAndProcess(ctx)
		}
	}
}

func (w *CleanupWorker) reclaimFailed(ctx context.Context) {
	reclaimed, err := w.repo.ReclaimFailed(ctx, w.maxAttempts)
	if err != nil {
		log.Printf("Cleanup worker: error reclaiming failed jobs: %v", err)
		return
	}
	if reclaimed > 0 {
		log.Printf("Cleanup worker: reclaimed %d failed job(s)", reclaimed)
	}
}

func (w *CleanupWorker) pollAndProcess(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	jobs, err := w.repo.ClaimBatch(ctx, w.batchSize)
	if err != nil {
		log.Printf("Cleanup worker: error claiming batch: %v", err)
		return
	}

	for _, job := range jobs {
		if ctx.Err() != nil {
			return
		}
		w.processJob(ctx, job)
	}
}

func (w *CleanupWorker) processJob(ctx context.Context, job domain.OrphanCleanupJob) {
	err := w.storage.Delete(ctx, job.ObjectKey)
	if err != nil {
		// Check if the object is already gone — that counts as success.
		exists, existsErr := w.storage.Exists(ctx, job.ObjectKey)
		if existsErr == nil && !exists {
			// Object already deleted; mark completed.
			if markErr := w.repo.MarkCompleted(ctx, job.ID); markErr != nil {
				log.Printf("Cleanup worker: error marking job %d completed: %v", job.ID, markErr)
			}
			return
		}

		log.Printf("Cleanup worker: failed to delete %q (attempt %d): %v", job.ObjectKey, job.Attempts, err)
		if markErr := w.repo.MarkFailed(ctx, job.ID, err.Error()); markErr != nil {
			log.Printf("Cleanup worker: error marking job %d failed: %v", job.ID, markErr)
		}
		return
	}

	if err := w.repo.MarkCompleted(ctx, job.ID); err != nil {
		log.Printf("Cleanup worker: error marking job %d completed: %v", job.ID, err)
	}
}
