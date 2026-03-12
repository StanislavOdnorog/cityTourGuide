//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// seedReportDeps creates the minimum related data (city → POI → story + user)
// needed for report tests and returns cleanup functions.
func seedReportDeps(t *testing.T, tp *repository.TestPool) (storyID int, userID string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)

	city := createTestCity(t, cityRepo, ctx)
	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Report Test POI", 41.69, 44.80)
	story := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Report test story")

	userID = "550e8400-e29b-41d4-a716-446655440099"
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return story.ID, userID, func() {
		_ = cityRepo.Delete(ctx, city.ID)
		_, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	}
}

func TestReportRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	comment := "Inaccurate date mentioned"
	lat, lng := 41.69, 44.80

	report, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, &comment, &lat, &lng)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if report.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if report.StoryID != storyID {
		t.Errorf("expected story_id %d, got %d", storyID, report.StoryID)
	}
	if report.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, report.UserID)
	}
	if report.Type != domain.ReportTypeWrongFact {
		t.Errorf("expected type wrong_fact, got %s", report.Type)
	}
	if report.Comment == nil || *report.Comment != comment {
		t.Error("expected comment to match")
	}
	if report.UserLat == nil || *report.UserLat != lat {
		t.Error("expected user_lat to match")
	}
	if report.Status != domain.ReportStatusNew {
		t.Errorf("expected status new, got %s", report.Status)
	}
	if report.ResolvedAt != nil {
		t.Error("expected resolved_at to be nil")
	}
	if report.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestReportRepo_Create_NilOptionalFields(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	report, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeInappropriateContent, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if report.Comment != nil {
		t.Error("expected nil comment")
	}
	if report.UserLat != nil {
		t.Error("expected nil user_lat")
	}
	if report.UserLng != nil {
		t.Error("expected nil user_lng")
	}
}

func TestReportRepo_GetByID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	created, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongLocation, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := reportRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
	if got.Type != domain.ReportTypeWrongLocation {
		t.Errorf("expected type wrong_location, got %s", got.Type)
	}
	if got.StoryID != storyID {
		t.Errorf("expected story_id %d, got %d", storyID, got.StoryID)
	}
}

func TestReportRepo_GetByID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	reportRepo := repository.NewReportRepo(tp.Pool)
	ctx := context.Background()

	_, err := reportRepo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReportRepo_UpdateStatus_Resolved(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	created, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := reportRepo.UpdateStatus(ctx, created.ID, domain.ReportStatusResolved)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	if updated.Status != domain.ReportStatusResolved {
		t.Errorf("expected status resolved, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set for resolved status")
	}
}

func TestReportRepo_UpdateStatus_Dismissed(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	created, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := reportRepo.UpdateStatus(ctx, created.ID, domain.ReportStatusDismissed)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	if updated.Status != domain.ReportStatusDismissed {
		t.Errorf("expected status dismissed, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set for dismissed status")
	}
}

func TestReportRepo_UpdateStatus_Reviewed_NoResolvedAt(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	created, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := reportRepo.UpdateStatus(ctx, created.ID, domain.ReportStatusReviewed)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	if updated.Status != domain.ReportStatusReviewed {
		t.Errorf("expected status reviewed, got %s", updated.Status)
	}
	if updated.ResolvedAt != nil {
		t.Error("expected resolved_at to be nil for reviewed status")
	}
}

func TestReportRepo_UpdateStatus_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	reportRepo := repository.NewReportRepo(tp.Pool)
	ctx := context.Background()

	_, err := reportRepo.UpdateStatus(ctx, 999999, domain.ReportStatusResolved)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReportRepo_ListAdmin_SortsByCreatedAtDesc(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	first, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("create first report: %v", err)
	}
	second, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongLocation, nil, nil, nil)
	if err != nil {
		t.Fatalf("create second report: %v", err)
	}

	result, err := reportRepo.ListAdmin(ctx, "", domain.PageRequest{Limit: 10}, repository.ListSort{
		By:  "created_at",
		Dir: repository.SortDirDesc,
	})
	if err != nil {
		t.Fatalf("ListAdmin failed: %v", err)
	}

	if len(result.Items) < 2 {
		t.Fatalf("expected at least 2 reports, got %d", len(result.Items))
	}
	if result.Items[0].ID != second.ID || result.Items[1].ID != first.ID {
		t.Fatalf("expected latest report first, got ids %d then %d", result.Items[0].ID, result.Items[1].ID)
	}
}

