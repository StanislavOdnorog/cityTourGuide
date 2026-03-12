package domain

import "time"

// OrphanCleanupStatus represents the processing state of a cleanup job.
type OrphanCleanupStatus string

const (
	OrphanCleanupPending   OrphanCleanupStatus = "pending"
	OrphanCleanupCompleted OrphanCleanupStatus = "completed"
	OrphanCleanupFailed    OrphanCleanupStatus = "failed"
)

// OrphanCleanupJob represents a queued request to delete an orphaned object
// from storage (e.g. an audio file whose owning story was deleted or replaced).
type OrphanCleanupJob struct {
	ID        int                 `json:"id"`
	ObjectKey string              `json:"object_key"`
	Status    OrphanCleanupStatus `json:"status"`
	Attempts  int                 `json:"attempts"`
	LastError *string             `json:"last_error,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}
