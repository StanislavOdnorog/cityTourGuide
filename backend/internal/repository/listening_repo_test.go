//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// createTestUserDirect inserts a test user using the pool directly.
func createTestUserDirect(t *testing.T, tp *repository.TestPool, ctx context.Context) string {
	t.Helper()
	var userID string
	err := tp.Pool.QueryRow(ctx,
		`INSERT INTO users (is_anonymous, auth_provider) VALUES (true, 'email') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	return userID
}

// deleteTestUser removes a test user by ID.
func deleteTestUser(t *testing.T, tp *repository.TestPool, ctx context.Context, userID string) {
	t.Helper()
	_, err := tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		t.Fatalf("deleteTestUser: %v", err)
	}
}

func TestListeningRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	listenRepo := repository.NewListeningRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Listen POI", 41.69, 44.80)
	story := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Listening test story")

	userID := createTestUserDirect(t, tp, ctx)
	defer deleteTestUser(t, tp, ctx, userID)

	// Create listening with location
	lat := 41.6927
	lng := 44.8090
	listening, err := listenRepo.Create(ctx, userID, story.ID, true, &lat, &lng)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if listening.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if listening.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, listening.UserID)
	}
	if listening.StoryID != story.ID {
		t.Errorf("expected story_id %d, got %d", story.ID, listening.StoryID)
	}
	if !listening.Completed {
		t.Error("expected completed to be true")
	}
	if listening.Lat == nil || *listening.Lat < 41.69 || *listening.Lat > 41.70 {
		t.Errorf("expected lat ~41.6927, got %v", listening.Lat)
	}
	if listening.Lng == nil || *listening.Lng < 44.80 || *listening.Lng > 44.81 {
		t.Errorf("expected lng ~44.8090, got %v", listening.Lng)
	}
	if listening.ListenedAt.IsZero() {
		t.Error("expected non-zero listened_at")
	}
}

func TestListeningRepo_Create_WithoutLocation(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	listenRepo := repository.NewListeningRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "NoLoc POI", 41.69, 44.80)
	story := createTestStory(t, storyRepo, ctx, poi.ID, "en", "No location story")

	userID := createTestUserDirect(t, tp, ctx)
	defer deleteTestUser(t, tp, ctx, userID)

	// Create listening without location
	listening, err := listenRepo.Create(ctx, userID, story.ID, false, nil, nil)
	if err != nil {
		t.Fatalf("Create without location failed: %v", err)
	}

	if listening.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if listening.Lat != nil {
		t.Errorf("expected nil lat, got %v", listening.Lat)
	}
	if listening.Lng != nil {
		t.Errorf("expected nil lng, got %v", listening.Lng)
	}
	if listening.Completed {
		t.Error("expected completed to be false")
	}
}

func TestListeningRepo_GetListenedStoryIDs(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	listenRepo := repository.NewListeningRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "IDs POI", 41.69, 44.80)
	story1 := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Story for IDs 1")
	story2 := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Story for IDs 2")
	story3 := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Story for IDs 3")

	userID := createTestUserDirect(t, tp, ctx)
	defer deleteTestUser(t, tp, ctx, userID)

	// Listen to story1 and story2 (not story3)
	_, err := listenRepo.Create(ctx, userID, story1.ID, true, nil, nil)
	if err != nil {
		t.Fatalf("Create listening 1 failed: %v", err)
	}
	_, err = listenRepo.Create(ctx, userID, story2.ID, false, nil, nil)
	if err != nil {
		t.Fatalf("Create listening 2 failed: %v", err)
	}

	ids, err := listenRepo.GetListenedStoryIDs(ctx, userID)
	if err != nil {
		t.Fatalf("GetListenedStoryIDs failed: %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("expected 2 listened story IDs, got %d", len(ids))
	}

	// Check that story3 is NOT in the list
	idSet := make(map[int]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet[story1.ID] {
		t.Errorf("expected story1 (ID=%d) in listened IDs", story1.ID)
	}
	if !idSet[story2.ID] {
		t.Errorf("expected story2 (ID=%d) in listened IDs", story2.ID)
	}
	if idSet[story3.ID] {
		t.Errorf("story3 (ID=%d) should NOT be in listened IDs", story3.ID)
	}
}

func TestListeningRepo_GetListenedStoryIDs_Empty(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	listenRepo := repository.NewListeningRepo(tp.Pool)
	ctx := context.Background()

	userID := createTestUserDirect(t, tp, ctx)
	defer deleteTestUser(t, tp, ctx, userID)

	ids, err := listenRepo.GetListenedStoryIDs(ctx, userID)
	if err != nil {
		t.Fatalf("GetListenedStoryIDs failed: %v", err)
	}

	if ids != nil {
		t.Errorf("expected nil (no listened stories), got %v", ids)
	}
}

func TestListeningRepo_GetListenedStoryIDs_Deduplicated(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	listenRepo := repository.NewListeningRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Dedup POI", 41.69, 44.80)
	story := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Dedup story")

	userID := createTestUserDirect(t, tp, ctx)
	defer deleteTestUser(t, tp, ctx, userID)

	// Listen to the same story twice
	_, _ = listenRepo.Create(ctx, userID, story.ID, false, nil, nil)
	_, _ = listenRepo.Create(ctx, userID, story.ID, true, nil, nil)

	ids, err := listenRepo.GetListenedStoryIDs(ctx, userID)
	if err != nil {
		t.Fatalf("GetListenedStoryIDs failed: %v", err)
	}

	// Should return 1 unique ID despite 2 listening records
	if len(ids) != 1 {
		t.Errorf("expected 1 deduplicated story ID, got %d", len(ids))
	}
}

func TestListeningRepo_HasListened(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	listenRepo := repository.NewListeningRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "HasListened POI", 41.69, 44.80)
	listened := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Listened story")
	notListened := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Not listened story")

	userID := createTestUserDirect(t, tp, ctx)
	defer deleteTestUser(t, tp, ctx, userID)

	// Listen to one story
	_, err := listenRepo.Create(ctx, userID, listened.ID, true, nil, nil)
	if err != nil {
		t.Fatalf("Create listening failed: %v", err)
	}

	// HasListened for listened story — should be true
	has, err := listenRepo.HasListened(ctx, userID, listened.ID)
	if err != nil {
		t.Fatalf("HasListened (listened) failed: %v", err)
	}
	if !has {
		t.Error("expected HasListened to be true for listened story")
	}

	// HasListened for not-listened story — should be false
	has, err = listenRepo.HasListened(ctx, userID, notListened.ID)
	if err != nil {
		t.Fatalf("HasListened (not listened) failed: %v", err)
	}
	if has {
		t.Error("expected HasListened to be false for not-listened story")
	}
}
