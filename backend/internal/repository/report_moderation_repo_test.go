//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

func createTestReport(t *testing.T, reportRepo *repository.ReportRepo, ctx context.Context, storyID int) *domain.Report {
	t.Helper()

	report, err := reportRepo.Create(ctx, storyID, "550e8400-e29b-41d4-a716-446655440000", domain.ReportTypeWrongFact, nil, nil, nil)
	if err != nil {
		t.Fatalf("createTestReport: %v", err)
	}
	return report
}

func TestReportRepo_ModerateDisableStory(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	reportRepo := repository.NewReportRepo(tp.Pool)

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Moderation POI", 41.69, 44.80)
	story := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Moderation story")
	report := createTestReport(t, reportRepo, ctx, story.ID)

	result, err := reportRepo.ModerateDisableStory(ctx, report.ID)
	if err != nil {
		t.Fatalf("ModerateDisableStory failed: %v", err)
	}

	if result.Report.Status != domain.ReportStatusResolved {
		t.Fatalf("report status = %q, want %q", result.Report.Status, domain.ReportStatusResolved)
	}
	if result.Report.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}
	if result.Story.Status != domain.StoryStatusDisabled {
		t.Fatalf("story status = %q, want %q", result.Story.Status, domain.StoryStatusDisabled)
	}

	storedStory, err := storyRepo.GetByID(ctx, story.ID)
	if err != nil {
		t.Fatalf("GetByID story failed: %v", err)
	}
	if storedStory.Status != domain.StoryStatusDisabled {
		t.Fatalf("stored story status = %q, want %q", storedStory.Status, domain.StoryStatusDisabled)
	}

	storedReport, err := reportRepo.GetByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetByID report failed: %v", err)
	}
	if storedReport.Status != domain.ReportStatusResolved {
		t.Fatalf("stored report status = %q, want %q", storedReport.Status, domain.ReportStatusResolved)
	}
}

func TestReportRepo_ModerateDisableStory_IdempotentForClosedReport(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	reportRepo := repository.NewReportRepo(tp.Pool)

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Closed Moderation POI", 41.69, 44.80)
	story := createTestStory(t, storyRepo, ctx, poi.ID, "en", "Already disabled story")
	story.Status = domain.StoryStatusDisabled
	if _, err := storyRepo.Update(ctx, story); err != nil {
		t.Fatalf("story update failed: %v", err)
	}

	report := createTestReport(t, reportRepo, ctx, story.ID)
	if _, err := reportRepo.UpdateStatus(ctx, report.ID, domain.ReportStatusDismissed); err != nil {
		t.Fatalf("report update failed: %v", err)
	}

	result, err := reportRepo.ModerateDisableStory(ctx, report.ID)
	if err != nil {
		t.Fatalf("ModerateDisableStory failed: %v", err)
	}

	if result.Report.Status != domain.ReportStatusDismissed {
		t.Fatalf("report status = %q, want %q", result.Report.Status, domain.ReportStatusDismissed)
	}
	if result.Story.Status != domain.StoryStatusDisabled {
		t.Fatalf("story status = %q, want %q", result.Story.Status, domain.StoryStatusDisabled)
	}
}