func TestReportRepo_List_CursorPagination(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	// Create 3 reports
	var createdIDs []int
	for i := 0; i < 3; i++ {
		r, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
		if err != nil {
			t.Fatalf("Create report %d: %v", i, err)
		}
		createdIDs = append(createdIDs, r.ID)
	}

	// First page: limit 2
	firstPage, err := reportRepo.List(ctx, "", domain.PageRequest{Limit: 2})
	if err != nil {
		t.Fatalf("List first page: %v", err)
	}

	if len(firstPage.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(firstPage.Items))
	}
	if !firstPage.HasMore {
		t.Fatal("expected first page to have more results")
	}
	if firstPage.NextCursor == "" {
		t.Fatal("expected next cursor on first page")
	}

	// Verify ascending ID order
	if firstPage.Items[0].ID >= firstPage.Items[1].ID {
		t.Errorf("expected ascending ID order, got %d >= %d", firstPage.Items[0].ID, firstPage.Items[1].ID)
	}

	// Second page using cursor
	secondPage, err := reportRepo.List(ctx, "", domain.PageRequest{
		Cursor: firstPage.NextCursor,
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("List second page: %v", err)
	}

	if len(secondPage.Items) < 1 {
		t.Fatal("expected at least 1 item on second page")
	}

	// No overlap between pages
	firstIDs := map[int]bool{}
	for _, item := range firstPage.Items {
		firstIDs[item.ID] = true
	}
	for _, item := range secondPage.Items {
		if firstIDs[item.ID] {
			t.Errorf("duplicate ID %d across pages", item.ID)
		}
	}

	// Second page items must have higher IDs than first page
	lastFirstID := firstPage.Items[len(firstPage.Items)-1].ID
	for _, item := range secondPage.Items {
		if item.ID <= lastFirstID {
			t.Errorf("second page item ID %d <= last first page ID %d", item.ID, lastFirstID)
		}
	}
}

func TestReportRepo_List_StatusFilter(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	// Create reports with different statuses
	r1, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create r1: %v", err)
	}
	r2, err := reportRepo.Create(ctx, storyID, userID, domain.ReportTypeWrongLocation, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create r2: %v", err)
	}

	// Resolve r1
	if _, err := reportRepo.UpdateStatus(ctx, r1.ID, domain.ReportStatusResolved); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// List only "new" reports — should include r2 but not r1
	result, err := reportRepo.List(ctx, string(domain.ReportStatusNew), domain.PageRequest{Limit: 50})
	if err != nil {
		t.Fatalf("List with status filter: %v", err)
	}

	ids := map[int]bool{}
	for _, item := range result.Items {
		ids[item.ID] = true
		if item.Status != domain.ReportStatusNew {
			t.Errorf("expected all items to have status new, got %s", item.Status)
		}
	}

	if ids[r1.ID] {
		t.Errorf("resolved report %d should not appear in 'new' filter", r1.ID)
	}
	if !ids[r2.ID] {
		t.Errorf("new report %d should appear in 'new' filter", r2.ID)
	}
}

func TestReportRepo_GetByPOIID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi1 := createTestPOI(t, poiRepo, ctx, city.ID, "POI With Reports", 41.69, 44.80)
	poi2 := createTestPOI(t, poiRepo, ctx, city.ID, "POI Without Reports", 41.70, 44.81)
	story1 := createTestStory(t, storyRepo, ctx, poi1.ID, "en", "Story for POI1")
	_ = createTestStory(t, storyRepo, ctx, poi2.ID, "en", "Story for POI2")

	userID := "550e8400-e29b-41d4-a716-446655440098"
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	defer func() { _, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID) }()

	// Create 2 reports for POI1's story
	r1, err := reportRepo.Create(ctx, story1.ID, userID, domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create r1: %v", err)
	}
	r2, err := reportRepo.Create(ctx, story1.ID, userID, domain.ReportTypeWrongLocation, nil, nil, nil)
	if err != nil {
		t.Fatalf("Create r2: %v", err)
	}

	// Get reports for POI1
	reports, err := reportRepo.GetByPOIID(ctx, poi1.ID)
	if err != nil {
		t.Fatalf("GetByPOIID failed: %v", err)
	}

	if len(reports) != 2 {
		t.Fatalf("expected 2 reports for POI1, got %d", len(reports))
	}

	ids := map[int]bool{r1.ID: false, r2.ID: false}
	for _, r := range reports {
		ids[r.ID] = true
	}
	for id, found := range ids {
		if !found {
			t.Errorf("expected report %d in results", id)
		}
	}

	// Get reports for POI2 — should be 0
	reports2, err := reportRepo.GetByPOIID(ctx, poi2.ID)
	if err != nil {
		t.Fatalf("GetByPOIID for POI2 failed: %v", err)
	}
	if len(reports2) != 0 {
		t.Errorf("expected 0 reports for POI2, got %d", len(reports2))
	}
}
