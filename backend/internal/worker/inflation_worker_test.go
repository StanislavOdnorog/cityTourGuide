package worker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
	"github.com/saas/city-stories-guide/backend/internal/platform/s3"
)

// --- mock repos ---

type mockInflationRepo struct {
	mu sync.Mutex

	// ClaimNextJob state
	claimableJobs []*domain.InflationJob // jobs returned by ClaimNextJob, in order
	claimCalls    int
	claimErr      error

	// heartbeat tracking
	heartbeatCalls map[int]int // jobID -> count of heartbeat updates
	heartbeatErr   error

	// active count
	activeCount    int
	activeCountErr error

	// completed / failed tracking
	completedJobs []int
	failedJobs    map[int]string

	// reclaim tracking
	reclaimCalls     int
	reclaimResult    int
	reclaimErr       error
	reclaimThreshold time.Duration
	reclaimMaxAtt    int
}

func newMockInflationRepo() *mockInflationRepo {
	return &mockInflationRepo{
		heartbeatCalls: make(map[int]int),
		failedJobs:     make(map[int]string),
	}
}

func (m *mockInflationRepo) ClaimNextJob(_ context.Context, _ string, _ time.Duration) (*domain.InflationJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.claimCalls++
	if m.claimErr != nil {
		return nil, m.claimErr
	}
	if len(m.claimableJobs) == 0 {
		return nil, nil
	}
	job := m.claimableJobs[0]
	m.claimableJobs = m.claimableJobs[1:]
	return job, nil
}

func (m *mockInflationRepo) UpdateHeartbeat(_ context.Context, jobID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.heartbeatErr != nil {
		return m.heartbeatErr
	}
	m.heartbeatCalls[jobID]++
	return nil
}

func (m *mockInflationRepo) SetCompleted(_ context.Context, jobID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completedJobs = append(m.completedJobs, jobID)
	return nil
}

func (m *mockInflationRepo) SetFailed(_ context.Context, jobID int, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedJobs[jobID] = errMsg
	return nil
}

func (m *mockInflationRepo) CountActiveByPOIID(_ context.Context, _ int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeCount, m.activeCountErr
}

func (m *mockInflationRepo) ReclaimStaleJobs(_ context.Context, threshold time.Duration, maxAttempts int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reclaimCalls++
	m.reclaimThreshold = threshold
	m.reclaimMaxAtt = maxAttempts
	return m.reclaimResult, m.reclaimErr
}

func (m *mockInflationRepo) getHeartbeatCount(jobID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.heartbeatCalls[jobID]
}

func (m *mockInflationRepo) getReclaimCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reclaimCalls
}

type mockStoryRepo struct {
	created    []*domain.Story
	createErr  error
	updated    []*domain.Story
	updateErr  error
	deletedIDs []int
	deleteErr  error
	storyCount int
	nextID     int
}

func (m *mockStoryRepo) Create(_ context.Context, story *domain.Story) (*domain.Story, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.nextID++
	result := *story
	result.ID = m.nextID
	result.CreatedAt = time.Now()
	result.UpdatedAt = time.Now()
	m.created = append(m.created, &result)
	return &result, nil
}

func (m *mockStoryRepo) Update(_ context.Context, story *domain.Story) (*domain.Story, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	result := *story
	result.UpdatedAt = time.Now()
	m.updated = append(m.updated, &result)
	return &result, nil
}

