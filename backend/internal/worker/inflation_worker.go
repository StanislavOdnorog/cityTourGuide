package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
	"github.com/saas/city-stories-guide/backend/internal/platform/s3"
)

const (
	defaultPollInterval    = 10 * time.Second
	defaultBatchSize       = 5
	defaultLeaseWindow     = 5 * time.Minute
	defaultMaxAttempts     = 3
	defaultReclaimInterval = 5 * time.Minute
	maxSegmentsPerPOI      = 3
)

// InflationRepo defines the repository interface used by the inflation worker.
type InflationRepo interface {
	ClaimNextJob(ctx context.Context, workerID string, leaseWindow time.Duration) (*domain.InflationJob, error)
	UpdateHeartbeat(ctx context.Context, jobID int) error
	SetCompleted(ctx context.Context, jobID int) error
	SetFailed(ctx context.Context, jobID int, errMsg string) error
	CountActiveByPOIID(ctx context.Context, poiID int) (int, error)
	ReclaimStaleJobs(ctx context.Context, staleThreshold time.Duration, maxAttempts int) (int, error)
}

// StoryRepo defines the story repository interface used by the inflation worker.
type StoryRepo interface {
	Create(ctx context.Context, story *domain.Story) (*domain.Story, error)
	Update(ctx context.Context, story *domain.Story) (*domain.Story, error)
	Delete(ctx context.Context, id int) error
	CountByPOI(ctx context.Context, poiID int) (int, error)
}

// POIRepo defines the POI repository interface used by the inflation worker.
type POIRepo interface {
	GetByID(ctx context.Context, id int) (*domain.POI, error)
}

// StoryGenerator defines the Claude dependency used by the inflation worker.
type StoryGenerator interface {
	GenerateStory(ctx context.Context, poi *domain.POI, language string) (*claude.StoryResult, error)
}

// AudioGenerator defines the TTS dependency used by the inflation worker.
type AudioGenerator interface {
	GenerateAudio(ctx context.Context, text, language string) (*elevenlabs.AudioResult, error)
}

// ObjectStorage defines the object storage dependency used by the inflation worker.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
}

// InflationWorker polls for pending inflation jobs and processes them
// by generating stories via Claude API, converting to audio via ElevenLabs,
// and uploading to S3.
type InflationWorker struct {
	inflationRepo     InflationRepo
	storyRepo         StoryRepo
	poiRepo           POIRepo
	claudeClient      StoryGenerator
	ttsClient         AudioGenerator
	s3Client          ObjectStorage
	pollInterval      time.Duration
	batchSize         int
	jobTimeout        time.Duration
	leaseWindow       time.Duration
	heartbeatInterval time.Duration
	maxAttempts       int
	reclaimInterval   time.Duration
	workerID          string
}

// Config holds optional configuration for the InflationWorker.
type Config struct {
	PollInterval    time.Duration
	BatchSize       int
	JobTimeout      time.Duration
	LeaseWindow     time.Duration
	MaxAttempts     int
	ReclaimInterval time.Duration
	WorkerID        string
}

// NewInflationWorker creates a new inflation worker with all required dependencies.
func NewInflationWorker(
	inflationRepo InflationRepo,
	storyRepo StoryRepo,
	poiRepo POIRepo,
	claudeClient StoryGenerator,
	ttsClient AudioGenerator,
	s3Client ObjectStorage,
	cfg *Config,
) *InflationWorker {
	pollInterval := defaultPollInterval
	batchSize := defaultBatchSize
	jobTimeout := 60 * time.Second
	leaseWindow := defaultLeaseWindow
	maxAttempts := defaultMaxAttempts
	reclaimInterval := defaultReclaimInterval
	workerID := defaultWorkerID()

	if cfg != nil {
		if cfg.PollInterval > 0 {
			pollInterval = cfg.PollInterval
		}
		if cfg.BatchSize > 0 {
			batchSize = cfg.BatchSize
		}
		if cfg.JobTimeout > 0 {
			jobTimeout = cfg.JobTimeout
		}
		if cfg.LeaseWindow > 0 {
			leaseWindow = cfg.LeaseWindow
		}
		if cfg.MaxAttempts > 0 {
			maxAttempts = cfg.MaxAttempts
		}
		if cfg.ReclaimInterval > 0 {
			reclaimInterval = cfg.ReclaimInterval
		}
		if cfg.WorkerID != "" {
			workerID = cfg.WorkerID
		}
	}

	// Heartbeat at 1/3 of lease window to ensure liveness well before expiry.
	heartbeatInterval := leaseWindow / 3

	return &InflationWorker{
		inflationRepo:     inflationRepo,
		storyRepo:         storyRepo,
		poiRepo:           poiRepo,
		claudeClient:      claudeClient,
		ttsClient:         ttsClient,
		s3Client:          s3Client,
		pollInterval:      pollInterval,
		batchSize:         batchSize,
		jobTimeout:        jobTimeout,
		leaseWindow:       leaseWindow,
		heartbeatInterval: heartbeatInterval,
		maxAttempts:       maxAttempts,
		reclaimInterval:   reclaimInterval,
		workerID:          workerID,
	}
}

