package domain

import "time"

// InflationJobStatus represents the processing status of an inflation job.
type InflationJobStatus string

const (
	InflationJobStatusPending   InflationJobStatus = "pending"
	InflationJobStatusRunning   InflationJobStatus = "running"
	InflationJobStatusCompleted InflationJobStatus = "completed"
	InflationJobStatusFailed    InflationJobStatus = "failed"
	InflationJobStatusDead      InflationJobStatus = "dead"
)

// InflationTriggerType represents what triggered the inflation job.
type InflationTriggerType string

const (
	InflationTriggerUserProximity InflationTriggerType = "user_proximity"
	InflationTriggerAdminManual   InflationTriggerType = "admin_manual"
)

// InflationJob represents a content generation job for a POI.
type InflationJob struct {
	ID            int                  `json:"id"`
	POIID         int                  `json:"poi_id"`
	Status        InflationJobStatus   `json:"status"`
	TriggerType   InflationTriggerType `json:"trigger_type"`
	SegmentsCount int16                `json:"segments_count"`
	MaxSegments   int16                `json:"max_segments"`
	StartedAt     *time.Time           `json:"started_at"`
	CompletedAt   *time.Time           `json:"completed_at"`
	ErrorLog      *string              `json:"error_log"`
	CreatedAt     time.Time            `json:"created_at"`
	HeartbeatAt   *time.Time           `json:"heartbeat_at"`
	WorkerID      *string              `json:"worker_id"`
	Attempts      int16                `json:"attempts"`
}