func (m *mockStoryRepo) Delete(_ context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

func (m *mockStoryRepo) CountByPOI(_ context.Context, _ int) (int, error) {
	return m.storyCount, nil
}

type mockPOIRepo struct {
	poi    *domain.POI
	getErr error
}

func (m *mockPOIRepo) GetByID(_ context.Context, _ int) (*domain.POI, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.poi, nil
}

type mockClaudeClient struct {
	result *claude.StoryResult
	err    error
}

func (m *mockClaudeClient) GenerateStory(_ context.Context, _ *domain.POI, _ string) (*claude.StoryResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockTTSClient struct {
	result *elevenlabs.AudioResult
	err    error
}

func (m *mockTTSClient) GenerateAudio(_ context.Context, _, _ string) (*elevenlabs.AudioResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockS3Client struct {
	uploadURL   string
	uploadErr   error
	uploadKeys  []string
	deletedKeys []string
	deleteErr   error
}

func (m *mockS3Client) Upload(_ context.Context, key string, _ io.Reader, _ string) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	m.uploadKeys = append(m.uploadKeys, key)
	return m.uploadURL, nil
}

func (m *mockS3Client) Delete(_ context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedKeys = append(m.deletedKeys, key)
	return nil
}

// --- tests ---

func TestClampInt16(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int16
	}{
		{"zero", 0, 0},
		{"positive", 42, 42},
		{"negative", -5, -5},
		{"max_int16", 32767, 32767},
		{"over_max", 40000, 32767},
		{"min_int16", -32768, -32768},
		{"under_min", -40000, -32768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampInt16(tt.input)
			if result != tt.expected {
				t.Errorf("clampInt16(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewInflationWorker_Defaults(t *testing.T) {
	w := NewInflationWorker(
		newMockInflationRepo(),
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	if w.pollInterval != defaultPollInterval {
		t.Errorf("pollInterval = %v, want %v", w.pollInterval, defaultPollInterval)
	}
	if w.batchSize != defaultBatchSize {
		t.Errorf("batchSize = %d, want %d", w.batchSize, defaultBatchSize)
	}
	if w.jobTimeout != 60*time.Second {
		t.Errorf("jobTimeout = %v, want 60s", w.jobTimeout)
	}
	if w.leaseWindow != defaultLeaseWindow {
		t.Errorf("leaseWindow = %v, want %v", w.leaseWindow, defaultLeaseWindow)
	}
	if w.heartbeatInterval != defaultLeaseWindow/3 {
		t.Errorf("heartbeatInterval = %v, want %v", w.heartbeatInterval, defaultLeaseWindow/3)
	}
	if w.workerID == "" {
		t.Error("workerID should not be empty")
	}
}

func TestNewInflationWorker_CustomConfig(t *testing.T) {
	w := NewInflationWorker(
		newMockInflationRepo(),
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{
			PollInterval: 30 * time.Second,
			BatchSize:    10,
			JobTimeout:   45 * time.Second,
			LeaseWindow:  3 * time.Minute,
			WorkerID:     "test-worker-1",
		},
	)

	if w.pollInterval != 30*time.Second {
		t.Errorf("pollInterval = %v, want 30s", w.pollInterval)
	}
	if w.batchSize != 10 {
		t.Errorf("batchSize = %d, want 10", w.batchSize)
	}
	if w.jobTimeout != 45*time.Second {
		t.Errorf("jobTimeout = %v, want 45s", w.jobTimeout)
	}
	if w.leaseWindow != 3*time.Minute {
		t.Errorf("leaseWindow = %v, want 3m", w.leaseWindow)
	}
	if w.heartbeatInterval != time.Minute {
		t.Errorf("heartbeatInterval = %v, want 1m", w.heartbeatInterval)
	}
	if w.workerID != "test-worker-1" {
		t.Errorf("workerID = %q, want %q", w.workerID, "test-worker-1")
	}
}

func TestPollAndProcess_NoClaimableJobs(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	ctx := context.Background()
	w.pollAndProcess(ctx)

	if inflationRepo.claimCalls != 1 {
		t.Errorf("claimCalls = %d, want 1", inflationRepo.claimCalls)
	}
}

func TestPollAndProcess_ClaimError(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	inflationRepo.claimErr = errors.New("db connection failed")

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	w.pollAndProcess(context.Background())

	if inflationRepo.claimCalls != 1 {
		t.Errorf("claimCalls = %d, want 1", inflationRepo.claimCalls)
	}
}

func TestProcessJob_MaxSegmentsReached(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	inflationRepo.activeCount = 3 // already at max

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	job := &domain.InflationJob{
		ID:    1,
		POIID: 100,
	}

	w.processJob(context.Background(), job)

	if _, ok := inflationRepo.failedJobs[1]; !ok {
		t.Error("expected job 1 to be marked as failed")
	}
}

func TestProcessJob_POINotFound(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	poiRepo := &mockPOIRepo{
		getErr: errors.New("poi not found"),
	}

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		poiRepo,
		nil, nil, nil,
		nil,
	)

	job := &domain.InflationJob{
		ID:    3,
		POIID: 999,
	}

	w.processJob(context.Background(), job)

	if errMsg, ok := inflationRepo.failedJobs[3]; !ok {
		t.Error("expected job 3 to be marked as failed")
	} else if errMsg == "" {
		t.Error("expected error message for failed job")
	}
}

func TestProcessJob_PartialFailureCompensation(t *testing.T) {
	tests := []struct {
		name               string
		s3UploadErr        error
		storyUpdateErr     error
		storyDeleteErr     error
		s3DeleteErr        error
		wantFailedContains string
		wantDeleteStory    bool
		wantDeleteAudio    bool
	}{
		{
			name:               "upload failure after story creation",
			s3UploadErr:        errors.New("s3 unavailable"),
			wantFailedContains: "upload audio: s3 unavailable",
			wantDeleteStory:    true,
			wantDeleteAudio:    false,
		},
		{
			name:               "failure after upload still deletes story when audio cleanup fails",
			storyUpdateErr:     errors.New("update failed"),
			s3DeleteErr:        errors.New("delete audio failed"),
			wantFailedContains: "update story",
			wantDeleteStory:    true,
			wantDeleteAudio:    false,
		},
		{
			name:               "final story update failure after upload",
			storyUpdateErr:     errors.New("update failed"),
			wantFailedContains: "update story",
			wantDeleteStory:    true,
			wantDeleteAudio:    true,
		},
		{
			name:               "cleanup logs when story delete fails after upload failure",
			s3UploadErr:        errors.New("s3 unavailable"),
			storyDeleteErr:     errors.New("delete story failed"),
			wantFailedContains: "upload audio: s3 unavailable",
			wantDeleteStory:    false,
			wantDeleteAudio:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inflationRepo := newMockInflationRepo()
			storyRepo := &mockStoryRepo{
				storyCount: 2,
				nextID:     100,
				updateErr:  tt.storyUpdateErr,
				deleteErr:  tt.storyDeleteErr,
			}
			poiRepo := &mockPOIRepo{
				poi: &domain.POI{
					ID:     55,
					CityID: 7,
				},
			}
			claudeClient := &mockClaudeClient{
				result: &claude.StoryResult{
					Text:       "Inflated story",
					LayerType:  domain.StoryLayerGeneral,
					Confidence: 87,
				},
			}
			ttsClient := &mockTTSClient{
				result: &elevenlabs.AudioResult{
					Audio:    bytes.NewReader([]byte("mp3")),
					Duration: 10 * time.Millisecond,
				},
			}
			s3Client := &mockS3Client{
				uploadURL: "https://cdn.example.com/audio/101.mp3",
				uploadErr: tt.s3UploadErr,
				deleteErr: tt.s3DeleteErr,
			}

			w := NewInflationWorker(
				inflationRepo,
				storyRepo,
				poiRepo,
				claudeClient,
				ttsClient,
				s3Client,
				nil,
			)

			job := &domain.InflationJob{
				ID:    10,
				POIID: 55,
			}

			w.processJob(context.Background(), job)

			errMsg, ok := inflationRepo.failedJobs[job.ID]
			if !ok {
				t.Fatalf("expected job %d to be marked failed", job.ID)
			}
			if !strings.Contains(errMsg, tt.wantFailedContains) {
				t.Fatalf("failed message = %q, want substring %q", errMsg, tt.wantFailedContains)
			}
			if len(inflationRepo.completedJobs) != 0 {
				t.Fatalf("expected no completed jobs, got %v", inflationRepo.completedJobs)
			}

			if got := len(storyRepo.created); got != 1 {
				t.Fatalf("created stories = %d, want 1", got)
			}

			if tt.wantDeleteStory {
				if len(storyRepo.deletedIDs) != 1 || storyRepo.deletedIDs[0] != storyRepo.created[0].ID {
					t.Fatalf("deleted story IDs = %v, want [%d]", storyRepo.deletedIDs, storyRepo.created[0].ID)
				}
			} else if len(storyRepo.deletedIDs) != 0 {
				t.Fatalf("deleted story IDs = %v, want none", storyRepo.deletedIDs)
			}

			if tt.storyUpdateErr != nil {
				if got := len(storyRepo.updated); got != 0 {
					t.Fatalf("successful updates = %d, want 0", got)
				}
			}

			wantAudioKey := s3.AudioKey(7, 55, storyRepo.created[0].ID)
			if tt.s3DeleteErr == nil && tt.wantDeleteAudio {
				if len(s3Client.deletedKeys) != 1 || s3Client.deletedKeys[0] != wantAudioKey {
					t.Fatalf("deleted audio keys = %v, want [%s]", s3Client.deletedKeys, wantAudioKey)
				}
			} else if len(s3Client.deletedKeys) != 0 {
				t.Fatalf("deleted audio keys = %v, want none", s3Client.deletedKeys)
			}
		})
	}
}

func TestProcessJob_Success(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	storyRepo := &mockStoryRepo{
		storyCount: 1,
		nextID:     200,
	}
	poiRepo := &mockPOIRepo{
		poi: &domain.POI{
			ID:     42,
			CityID: 9,
		},
	}
	claudeClient := &mockClaudeClient{
		result: &claude.StoryResult{
			Text:       "Fresh story",
			LayerType:  domain.StoryLayerHumanStory,
			Confidence: 91,
			TokensIn:   100,
			TokensOut:  40,
			Duration:   20 * time.Millisecond,
		},
	}
	ttsClient := &mockTTSClient{
		result: &elevenlabs.AudioResult{
			Audio:    bytes.NewReader([]byte("mp3")),
			Duration: 15 * time.Millisecond,
		},
	}
	s3Client := &mockS3Client{
		uploadURL: "https://cdn.example.com/audio/201.mp3",
	}

	w := NewInflationWorker(
		inflationRepo,
		storyRepo,
		poiRepo,
		claudeClient,
		ttsClient,
		s3Client,
		nil,
	)

	job := &domain.InflationJob{
		ID:    11,
		POIID: 42,
	}

	w.processJob(context.Background(), job)

	if len(inflationRepo.completedJobs) != 1 || inflationRepo.completedJobs[0] != job.ID {
		t.Fatalf("completed jobs = %v, want [%d]", inflationRepo.completedJobs, job.ID)
	}
	if len(inflationRepo.failedJobs) != 0 {
		t.Fatalf("failed jobs = %v, want none", inflationRepo.failedJobs)
	}
	if len(storyRepo.created) != 1 {
		t.Fatalf("created stories = %d, want 1", len(storyRepo.created))
	}
	if len(storyRepo.updated) != 1 {
		t.Fatalf("updated stories = %d, want 1", len(storyRepo.updated))
	}
	if len(storyRepo.deletedIDs) != 0 {
		t.Fatalf("deleted story IDs = %v, want none", storyRepo.deletedIDs)
	}
	if storyRepo.updated[0].AudioURL == nil || *storyRepo.updated[0].AudioURL != s3Client.uploadURL {
		t.Fatalf("updated audio URL = %v, want %q", storyRepo.updated[0].AudioURL, s3Client.uploadURL)
	}
	if storyRepo.updated[0].Status != domain.StoryStatusActive {
		t.Fatalf("updated status = %q, want %q", storyRepo.updated[0].Status, domain.StoryStatusActive)
	}
	if len(s3Client.deletedKeys) != 0 {
		t.Fatalf("deleted audio keys = %v, want none", s3Client.deletedKeys)
	}
}

func TestStart_CanceledContext(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{PollInterval: 100 * time.Millisecond},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	err := w.Start(ctx)
	if err != nil {
		t.Errorf("Start() returned error: %v", err)
	}
}

// TestProcessJob_HeartbeatUpdated verifies that heartbeat is sent during long-running job processing.
func TestProcessJob_HeartbeatUpdated(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	// Use a slow Claude client to simulate a long-running job
	slowClaude := &slowMockClaudeClient{
		delay: 250 * time.Millisecond,
		result: &claude.StoryResult{
			Text:       "Story",
			LayerType:  domain.StoryLayerGeneral,
			Confidence: 80,
		},
	}

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{nextID: 300, storyCount: 0},
		&mockPOIRepo{poi: &domain.POI{ID: 1, CityID: 1}},
		slowClaude,
		&mockTTSClient{result: &elevenlabs.AudioResult{
			Audio:    bytes.NewReader([]byte("mp3")),
			Duration: time.Millisecond,
		}},
		&mockS3Client{uploadURL: "https://cdn.example.com/audio.mp3"},
		&Config{
			LeaseWindow: 300 * time.Millisecond, // heartbeat interval = 100ms
			JobTimeout:  5 * time.Second,
			WorkerID:    "test-hb",
		},
	)

	job := &domain.InflationJob{ID: 50, POIID: 1}
	w.processJob(context.Background(), job)

	// With 250ms delay and 100ms heartbeat interval, we expect at least 1 heartbeat
	hbCount := inflationRepo.getHeartbeatCount(50)
	if hbCount < 1 {
		t.Errorf("heartbeat count for job 50 = %d, want >= 1", hbCount)
	}

	// Job should still complete successfully
	if len(inflationRepo.completedJobs) != 1 || inflationRepo.completedJobs[0] != 50 {
		t.Fatalf("completed jobs = %v, want [50]", inflationRepo.completedJobs)
	}
}

// TestProcessJob_HeartbeatStopsOnCompletion verifies heartbeat goroutine stops after job finishes.
func TestProcessJob_HeartbeatStopsOnCompletion(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{nextID: 400, storyCount: 0},
		&mockPOIRepo{poi: &domain.POI{ID: 1, CityID: 1}},
		&mockClaudeClient{result: &claude.StoryResult{
			Text:       "Story",
			LayerType:  domain.StoryLayerGeneral,
			Confidence: 80,
		}},
		&mockTTSClient{result: &elevenlabs.AudioResult{
			Audio:    bytes.NewReader([]byte("mp3")),
			Duration: time.Millisecond,
		}},
		&mockS3Client{uploadURL: "https://cdn.example.com/audio.mp3"},
		&Config{
			LeaseWindow: 150 * time.Millisecond, // heartbeat interval = 50ms
			JobTimeout:  5 * time.Second,
			WorkerID:    "test-hb-stop",
		},
	)

	job := &domain.InflationJob{ID: 60, POIID: 1}
	w.processJob(context.Background(), job)

	// Record heartbeat count right after completion
	countAfterDone := inflationRepo.getHeartbeatCount(60)

	// Wait a few heartbeat intervals and verify no more heartbeats arrive
	time.Sleep(200 * time.Millisecond)

	countLater := inflationRepo.getHeartbeatCount(60)
	if countLater != countAfterDone {
		t.Errorf("heartbeat count grew from %d to %d after job completed", countAfterDone, countLater)
	}
}

// TestPollAndProcess_ClaimsBatchOfJobs verifies the worker claims multiple jobs up to batchSize.
func TestPollAndProcess_ClaimsBatchOfJobs(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	inflationRepo.claimableJobs = []*domain.InflationJob{
		{ID: 1, POIID: 10},
		{ID: 2, POIID: 20},
	}
	inflationRepo.activeCount = 3 // will cause immediate fail, but that's fine — we're testing claim count

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{BatchSize: 5, WorkerID: "test-batch"},
	)

	w.pollAndProcess(context.Background())

	// Should have claimed 2 jobs, then 3rd call returns nil, stopping the loop
	if inflationRepo.claimCalls != 3 {
		t.Errorf("claimCalls = %d, want 3 (2 jobs + 1 nil)", inflationRepo.claimCalls)
	}
}

// TestPollAndProcess_StaleJobRecovery verifies that the worker can reclaim stale running jobs
// through the ClaimNextJob mechanism. This tests the contract: stale running jobs appear
// in the claimable set alongside pending jobs.
func TestPollAndProcess_StaleJobRecovery(t *testing.T) {
	staleStarted := time.Now().Add(-10 * time.Minute)
	staleHeartbeat := time.Now().Add(-10 * time.Minute)
	oldWorker := "crashed-worker-1"

	inflationRepo := newMockInflationRepo()
	// Simulate a stale running job being returned by ClaimNextJob
	inflationRepo.claimableJobs = []*domain.InflationJob{
		{
			ID:          99,
			POIID:       42,
			Status:      domain.InflationJobStatusRunning,
			StartedAt:   &staleStarted,
			HeartbeatAt: &staleHeartbeat,
			WorkerID:    &oldWorker,
		},
	}
	inflationRepo.activeCount = 3 // will fail the job, but we're testing claim behavior

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{
			LeaseWindow: 5 * time.Minute,
			WorkerID:    "recovery-worker",
		},
	)

	w.pollAndProcess(context.Background())

	// The stale job should have been claimed and processed (failed due to max segments)
	if inflationRepo.claimCalls < 1 {
		t.Fatalf("claimCalls = %d, want >= 1", inflationRepo.claimCalls)
	}
	if _, ok := inflationRepo.failedJobs[99]; !ok {
		t.Error("expected stale job 99 to be processed (and failed due to max segments)")
	}
}

