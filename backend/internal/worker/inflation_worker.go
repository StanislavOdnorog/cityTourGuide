package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
	"github.com/saas/city-stories-guide/backend/internal/platform/s3"
)

const (
	defaultPollInterval = 10 * time.Second
	defaultBatchSize    = 5
	maxSegmentsPerPOI   = 3
)

// InflationRepo defines the repository interface used by the inflation worker.
type InflationRepo interface {
	GetPendingJobs(ctx context.Context, limit int) ([]domain.InflationJob, error)
	SetRunning(ctx context.Context, jobID int) error
	SetCompleted(ctx context.Context, jobID int) error
	SetFailed(ctx context.Context, jobID int, errMsg string) error
	CountActiveByPOIID(ctx context.Context, poiID int) (int, error)
}

// StoryRepo defines the story repository interface used by the inflation worker.
type StoryRepo interface {
	Create(ctx context.Context, story *domain.Story) (*domain.Story, error)
	Update(ctx context.Context, story *domain.Story) (*domain.Story, error)
	CountByPOI(ctx context.Context, poiID int) (int, error)
}

// POIRepo defines the POI repository interface used by the inflation worker.
type POIRepo interface {
	GetByID(ctx context.Context, id int) (*domain.POI, error)
}

// InflationWorker polls for pending inflation jobs and processes them
// by generating stories via Claude API, converting to audio via ElevenLabs,
// and uploading to S3.
type InflationWorker struct {
	inflationRepo InflationRepo
	storyRepo     StoryRepo
	poiRepo       POIRepo
	claudeClient  *claude.Client
	ttsClient     *elevenlabs.Client
	s3Client      *s3.Client
	pollInterval  time.Duration
	batchSize     int
	jobTimeout    time.Duration
}

// Config holds optional configuration for the InflationWorker.
type Config struct {
	PollInterval time.Duration
	BatchSize    int
	JobTimeout   time.Duration
}

// NewInflationWorker creates a new inflation worker with all required dependencies.
func NewInflationWorker(
	inflationRepo InflationRepo,
	storyRepo StoryRepo,
	poiRepo POIRepo,
	claudeClient *claude.Client,
	ttsClient *elevenlabs.Client,
	s3Client *s3.Client,
	cfg *Config,
) *InflationWorker {
	pollInterval := defaultPollInterval
	batchSize := defaultBatchSize
	jobTimeout := 60 * time.Second
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
	}

	return &InflationWorker{
		inflationRepo: inflationRepo,
		storyRepo:     storyRepo,
		poiRepo:       poiRepo,
		claudeClient:  claudeClient,
		ttsClient:     ttsClient,
		s3Client:      s3Client,
		pollInterval:  pollInterval,
		batchSize:     batchSize,
		jobTimeout:    jobTimeout,
	}
}

// Start begins the polling loop. It blocks until the context is canceled.
func (w *InflationWorker) Start(ctx context.Context) error {
	log.Printf("Inflation worker started (poll interval: %s, batch size: %d)", w.pollInterval, w.batchSize)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Process immediately on start, then on each tick
	w.pollAndProcess(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Inflation worker stopping...")
			return nil
		case <-ticker.C:
			w.pollAndProcess(ctx)
		}
	}
}

// pollAndProcess fetches pending jobs and processes each one.
func (w *InflationWorker) pollAndProcess(ctx context.Context) {
	jobs, err := w.inflationRepo.GetPendingJobs(ctx, w.batchSize)
	if err != nil {
		log.Printf("Error fetching pending jobs: %v", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	log.Printf("Found %d pending inflation job(s)", len(jobs))

	for i := range jobs {
		if ctx.Err() != nil {
			return
		}

		jobCtx := context.WithoutCancel(ctx)
		deadlineCtx, cancel := context.WithTimeout(jobCtx, w.jobTimeout)
		w.processJob(deadlineCtx, &jobs[i])
		cancel()
	}
}

// processJob handles a single inflation job end-to-end:
// 1. Check segment limit
// 2. Mark as running
// 3. Fetch POI
// 4. Generate story text via Claude
// 5. Generate audio via ElevenLabs
// 6. Upload audio to S3
// 7. Save story to DB
// 8. Mark job as completed
func (w *InflationWorker) processJob(ctx context.Context, job *domain.InflationJob) {
	log.Printf("Processing inflation job %d for POI %d", job.ID, job.POIID)

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

	// Atomically claim the job (prevents double processing)
	if claimErr := w.inflationRepo.SetRunning(ctx, job.ID); claimErr != nil {
		log.Printf("Job %d already claimed by another worker: %v", job.ID, claimErr)
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
		w.failJob(ctx, job.ID, fmt.Sprintf("upload audio: %v", err))
		return
	}

	// Update story with audio URL
	created.AudioURL = &audioURL
	if _, err := w.storyRepo.Update(ctx, created); err != nil {
		log.Printf("Warning: job %d — could not update story %d with audio URL: %v", job.ID, created.ID, err)
	}

	// Mark job as completed
	if err := w.inflationRepo.SetCompleted(ctx, job.ID); err != nil {
		log.Printf("Error marking job %d as completed: %v", job.ID, err)
		return
	}

	log.Printf("Job %d completed: story %d created for POI %d (audio: %s)",
		job.ID, created.ID, job.POIID, audioURL)
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
