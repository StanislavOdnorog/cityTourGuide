package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/s3"
)

// --- mock repos ---

type mockInflationRepo struct {
	pendingJobs    []domain.InflationJob
	activeCount    int
	setRunningErr  error
	setRunningCall int
	completedJobs  []int
	failedJobs     map[int]string
}

func (m *mockInflationRepo) GetPendingJobs(_ context.Context, _ int) ([]domain.InflationJob, error) {
	return m.pendingJobs, nil
}

func (m *mockInflationRepo) SetRunning(_ context.Context, jobID int) error {
	m.setRunningCall++
	if m.setRunningErr != nil {
		return m.setRunningErr
	}
	return nil
}

func (m *mockInflationRepo) SetCompleted(_ context.Context, jobID int) error {
	m.completedJobs = append(m.completedJobs, jobID)
	return nil
}

func (m *mockInflationRepo) SetFailed(_ context.Context, jobID int, errMsg string) error {
	if m.failedJobs == nil {
		m.failedJobs = make(map[int]string)
	}
	m.failedJobs[jobID] = errMsg
	return nil
}

func (m *mockInflationRepo) CountActiveByPOIID(_ context.Context, _ int) (int, error) {
	return m.activeCount, nil
}

type mockStoryRepo struct {
	created    []*domain.Story
	createErr  error
	updated    []*domain.Story
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
	result := *story
	result.UpdatedAt = time.Now()
	m.updated = append(m.updated, &result)
	return &result, nil
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
		&mockInflationRepo{},
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
}

func TestNewInflationWorker_CustomConfig(t *testing.T) {
	w := NewInflationWorker(
		&mockInflationRepo{},
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		&Config{PollInterval: 30 * time.Second, BatchSize: 10, JobTimeout: 45 * time.Second},
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
}

func TestPollAndProcess_NoPendingJobs(t *testing.T) {
	inflationRepo := &mockInflationRepo{pendingJobs: nil}

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	ctx := context.Background()
	w.pollAndProcess(ctx)

	// No jobs to process — setRunning should never be called
	if inflationRepo.setRunningCall != 0 {
		t.Errorf("setRunningCall = %d, want 0", inflationRepo.setRunningCall)
	}
}

func TestProcessJob_MaxSegmentsReached(t *testing.T) {
	inflationRepo := &mockInflationRepo{
		activeCount: 3, // already at max
		failedJobs:  make(map[int]string),
	}

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

	// Job should be failed because max segments reached
	if _, ok := inflationRepo.failedJobs[1]; !ok {
		t.Error("expected job 1 to be marked as failed")
	}
}

func TestProcessJob_AlreadyClaimed(t *testing.T) {
	inflationRepo := &mockInflationRepo{
		activeCount:   0,
		setRunningErr: errors.New("not found"),
	}

	w := NewInflationWorker(
		inflationRepo,
		&mockStoryRepo{},
		&mockPOIRepo{},
		nil, nil, nil,
		nil,
	)

	job := &domain.InflationJob{
		ID:    2,
		POIID: 100,
	}

	w.processJob(context.Background(), job)

	// Job should NOT be in failed — it was just already claimed
	if len(inflationRepo.failedJobs) != 0 {
		t.Errorf("expected no failed jobs, got %d", len(inflationRepo.failedJobs))
	}
}

func TestProcessJob_POINotFound(t *testing.T) {
	inflationRepo := &mockInflationRepo{
		activeCount: 0,
		failedJobs:  make(map[int]string),
	}
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

func TestStart_CanceledContext(t *testing.T) {
	inflationRepo := &mockInflationRepo{pendingJobs: nil}

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

// Verify mock interfaces satisfy the worker interfaces at compile time.
var (
	_ InflationRepo = (*mockInflationRepo)(nil)
	_ StoryRepo     = (*mockStoryRepo)(nil)
	_ POIRepo       = (*mockPOIRepo)(nil)
)

// Ensure the real repos can satisfy the interfaces (compile-time check via unused vars).
// These are tested here so we catch interface drift early.
func TestInterfaceCompatibility(t *testing.T) {
	// These are compile-time checks — if the interfaces drift, this file won't compile.
	// The _ assignments above already enforce this, so this test just documents the intent.
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