// Start begins the polling loop. It blocks until the context is canceled.
func (w *InflationWorker) Start(ctx context.Context) error {
	log.Printf("Inflation worker started (id: %s, poll: %s, batch: %d, lease: %s, max_attempts: %d)",
		w.workerID, w.pollInterval, w.batchSize, w.leaseWindow, w.maxAttempts)

	// Reclaim stale jobs from crashed workers on startup.
	w.reclaimStaleJobs(ctx)

	pollTicker := time.NewTicker(w.pollInterval)
	defer pollTicker.Stop()

	reclaimTicker := time.NewTicker(w.reclaimInterval)
	defer reclaimTicker.Stop()

	// Process immediately on start, then on each tick
	w.pollAndProcess(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Inflation worker stopping...")
			return nil
		case <-reclaimTicker.C:
			w.reclaimStaleJobs(ctx)
		case <-pollTicker.C:
			w.pollAndProcess(ctx)
		}
	}
}

// reclaimStaleJobs reclaims stuck jobs and logs the result.
func (w *InflationWorker) reclaimStaleJobs(ctx context.Context) {
	reclaimed, err := w.inflationRepo.ReclaimStaleJobs(ctx, w.leaseWindow, w.maxAttempts)
	if err != nil {
		log.Printf("Error reclaiming stale jobs: %v", err)
		return
	}
	if reclaimed > 0 {
		log.Printf("Reclaimed %d stale job(s)", reclaimed)
	}
}

// pollAndProcess claims and processes jobs up to batchSize.
func (w *InflationWorker) pollAndProcess(ctx context.Context) {
	for i := 0; i < w.batchSize; i++ {
		if ctx.Err() != nil {
			return
		}

		job, err := w.inflationRepo.ClaimNextJob(ctx, w.workerID, w.leaseWindow)
		if err != nil {
			log.Printf("Error claiming job: %v", err)
			return
		}
		if job == nil {
			return // no more claimable jobs
		}

		log.Printf("Claimed job %d for POI %d", job.ID, job.POIID)

		jobCtx := context.WithoutCancel(ctx)
		deadlineCtx, cancel := context.WithTimeout(jobCtx, w.jobTimeout)
		w.processJob(deadlineCtx, job)
		cancel()
	}
}

