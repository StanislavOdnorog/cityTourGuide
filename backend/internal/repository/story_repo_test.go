//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

func createTestStory(t *testing.T, storyRepo *repository.StoryRepo, ctx context.Context, poiID int, language string, text string) *domain.Story {
	t.Helper()
	story, err := storyRepo.Create(ctx, &domain.Story{
		POIID:      poiID,
		Language:   language,
		Text:       text,
		LayerType:  domain.StoryLayerGeneral,
		Confidence: 80,
		Sources:    json.RawMessage(`[]`),
		Status:     domain.StoryStatusActive,
	})
	if err != nil {
		t.Fatalf("createTestStory(%s): %v", text, err)
	}
	return story
}

func TestStoryRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Story Test POI", 41.69, 44.80)

	audioURL := "https://cdn.example.com/audio/test.mp3"
	durationSec := int16(30)
	story := &domain.Story{
		POIID:       poi.ID,
		Language:    "en",
		Text:        "This is a fascinating story about the old city.",
		AudioURL:    &audioURL,
		DurationSec: &durationSec,
		LayerType:   domain.StoryLayerHumanStory,
		OrderIndex:  0,
		IsInflation: false,
		Confidence:  85,
		Sources:     json.RawMessage(`[{"url": "https://example.com"}]`),
		Status:      domain.StoryStatusActive,
	}

	created, err := storyRepo.Create(ctx, story)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.POIID != poi.ID {
		t.Errorf("expected poi_id %d, got %d", poi.ID, created.POIID)
	}
	if created.Language != "en" {
		t.Errorf("expected language 'en', got %s", created.Language)
	}
	if created.Text != "This is a fascinating story about the old city." {
		t.Errorf("unexpected text: %s", created.Text)
	}
	if created.AudioURL == nil || *created.AudioURL != audioURL {
		t.Error("expected audio_url to match")
	}
	if created.DurationSec == nil || *created.DurationSec != 30 {
		t.Error("expected duration_sec 30")
	}
	if created.LayerType != domain.StoryLayerHumanStory {
		t.Errorf("expected layer_type 'human_story', got %s", created.LayerType)
	}
	if created.Confidence != 85 {
		t.Errorf("expected confidence 85, got %d", created.Confidence)
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestStoryRepo_GetByID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "GetByID POI", 41.69, 44.80)
	created := createTestStory(t, storyRepo, ctx, poi.ID, "en", "A test story for GetByID")

	got, err := storyRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Text != "A test story for GetByID" {
		t.Errorf("expected text 'A test story for GetByID', got %s", got.Text)
	}
	if got.POIID != poi.ID {
		t.Errorf("expected poi_id %d, got %d", poi.ID, got.POIID)
	}
}

func TestStoryRepo_GetByID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	_, err := storyRepo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStoryRepo_GetByPOIID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "GetByPOIID POI", 41.69, 44.80)

	// Create 3 stories: 2 EN, 1 RU
	createTestStory(t, storyRepo, ctx, poi.ID, "en", "English story 1")
	createTestStory(t, storyRepo, ctx, poi.ID, "en", "English story 2")
	createTestStory(t, storyRepo, ctx, poi.ID, "ru", "Русская история")

	// Get EN stories without status filter
	enStories, err := storyRepo.GetByPOIID(ctx, poi.ID, "en", nil)
	if err != nil {
		t.Fatalf("GetByPOIID EN failed: %v", err)
	}
	if len(enStories) != 2 {
		t.Errorf("expected 2 EN stories, got %d", len(enStories))
	}

	// Get RU stories
	ruStories, err := storyRepo.GetByPOIID(ctx, poi.ID, "ru", nil)
	if err != nil {
		t.Fatalf("GetByPOIID RU failed: %v", err)
	}
	if len(ruStories) != 1 {
		t.Errorf("expected 1 RU story, got %d", len(ruStories))
	}

	// Get EN stories with active status filter
	activeStatus := domain.StoryStatusActive
	activeStories, err := storyRepo.GetByPOIID(ctx, poi.ID, "en", &activeStatus)
	if err != nil {
		t.Fatalf("GetByPOIID EN active failed: %v", err)
	}
	if len(activeStories) != 2 {
		t.Errorf("expected 2 active EN stories, got %d", len(activeStories))
	}

	// Get EN stories with disabled status filter — should be 0
	disabledStatus := domain.StoryStatusDisabled
	disabledStories, err := storyRepo.GetByPOIID(ctx, poi.ID, "en", &disabledStatus)
	if err != nil {
		t.Fatalf("GetByPOIID EN disabled failed: %v", err)
	}
	if len(disabledStories) != 0 {
		t.Errorf("expected 0 disabled EN stories, got %d", len(disabledStories))
	}
}

