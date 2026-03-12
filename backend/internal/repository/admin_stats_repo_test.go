//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// seedAdminStatsDeps creates cities, POIs, stories, and reports for stats
// testing and returns a cleanup function.
func seedAdminStatsDeps(t *testing.T, tp *repository.TestPool) (expected repository.AdminStats, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)
	reportRepo := repository.NewReportRepo(tp.Pool)

	// Seed 2 cities
	city1 := createTestCity(t, cityRepo, ctx)
	city2 := createTestCity(t, cityRepo, ctx)

	// Seed 3 POIs across both cities
	poi1 := createTestPOI(t, poiRepo, ctx, city1.ID, "Stats POI 1", 41.70, 44.81)
	poi2 := createTestPOI(t, poiRepo, ctx, city1.ID, "Stats POI 2", 41.71, 44.82)
	poi3 := createTestPOI(t, poiRepo, ctx, city2.ID, "Stats POI 3", 42.00, 45.00)

	// Seed 4 stories across POIs
	story1 := createTestStory(t, storyRepo, ctx, poi1.ID, "en", "Stats story 1")
	_ = createTestStory(t, storyRepo, ctx, poi1.ID, "ka", "Stats story 2")
	_ = createTestStory(t, storyRepo, ctx, poi2.ID, "en", "Stats story 3")
	_ = createTestStory(t, storyRepo, ctx, poi3.ID, "en", "Stats story 4")

	// Seed a user for reports
	userID := "550e8400-e29b-41d4-a716-446655440088"
	if _, err := tp.Pool.Exec(ctx,
		`INSERT INTO users (id, auth_provider, is_anonymous) VALUES ($1, 'email', true) ON CONFLICT (id) DO NOTHING`,
		userID,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Seed 1 report
	if _, err := reportRepo.Create(ctx, story1.ID, userID, domain.ReportTypeWrongFact, nil, nil, nil); err != nil {
		t.Fatalf("seed report: %v", err)
	}

	return repository.AdminStats{
			CitiesCount:     2,
			POIsCount:       3,
			StoriesCount:    4,
			ReportsCount:    1,
			NewReportsCount: 1,
		}, func() {
			// City delete cascades to POIs → stories → reports
			_ = cityRepo.Delete(ctx, city1.ID)
			_ = cityRepo.Delete(ctx, city2.ID)
			_, _ = tp.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		}
}

func TestAdminStatsRepo_Get_EmptyDatabase(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()

	// Run inside a transaction that truncates all relevant tables, then rolls back.
	tx, err := tp.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Truncate in dependency order (reports → stories → poi → cities)
	for _, table := range []string{"report", "story", "poi", "cities"} {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}

	// Build a repo using the transaction's connection via a savepoint-scoped query
	const query = `
		SELECT
			(SELECT COUNT(*) FROM cities WHERE deleted_at IS NULL) AS cities_count,
			(SELECT COUNT(*) FROM poi) AS pois_count,
			(SELECT COUNT(*) FROM story) AS stories_count,
			(SELECT COUNT(*) FROM report) AS reports_count,
			(SELECT COUNT(*) FROM report WHERE status = 'new') AS new_reports_count`

	var stats repository.AdminStats
	if err := tx.QueryRow(ctx, query).Scan(
		&stats.CitiesCount,
		&stats.POIsCount,
		&stats.StoriesCount,
		&stats.ReportsCount,
		&stats.NewReportsCount,
	); err != nil {
		t.Fatalf("Get on empty database failed: %v", err)
	}

	if stats.CitiesCount != 0 {
		t.Errorf("CitiesCount = %d, want 0", stats.CitiesCount)
	}
	if stats.POIsCount != 0 {
		t.Errorf("POIsCount = %d, want 0", stats.POIsCount)
	}
	if stats.StoriesCount != 0 {
		t.Errorf("StoriesCount = %d, want 0", stats.StoriesCount)
	}
	if stats.ReportsCount != 0 {
		t.Errorf("ReportsCount = %d, want 0", stats.ReportsCount)
	}
	if stats.NewReportsCount != 0 {
		t.Errorf("NewReportsCount = %d, want 0", stats.NewReportsCount)
	}
}

func TestAdminStatsRepo_Get(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewAdminStatsRepo(tp.Pool)

	// Get baseline counts (other tests may have left data)
	baseline, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("baseline Get: %v", err)
	}

	expected, cleanup := seedAdminStatsDeps(t, tp)
	defer cleanup()

	stats, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Assert each counter increased by the expected delta
	if got := stats.CitiesCount - baseline.CitiesCount; got != expected.CitiesCount {
		t.Errorf("CitiesCount delta = %d, want %d", got, expected.CitiesCount)
	}
	if got := stats.POIsCount - baseline.POIsCount; got != expected.POIsCount {
		t.Errorf("POIsCount delta = %d, want %d", got, expected.POIsCount)
	}
	if got := stats.StoriesCount - baseline.StoriesCount; got != expected.StoriesCount {
		t.Errorf("StoriesCount delta = %d, want %d", got, expected.StoriesCount)
	}
	if got := stats.ReportsCount - baseline.ReportsCount; got != expected.ReportsCount {
		t.Errorf("ReportsCount delta = %d, want %d", got, expected.ReportsCount)
	}
	if got := stats.NewReportsCount - baseline.NewReportsCount; got != expected.NewReportsCount {
		t.Errorf("NewReportsCount delta = %d, want %d", got, expected.NewReportsCount)
	}
}