// TestProcessJob_HeartbeatStopsOnFailure verifies the heartbeat stops when a job fails.
func TestProcessJob_HeartbeatStopsOnFailure(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{getErr: errors.New("poi gone")},
		nil, nil, nil,
		&Config{
			LeaseWindow: 150 * time.Millisecond,
			WorkerID:    "test-hb-fail",
		},
	)

	job := &domain.InflationJob{ID: 70, POIID: 999}
	w.processJob(context.Background(), job)

	countAfterFail := inflationRepo.getHeartbeatCount(70)

	time.Sleep(200 * time.Millisecond)

	countLater := inflationRepo.getHeartbeatCount(70)
	if countLater != countAfterFail {
		t.Errorf("heartbeat count grew from %d to %d after job failed", countAfterFail, countLater)
	}
}

// TestStart_ReclaimsStaleJobsAtStartup verifies ReclaimStaleJobs is called on startup.
func TestStart_ReclaimsStaleJobsAtStartup(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	inflationRepo.reclaimResult = 2

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{
			PollInterval:    100 * time.Millisecond,
			ReclaimInterval: 10 * time.Second, // long enough to not fire during test
			WorkerID:        "test-reclaim",
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_ = w.Start(ctx)

	calls := inflationRepo.getReclaimCalls()
	if calls < 1 {
		t.Errorf("reclaimCalls = %d, want >= 1 (startup reclaim)", calls)
	}
}

// TestStart_ReclaimsStaleJobsPeriodically verifies ReclaimStaleJobs is called periodically.
func TestStart_ReclaimsStaleJobsPeriodically(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{
			PollInterval:    50 * time.Millisecond,
			ReclaimInterval: 100 * time.Millisecond,
			WorkerID:        "test-reclaim-periodic",
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	_ = w.Start(ctx)

	// 1 at startup + at least 2 from periodic ticks (100ms interval over 350ms)
	calls := inflationRepo.getReclaimCalls()
	if calls < 3 {
		t.Errorf("reclaimCalls = %d, want >= 3 (1 startup + periodic)", calls)
	}
}

// TestReclaimStaleJobs_PassesConfig verifies the worker passes correct threshold and maxAttempts.
func TestReclaimStaleJobs_PassesConfig(t *testing.T) {
	inflationRepo := newMockInflationRepo()

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{
			PollInterval:    100 * time.Millisecond,
			LeaseWindow:     7 * time.Minute,
			MaxAttempts:     5,
			ReclaimInterval: 10 * time.Second,
			WorkerID:        "test-config",
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_ = w.Start(ctx)

	inflationRepo.mu.Lock()
	defer inflationRepo.mu.Unlock()

	if inflationRepo.reclaimThreshold != 7*time.Minute {
		t.Errorf("reclaimThreshold = %v, want 7m", inflationRepo.reclaimThreshold)
	}
	if inflationRepo.reclaimMaxAtt != 5 {
		t.Errorf("reclaimMaxAttempts = %d, want 5", inflationRepo.reclaimMaxAtt)
	}
}

// TestNewInflationWorker_DefaultMaxAttempts verifies defaults for new config fields.
func TestNewInflationWorker_DefaultMaxAttempts(t *testing.T) {
	w := NewInflationWorker(
		newMockInflationRepo(),
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	if w.maxAttempts != defaultMaxAttempts {
		t.Errorf("maxAttempts = %d, want %d", w.maxAttempts, defaultMaxAttempts)
	}
	if w.reclaimInterval != defaultReclaimInterval {
		t.Errorf("reclaimInterval = %v, want %v", w.reclaimInterval, defaultReclaimInterval)
	}
}

// TestReclaimStaleJobs_ErrorHandling verifies reclaim errors are handled gracefully.
func TestReclaimStaleJobs_ErrorHandling(t *testing.T) {
	inflationRepo := newMockInflationRepo()
	inflationRepo.reclaimErr = errors.New("db connection failed")

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{
			PollInterval:    100 * time.Millisecond,
			ReclaimInterval: 10 * time.Second,
			WorkerID:        "test-reclaim-err",
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// Should not panic even when reclaim fails
	err := w.Start(ctx)
	if err != nil {
		t.Errorf("Start() returned error: %v", err)
	}
}

// Verify mock interfaces satisfy the worker interfaces at compile time.
var (
	_ InflationRepo  = (*mockInflationRepo)(nil)
	_ StoryRepo      = (*mockStoryRepo)(nil)
	_ POIRepo        = (*mockPOIRepo)(nil)
	_ StoryGenerator = (*mockClaudeClient)(nil)
	_ AudioGenerator = (*mockTTSClient)(nil)
	_ ObjectStorage  = (*mockS3Client)(nil)
)

// TestInterfaceCompatibility documents the compile-time interface checks.
func TestInterfaceCompatibility(t *testing.T) {
	t.Log("Interface compatibility verified at compile time")
}

// TestAudioKeyFormat verifies the S3 key format helper.
func TestAudioKeyFormat(t *testing.T) {
	key := s3.AudioKey(1, 42, 100)
	expected := "audio/1/42/100.mp3"
	if key != expected {
		t.Errorf("AudioKey(1, 42, 100) = %q, want %q", key, expected)
	}
}

// --- slow mock for heartbeat tests ---

type slowMockClaudeClient struct {
	delay  time.Duration
	result *claude.StoryResult
}

func (m *slowMockClaudeClient) GenerateStory(ctx context.Context, _ *domain.POI, _ string) (*claude.StoryResult, error) {
	select {
	case <-time.After(m.delay):
		return m.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