func TestStoryRepo_Update(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Update POI", 41.69, 44.80)
	created := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Original text")

	// Update text and status
	audioURL := "https://cdn.example.com/updated.mp3"
	created.Text = "Updated text with more details"
	created.AudioURL = &audioURL
	created.Status = domain.StoryStatusDisabled
	created.Confidence = 95

	updated, err := storyRepo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Text != "Updated text with more details" {
		t.Errorf("expected updated text, got %s", updated.Text)
	}
	if updated.AudioURL == nil || *updated.AudioURL != audioURL {
		t.Error("expected updated audio_url")
	}
	if updated.Status != domain.StoryStatusDisabled {
		t.Errorf("expected status 'disabled', got %s", updated.Status)
	}
	if updated.Confidence != 95 {
		t.Errorf("expected confidence 95, got %d", updated.Confidence)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) {
		t.Error("expected updated_at to be after created_at")
	}
}

func TestStoryRepo_Update_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Ghost POI", 41.69, 44.80)

	_, err := storyRepo.Update(ctx, &domain.Story{
		ID: 999999, POIID: poi.ID, Language: "en", Text: "Ghost story",
		LayerType: domain.StoryLayerGeneral, Confidence: 80,
		Sources: json.RawMessage(`[]`), Status: domain.StoryStatusActive,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStoryRepo_Delete(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Delete POI", 41.69, 44.80)
	created := createTestStory(t, storyRepo, ctx, poi.ID, "en", "To be deleted")

	err := storyRepo.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = storyRepo.GetByID(ctx, created.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStoryRepo_Delete_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	err := storyRepo.Delete(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStoryRepo_CountByPOI(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Count POI", 41.69, 44.80)

	// Initially 0
	count, err := storyRepo.CountByPOI(ctx, poi.ID)
	if err != nil {
		t.Fatalf("CountByPOI failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 stories, got %d", count)
	}

	// Create 3 stories
	createTestStory(t, storyRepo, ctx, poi.ID, "en", "Story 1")
	createTestStory(t, storyRepo, ctx, poi.ID, "en", "Story 2")
	createTestStory(t, storyRepo, ctx, poi.ID, "ru", "Story 3")

	count, err = storyRepo.CountByPOI(ctx, poi.ID)
	if err != nil {
		t.Fatalf("CountByPOI after inserts failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 stories, got %d", count)
	}
}

func TestStoryRepo_FullCRUDCycle(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "CRUD POI", 41.69, 44.80)

	// Create
	story, err := storyRepo.Create(ctx, &domain.Story{
		POIID: poi.ID, Language: "en", Text: "CRUD test story",
		LayerType: domain.StoryLayerAtmosphere, Confidence: 80,
		Sources: json.RawMessage(`[]`), Status: domain.StoryStatusActive,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Read
	got, err := storyRepo.GetByID(ctx, story.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Text != "CRUD test story" {
		t.Errorf("expected 'CRUD test story', got %s", got.Text)
	}

	// Update
	got.Text = "Updated CRUD test story"
	got.LayerType = domain.StoryLayerHiddenDetail
	updated, err := storyRepo.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Text != "Updated CRUD test story" {
		t.Errorf("expected updated text, got %s", updated.Text)
	}
	if updated.LayerType != domain.StoryLayerHiddenDetail {
		t.Errorf("expected layer_type 'hidden_detail', got %s", updated.LayerType)
	}

	// Delete
	err = storyRepo.Delete(ctx, updated.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = storyRepo.GetByID(ctx, updated.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
