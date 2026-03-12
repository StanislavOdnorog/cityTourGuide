//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// assertIndexExists checks that the named index exists in pg_indexes.
func assertIndexExists(t *testing.T, pool *pgxpool.Pool, tableName, indexName string) {
	t.Helper()
	ctx := context.Background()

	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes WHERE tablename = $1 AND indexname = $2`,
		tableName, indexName,
	).Scan(&count)
	if err != nil {
		t.Fatalf("pg_indexes query for %s: %v", indexName, err)
	}
	if count == 0 {
		t.Errorf("expected index %s on table %s to exist", indexName, tableName)
	}
}

// explainUsesIndex acquires a connection, disables seq scans for that session,
// runs EXPLAIN (FORMAT JSON), and checks that the plan references the expected
// index name. Disabling seq scans forces the planner to reveal whether the
// index is usable, regardless of table size.
func explainUsesIndex(t *testing.T, pool *pgxpool.Pool, indexName, query string, args ...interface{}) {
	t.Helper()
	ctx := context.Background()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	// Disable sequential scans so the planner must use an index if available.
	if _, err := conn.Exec(ctx, "SET enable_seqscan = off"); err != nil {
		t.Fatalf("SET enable_seqscan: %v", err)
	}

	explainQuery := "EXPLAIN (FORMAT JSON) " + query
	var planJSON json.RawMessage
	err = conn.QueryRow(ctx, explainQuery, args...).Scan(&planJSON)
	if err != nil {
		t.Fatalf("EXPLAIN failed: %v", err)
	}

	if !strings.Contains(string(planJSON), indexName) {
		t.Errorf("expected query plan to reference index %s, got:\n%s", indexName, string(planJSON))
	}
}

func TestIndexes_Exist(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	tests := []struct {
		table string
		index string
	}{
		{"report", "idx_report_status_id"},
		{"story", "idx_story_poi_lang_status_order"},
		{"device_tokens", "idx_device_tokens_active"},
	}

	for _, tt := range tests {
		t.Run(tt.index, func(t *testing.T) {
			assertIndexExists(t, tp.Pool, tt.table, tt.index)
		})
	}
}

func TestReportIndex_ListCursorQuery(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	reportRepo := repository.NewReportRepo(tp.Pool)
	storyID, userID, cleanup := seedReportDeps(t, tp)
	defer cleanup()

	// Seed enough reports for the planner to prefer the index.
	const count = 200
	for i := 0; i < count; i++ {
		rt := domain.ReportTypeWrongFact
		if i%2 == 0 {
			rt = domain.ReportTypeWrongLocation
		}
		_, err := reportRepo.Create(ctx, storyID, userID, rt, nil, nil, nil)
		if err != nil {
			t.Fatalf("seed report %d: %v", i, err)
		}
	}

	// Mark half as resolved so the status filter is selective.
	_, err := tp.Pool.Exec(ctx,
		`UPDATE report SET status = 'resolved', resolved_at = NOW()
		 WHERE story_id = $1 AND id % 2 = 0`, storyID)
	if err != nil {
		t.Fatalf("update statuses: %v", err)
	}

	// Force Postgres to use real statistics.
	_, err = tp.Pool.Exec(ctx, "ANALYZE report")
	if err != nil {
		t.Fatalf("ANALYZE: %v", err)
	}

	// Verify List still returns correct results.
	result, err := reportRepo.List(ctx, string(domain.ReportStatusNew), domain.PageRequest{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, item := range result.Items {
		if item.Status != domain.ReportStatusNew {
			t.Errorf("expected status new, got %s", item.Status)
		}
	}

	// Verify ListAdmin returns same data pattern.
	adminResult, err := reportRepo.ListAdmin(ctx, string(domain.ReportStatusNew), domain.PageRequest{Limit: 10}, repository.ListSort{})
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}
	for _, item := range adminResult.Items {
		if item.Status != domain.ReportStatusNew {
			t.Errorf("ListAdmin: expected status new, got %s", item.Status)
		}
	}

	// Check EXPLAIN references the index for the cursor query.
	explainUsesIndex(t, tp.Pool, "idx_report_status_id",
		`SELECT id, story_id, user_id, type, comment,
		        user_lat, user_lng, status, resolved_at, created_at
		 FROM report
		 WHERE status = $1 AND id > $2
		 ORDER BY id ASC LIMIT $3`,
		string(domain.ReportStatusNew), 0, 10,
	)
}

func TestStoryIndex_GetByPOIIDQuery(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	storyRepo := repository.NewStoryRepo(tp.Pool)

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Index Test POI", 41.69, 44.80)

	// Seed stories across multiple POIs so the index is selective.
	const storiesPerPOI = 50
	const extraPOIs = 4
	pois := []*domain.POI{poi}
	for i := 0; i < extraPOIs; i++ {
		p := createTestPOI(t, poiRepo, ctx, city.ID,
			fmt.Sprintf("Extra POI %d", i), 41.69+float64(i)*0.01, 44.80)
		pois = append(pois, p)
	}

	for _, p := range pois {
		for i := 0; i < storiesPerPOI; i++ {
			_, err := storyRepo.Create(ctx, &domain.Story{
				POIID:      p.ID,
				Language:   "en",
				Text:       fmt.Sprintf("Story %d for POI %d", i, p.ID),
				LayerType:  domain.StoryLayerGeneral,
				OrderIndex: int16(i),
				Confidence: 80,
				Sources:    json.RawMessage(`[]`),
				Status:     domain.StoryStatusActive,
			})
			if err != nil {
				t.Fatalf("seed story: %v", err)
			}
		}
	}

	// ANALYZE for fresh statistics.
	_, err := tp.Pool.Exec(ctx, "ANALYZE story")
	if err != nil {
		t.Fatalf("ANALYZE: %v", err)
	}

	// Verify GetByPOIID returns correct results.
	activeStatus := domain.StoryStatusActive
	stories, err := storyRepo.GetByPOIID(ctx, poi.ID, "en", &activeStatus)
	if err != nil {
		t.Fatalf("GetByPOIID: %v", err)
	}
	if len(stories) != storiesPerPOI {
		t.Errorf("expected %d stories, got %d", storiesPerPOI, len(stories))
	}
	// Verify ordering by order_index.
	for i := 1; i < len(stories); i++ {
		if stories[i].OrderIndex < stories[i-1].OrderIndex {
			t.Errorf("stories not ordered by order_index: [%d]=%d > [%d]=%d",
				i-1, stories[i-1].OrderIndex, i, stories[i].OrderIndex)
		}
	}

	// Verify ListByPOIID returns correct paginated results.
	page, err := storyRepo.ListByPOIID(ctx, poi.ID, "en", &activeStatus, domain.PageRequest{Limit: 10}, repository.ListSort{})
	if err != nil {
		t.Fatalf("ListByPOIID: %v", err)
	}
	if len(page.Items) != 10 {
		t.Errorf("expected 10 items, got %d", len(page.Items))
	}
	if !page.HasMore {
		t.Error("expected HasMore to be true")
	}

	// Verify GetDownloadManifest returns correct results.
	// First, set audio_url on some stories.
	_, err = tp.Pool.Exec(ctx,
		`UPDATE story SET audio_url = 'https://example.com/audio.mp3'
		 WHERE poi_id = $1 AND language = 'en'`, poi.ID)
	if err != nil {
		t.Fatalf("set audio_url: %v", err)
	}

	manifest, err := storyRepo.GetDownloadManifest(ctx, city.ID, "en")
	if err != nil {
		t.Fatalf("GetDownloadManifest: %v", err)
	}
	if len(manifest) == 0 {
		t.Error("expected non-empty download manifest")
	}

	// Verify the new index exists (it was already checked in TestIndexes_Exist,
	// but confirm it's still present after all the seeding).
	assertIndexExists(t, tp.Pool, "story", "idx_story_poi_lang_status_order")
}

func TestDeviceTokenIndex_GetAllActivePageQuery(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	ctx := context.Background()
	repo := repository.NewDeviceTokenRepo(tp.Pool)

	cleanup1 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID)
	defer cleanup1()
	cleanup2 := seedDeviceTokenUser(t, tp, deviceTokenTestUserID2)
	defer cleanup2()

	const count = 200
	for i := 0; i < count; i++ {
		userID := deviceTokenTestUserID
		if i%2 == 0 {
			userID = deviceTokenTestUserID2
		}

		token := fmt.Sprintf("token-index-%03d", i)
		if _, err := repo.Upsert(ctx, userID, token, domain.DevicePlatformIOS); err != nil {
			t.Fatalf("seed device token %d: %v", i, err)
		}
	}

	_, err := tp.Pool.Exec(ctx, `UPDATE device_tokens SET is_active = false WHERE token LIKE 'token-index-%' AND id % 3 = 0`)
	if err != nil {
		t.Fatalf("deactivate seeded tokens: %v", err)
	}

	_, err = tp.Pool.Exec(ctx, "ANALYZE device_tokens")
	if err != nil {
		t.Fatalf("ANALYZE device_tokens: %v", err)
	}

	page, err := repo.GetAllActivePage(ctx, domain.PageRequest{Limit: 25})
	if err != nil {
		t.Fatalf("GetAllActivePage: %v", err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected non-empty active device token page")
	}
	for _, item := range page.Items {
		if !item.IsActive {
			t.Fatal("GetAllActivePage returned inactive token")
		}
	}

	explainUsesIndex(t, tp.Pool, "idx_device_tokens_active",
		`SELECT id, user_id, token, platform, is_active, created_at, updated_at
		 FROM device_tokens
		 WHERE is_active = true
		   AND (
		     user_id > $1
		     OR (user_id = $1 AND updated_at < $2)
		     OR (user_id = $1 AND updated_at = $2 AND id > $3)
		 )
		 ORDER BY user_id ASC, updated_at DESC, id ASC
		 LIMIT $4`,
		deviceTokenTestUserID, time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC), 0, 25,
	)
}