// processJob handles a single inflation job end-to-end:
// 1. Start heartbeat loop
// 2. Check segment limit
// 3. Fetch POI
// 4. Generate story text via Claude
// 5. Generate audio via ElevenLabs
// 6. Create story row
// 7. Upload audio to S3
// 8. Update story with audio URL
// 9. Mark job as completed
func (w *InflationWorker) processJob(ctx context.Context, job *domain.InflationJob) {
	log.Printf("Processing inflation job %d for POI %d", job.ID, job.POIID)

	// Start heartbeat goroutine — keeps the lease alive while processing.
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go w.heartbeatLoop(heartbeatCtx, job.ID)

	// Check segment limit before starting
	activeCount, err := w.inflationRepo.CountActiveByPOIID(ctx, job.POIID)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("count active segments: %v", err))
		return
	}

	if activeCount >= maxSegmentsPerPOI {
		w.failJob(ctx, job.ID, fmt.Sprintf("POI %d already has %d/%d inflation segments", job.POIID, activeCount, maxSegmentsPerPOI))
		return
	}

	// Fetch POI details
	poi, err := w.poiRepo.GetByID(ctx, job.POIID)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("fetch POI %d: %v", job.POIID, err))
		return
	}

	// Generate story text via Claude
	language := "en"
	storyResult, err := w.claudeClient.GenerateStory(ctx, poi, language)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("generate story: %v", err))
		return
	}

	log.Printf("Job %d: story generated (%d tokens in, %d tokens out, %s)",
		job.ID, storyResult.TokensIn, storyResult.TokensOut, storyResult.Duration)

	// Generate audio via ElevenLabs
	audioResult, err := w.ttsClient.GenerateAudio(ctx, storyResult.Text, language)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("generate audio: %v", err))
		return
	}

	log.Printf("Job %d: audio generated (%s)", job.ID, audioResult.Duration)

	// Determine order index for the new story
	existingCount, err := w.storyRepo.CountByPOI(ctx, job.POIID)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("count existing stories: %v", err))
		return
	}

	// Create story record first (to get the ID for the S3 key)
	story := &domain.Story{
		POIID:       job.POIID,
		Language:    language,
		Text:        storyResult.Text,
		LayerType:   storyResult.LayerType,
		OrderIndex:  clampInt16(existingCount),
		IsInflation: true,
		Confidence:  clampInt16(storyResult.Confidence),
		Sources:     json.RawMessage(`[]`),
		Status:      domain.StoryStatusActive,
	}

	created, err := w.storyRepo.Create(ctx, story)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("create story: %v", err))
		return
	}

	// Upload audio to S3
	audioKey := s3.AudioKey(poi.CityID, poi.ID, created.ID)
	audioURL, err := w.s3Client.Upload(ctx, audioKey, audioResult.Audio, "audio/mpeg")
	if err != nil {
		w.failJobWithCompensation(ctx, job, created, "", fmt.Sprintf("upload audio: %v", err))
		return
	}

	// Update story with audio URL
	created.AudioURL = &audioURL
	if _, err := w.storyRepo.Update(ctx, created); err != nil {
		w.failJobWithCompensation(ctx, job, created, audioKey, fmt.Sprintf("update story %d with audio URL %q: %v", created.ID, audioURL, err))
		return
	}

	// Mark job as completed (also clears heartbeat_at)
	if err := w.inflationRepo.SetCompleted(ctx, job.ID); err != nil {
		log.Printf("Error marking job %d as completed: %v", job.ID, err)
		return
	}

	log.Printf("Job %d completed: story %d created for POI %d (audio: %s)",
		job.ID, created.ID, job.POIID, audioURL)
}

// heartbeatLoop periodically updates the heartbeat timestamp for a running job.
// It stops when the context is canceled (i.e., when processJob finishes).
func (w *InflationWorker) heartbeatLoop(ctx context.Context, jobID int) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.inflationRepo.UpdateHeartbeat(ctx, jobID); err != nil {
				log.Printf("Failed to update heartbeat for job %d: %v", jobID, err)
			}
		}
	}
}

func (w *InflationWorker) failJobWithCompensation(ctx context.Context, job *domain.InflationJob, story *domain.Story, audioKey string, errMsg string) {
	if story != nil {
		if err := w.compensateFailedStory(ctx, job, story, audioKey); err != nil {
			log.Printf("Job %d compensation failed for story %d (poi %d, audio_key=%q): %v", job.ID, story.ID, job.POIID, audioKey, err)
		}
	}

	w.failJob(ctx, job.ID, errMsg)
}

func (w *InflationWorker) compensateFailedStory(ctx context.Context, job *domain.InflationJob, story *domain.Story, audioKey string) error {
	var cleanupErrs []string

	if audioKey != "" {
		if err := w.s3Client.Delete(ctx, audioKey); err != nil {
			cleanupErrs = append(cleanupErrs, fmt.Sprintf("delete uploaded audio %q for story %d: %v", audioKey, story.ID, err))
		}
	}

	if err := w.storyRepo.Delete(ctx, story.ID); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Sprintf("delete created story %d for job %d: %v", story.ID, job.ID, err))
	}

	if len(cleanupErrs) > 0 {
		return errors.New(strings.Join(cleanupErrs, "; "))
	}

	log.Printf("Job %d compensation succeeded: removed story %d for POI %d", job.ID, story.ID, job.POIID)
	return nil
}

// failJob marks a job as failed and logs the error.
func (w *InflationWorker) failJob(ctx context.Context, jobID int, errMsg string) {
	log.Printf("Job %d failed: %s", jobID, errMsg)
	if err := w.inflationRepo.SetFailed(ctx, jobID, errMsg); err != nil {
		log.Printf("Error marking job %d as failed: %v", jobID, err)
	}
}

// clampInt16 safely converts an int to int16, clamping to the int16 range.
func clampInt16(v int) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v) //nolint:gosec // value is clamped above
}

func defaultWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}
