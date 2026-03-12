package service

import (
	"context"
	"errors"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

type mockAudioCleanupStoryRepo struct {
	getByIDFn func(ctx context.Context, id int) (*domain.Story, error)
	updateFn  func(ctx context.Context, story *domain.Story) (*domain.Story, error)
	deleteFn  func(ctx context.Context, id int) error
}

func (m *mockAudioCleanupStoryRepo) GetByID(ctx context.Context, id int) (*domain.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *mockAudioCleanupStoryRepo) Update(ctx context.Context, story *domain.Story) (*domain.Story, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, story)
	}
	return story, nil
}

func (m *mockAudioCleanupStoryRepo) Delete(ctx context.Context, id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockAudioCleanupPOIRepo struct {
	deleteFn func(ctx context.Context, id int) error
}

func (m *mockAudioCleanupPOIRepo) Delete(ctx context.Context, id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockAudioCleanupCityRepo struct {
	deleteFn func(ctx context.Context, id int) error
}

func (m *mockAudioCleanupCityRepo) Delete(ctx context.Context, id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockAudioURLCollector struct {
	storyURLsFn func(ctx context.Context, storyID int) ([]string, error)
	poiURLsFn   func(ctx context.Context, poiID int) ([]string, error)
	cityURLsFn  func(ctx context.Context, cityID int) ([]string, error)
}

func (m *mockAudioURLCollector) AudioURLsByStoryID(ctx context.Context, storyID int) ([]string, error) {
	if m.storyURLsFn != nil {
		return m.storyURLsFn(ctx, storyID)
	}
	return nil, nil
}

func (m *mockAudioURLCollector) AudioURLsByPOIID(ctx context.Context, poiID int) ([]string, error) {
	if m.poiURLsFn != nil {
		return m.poiURLsFn(ctx, poiID)
	}
	return nil, nil
}

func (m *mockAudioURLCollector) AudioURLsByCityID(ctx context.Context, cityID int) ([]string, error) {
	if m.cityURLsFn != nil {
		return m.cityURLsFn(ctx, cityID)
	}
	return nil, nil
}

type mockCleanupEnqueuer struct {
	keys [][]string
	err  error
}

func (m *mockCleanupEnqueuer) Enqueue(_ context.Context, objectKeys []string) error {
	cloned := append([]string(nil), objectKeys...)
	m.keys = append(m.keys, cloned)
	return m.err
}

func TestAudioCleanupServiceUpdateStorySchedulesOldAudioAfterSuccessfulUpdate(t *testing.T) {
	const oldURL = "https://cdn.example.com/city-stories/audio/1/2/3-old.mp3"
	const newURL = "https://cdn.example.com/city-stories/audio/1/2/3-new.mp3"

	storyRepo := &mockAudioCleanupStoryRepo{
		getByIDFn: func(context.Context, int) (*domain.Story, error) {
			return &domain.Story{ID: 10, AudioURL: ptr(oldURL)}, nil
		},
		updateFn: func(_ context.Context, story *domain.Story) (*domain.Story, error) {
			updated := *story
			return &updated, nil
		},
	}
	enqueuer := &mockCleanupEnqueuer{}
	svc := NewAudioCleanupService(nil, storyRepo, &mockAudioCleanupPOIRepo{}, &mockAudioCleanupCityRepo{}, &mockAudioURLCollector{}, enqueuer)

	updated, err := svc.UpdateStory(context.Background(), &domain.Story{ID: 10, AudioURL: ptr(newURL)})
	if err != nil {
		t.Fatalf("UpdateStory returned error: %v", err)
	}
	if updated == nil || updated.AudioURL == nil || *updated.AudioURL != newURL {
		t.Fatalf("updated story audio = %v, want %q", updated.AudioURL, newURL)
	}
	if len(enqueuer.keys) != 1 {
		t.Fatalf("enqueue calls = %d, want 1", len(enqueuer.keys))
	}
	want := AudioURLToObjectKey(oldURL)
	if len(enqueuer.keys[0]) != 1 || enqueuer.keys[0][0] != want {
		t.Fatalf("enqueued keys = %v, want [%q]", enqueuer.keys[0], want)
	}
}

func TestAudioCleanupServiceUpdateStoryDoesNotScheduleCleanupWhenUpdateFails(t *testing.T) {
	const oldURL = "https://cdn.example.com/city-stories/audio/1/2/3-old.mp3"

	storyRepo := &mockAudioCleanupStoryRepo{
		getByIDFn: func(context.Context, int) (*domain.Story, error) {
			return &domain.Story{ID: 10, AudioURL: ptr(oldURL)}, nil
		},
		updateFn: func(context.Context, *domain.Story) (*domain.Story, error) {
			return nil, repository.ErrConflict
		},
	}
	enqueuer := &mockCleanupEnqueuer{}
	svc := NewAudioCleanupService(nil, storyRepo, &mockAudioCleanupPOIRepo{}, &mockAudioCleanupCityRepo{}, &mockAudioURLCollector{}, enqueuer)

	_, err := svc.UpdateStory(context.Background(), &domain.Story{ID: 10, AudioURL: ptr("https://cdn.example.com/city-stories/audio/1/2/3-new.mp3")})
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("UpdateStory error = %v, want %v", err, repository.ErrConflict)
	}
	if len(enqueuer.keys) != 0 {
		t.Fatalf("enqueue calls = %v, want none", enqueuer.keys)
	}
}

func TestAudioCleanupServiceDeleteCitySchedulesOwnedAudio(t *testing.T) {
	cityDeleted := false
	collector := &mockAudioURLCollector{
		cityURLsFn: func(context.Context, int) ([]string, error) {
			return []string{
				"https://cdn.example.com/city-stories/audio/1/2/10.mp3",
				"https://cdn.example.com/city-stories/audio/1/3/11.mp3",
			}, nil
		},
	}
	cityRepo := &mockAudioCleanupCityRepo{
		deleteFn: func(context.Context, int) error {
			cityDeleted = true
			return nil
		},
	}
	enqueuer := &mockCleanupEnqueuer{}
	svc := NewAudioCleanupService(nil, &mockAudioCleanupStoryRepo{}, &mockAudioCleanupPOIRepo{}, cityRepo, collector, enqueuer)

	if err := svc.DeleteCity(context.Background(), 1); err != nil {
		t.Fatalf("DeleteCity returned error: %v", err)
	}
	if !cityDeleted {
		t.Fatal("DeleteCity did not call city repo delete")
	}
	if len(enqueuer.keys) != 1 {
		t.Fatalf("enqueue calls = %d, want 1", len(enqueuer.keys))
	}
	want := []string{"audio/1/2/10.mp3", "audio/1/3/11.mp3"}
	if len(enqueuer.keys[0]) != len(want) {
		t.Fatalf("enqueued keys = %v, want %v", enqueuer.keys[0], want)
	}
	for i := range want {
		if enqueuer.keys[0][i] != want[i] {
			t.Fatalf("enqueued keys = %v, want %v", enqueuer.keys[0], want)
		}
	}
}

func TestAudioCleanupServiceDeletePOISchedulesOwnedAudio(t *testing.T) {
	poiDeleted := false
	collector := &mockAudioURLCollector{
		poiURLsFn: func(context.Context, int) ([]string, error) {
			return []string{"https://cdn.example.com/city-stories/audio/1/2/10.mp3"}, nil
		},
	}
	poiRepo := &mockAudioCleanupPOIRepo{
		deleteFn: func(context.Context, int) error {
			poiDeleted = true
			return nil
		},
	}
	enqueuer := &mockCleanupEnqueuer{}
	svc := NewAudioCleanupService(nil, &mockAudioCleanupStoryRepo{}, poiRepo, &mockAudioCleanupCityRepo{}, collector, enqueuer)

	if err := svc.DeletePOI(context.Background(), 2); err != nil {
		t.Fatalf("DeletePOI returned error: %v", err)
	}
	if !poiDeleted {
		t.Fatal("DeletePOI did not call POI repo delete")
	}
	if len(enqueuer.keys) != 1 || len(enqueuer.keys[0]) != 1 || enqueuer.keys[0][0] != "audio/1/2/10.mp3" {
		t.Fatalf("enqueued keys = %v, want [[audio/1/2/10.mp3]]", enqueuer.keys)
	}
}

func TestAudioCleanupServiceDeleteStoryReturnsNilWhenEnqueueFails(t *testing.T) {
	storyDeleted := false
	storyRepo := &mockAudioCleanupStoryRepo{
		deleteFn: func(context.Context, int) error {
			storyDeleted = true
			return nil
		},
	}
	collector := &mockAudioURLCollector{
		storyURLsFn: func(context.Context, int) ([]string, error) {
			return []string{"https://cdn.example.com/city-stories/audio/1/2/10.mp3"}, nil
		},
	}
	enqueuer := &mockCleanupEnqueuer{err: errors.New("queue unavailable")}
	svc := NewAudioCleanupService(nil, storyRepo, &mockAudioCleanupPOIRepo{}, &mockAudioCleanupCityRepo{}, collector, enqueuer)

	if err := svc.DeleteStory(context.Background(), 10); err != nil {
		t.Fatalf("DeleteStory returned error: %v", err)
	}
	if !storyDeleted {
		t.Fatal("DeleteStory did not delete the story")
	}
	if len(enqueuer.keys) != 1 {
		t.Fatalf("enqueue calls = %d, want 1", len(enqueuer.keys))
	}
}

func ptr[T any](v T) *T {
	return &v
}
