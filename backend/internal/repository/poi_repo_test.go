//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

func createTestCity(t *testing.T, cityRepo *repository.CityRepo, ctx context.Context) *domain.City {
	t.Helper()
	city, err := cityRepo.Create(ctx, &domain.City{
		Name:      "TestCity",
		Country:   "TestCountry",
		CenterLat: 41.7151,
		CenterLng: 44.8271,
		RadiusKm:  10.0,
		IsActive:  true,
	})
	if err != nil {
		t.Fatalf("createTestCity: %v", err)
	}
	return city
}

func createTestPOI(t *testing.T, poiRepo *repository.POIRepo, ctx context.Context, cityID int, name string, lat, lng float64) *domain.POI {
	t.Helper()
	poi, err := poiRepo.Create(ctx, &domain.POI{
		CityID:        cityID,
		Name:          name,
		Lat:           lat,
		Lng:           lng,
		Type:          domain.POITypeMonument,
		Tags:          json.RawMessage(`{}`),
		InterestScore: 50,
		Status:        domain.POIStatusActive,
	})
	if err != nil {
		t.Fatalf("createTestPOI(%s): %v", name, err)
	}
	return poi
}

func TestPOIRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	nameRu := "Нарикала"
	address := "Narikala, Old Tbilisi"
	poi := &domain.POI{
		CityID:        city.ID,
		Name:          "Narikala Fortress",
		NameRu:        &nameRu,
		Lat:           41.6875,
		Lng:           44.8074,
		Type:          domain.POITypeMonument,
		Tags:          json.RawMessage(`{"era": "4th century"}`),
		Address:       &address,
		InterestScore: 90,
		Status:        domain.POIStatusActive,
	}

	created, err := poiRepo.Create(ctx, poi)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.Name != "Narikala Fortress" {
		t.Errorf("expected name 'Narikala Fortress', got %s", created.Name)
	}
	if created.NameRu == nil || *created.NameRu != "Нарикала" {
		t.Error("expected name_ru 'Нарикала'")
	}
	if created.Lat < 41.68 || created.Lat > 41.69 {
		t.Errorf("expected lat ~41.6875, got %f", created.Lat)
	}
	if created.Lng < 44.80 || created.Lng > 44.81 {
		t.Errorf("expected lng ~44.8074, got %f", created.Lng)
	}
	if created.InterestScore != 90 {
		t.Errorf("expected interest_score 90, got %d", created.InterestScore)
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestPOIRepo_GetByID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	created := createTestPOI(t, poiRepo, ctx, city.ID, "Rike Park", 41.6927, 44.8090)

	got, err := poiRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != "Rike Park" {
		t.Errorf("expected 'Rike Park', got %s", got.Name)
	}
	if got.CityID != city.ID {
		t.Errorf("expected city_id %d, got %d", city.ID, got.CityID)
	}
}

func TestPOIRepo_GetByID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	_, err := poiRepo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPOIRepo_GetByCityID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	createTestPOI(t, poiRepo, ctx, city.ID, "POI A", 41.69, 44.80)
	createTestPOI(t, poiRepo, ctx, city.ID, "POI B", 41.70, 44.81)

	pois, err := poiRepo.GetByCityID(ctx, city.ID, nil, nil)
	if err != nil {
		t.Fatalf("GetByCityID failed: %v", err)
	}

	if len(pois) != 2 {
		t.Errorf("expected 2 POIs, got %d", len(pois))
	}
}

func TestPOIRepo_GetByCityID_WithFilters(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	// Create POIs with different types
	poi1 := &domain.POI{
		CityID: city.ID, Name: "Church 1", Lat: 41.69, Lng: 44.80,
		Type: domain.POITypeChurch, Tags: json.RawMessage(`{}`),
		InterestScore: 50, Status: domain.POIStatusActive,
	}
	_, err := poiRepo.Create(ctx, poi1)
	if err != nil {
		t.Fatalf("Create poi1 failed: %v", err)
	}

	poi2 := &domain.POI{
		CityID: city.ID, Name: "Park 1", Lat: 41.70, Lng: 44.81,
		Type: domain.POITypePark, Tags: json.RawMessage(`{}`),
		InterestScore: 50, Status: domain.POIStatusActive,
	}
	_, err = poiRepo.Create(ctx, poi2)
	if err != nil {
		t.Fatalf("Create poi2 failed: %v", err)
	}

	// Filter by type
	churchType := domain.POITypeChurch
	pois, err := poiRepo.GetByCityID(ctx, city.ID, nil, &churchType)
	if err != nil {
		t.Fatalf("GetByCityID with type filter failed: %v", err)
	}
	if len(pois) != 1 {
		t.Errorf("expected 1 church POI, got %d", len(pois))
	}
	if len(pois) > 0 && pois[0].Type != domain.POITypeChurch {
		t.Errorf("expected type 'church', got %s", pois[0].Type)
	}

	// Filter by status
	activeStatus := domain.POIStatusActive
	pois, err = poiRepo.GetByCityID(ctx, city.ID, &activeStatus, nil)
	if err != nil {
		t.Fatalf("GetByCityID with status filter failed: %v", err)
	}
	if len(pois) != 2 {
		t.Errorf("expected 2 active POIs, got %d", len(pois))
	}
}

func TestPOIRepo_Update(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	created := createTestPOI(t, poiRepo, ctx, city.ID, "Old Name", 41.69, 44.80)

	nameRu := "Новое Имя"
	created.Name = "New Name"
	created.NameRu = &nameRu
	created.InterestScore = 95
	created.Lat = 41.70
	created.Lng = 44.82

	updated, err := poiRepo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("expected 'New Name', got %s", updated.Name)
	}
	if updated.NameRu == nil || *updated.NameRu != "Новое Имя" {
		t.Error("expected name_ru 'Новое Имя'")
	}
	if updated.InterestScore != 95 {
		t.Errorf("expected interest_score 95, got %d", updated.InterestScore)
	}
	if updated.Lat < 41.69 || updated.Lat > 41.71 {
		t.Errorf("expected lat ~41.70, got %f", updated.Lat)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) {
		t.Error("expected updated_at to be after created_at")
	}
}

func TestPOIRepo_Update_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	_, err := poiRepo.Update(ctx, &domain.POI{
		ID: 999999, CityID: city.ID, Name: "Ghost", Lat: 0, Lng: 0,
		Type: domain.POITypeOther, Tags: json.RawMessage(`{}`),
		InterestScore: 50, Status: domain.POIStatusActive,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPOIRepo_Delete(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	created := createTestPOI(t, poiRepo, ctx, city.ID, "ToDelete", 41.69, 44.80)

	err := poiRepo.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = poiRepo.GetByID(ctx, created.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestPOIRepo_Delete_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	err := poiRepo.Delete(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPOIRepo_ListByCityID_PaginationTraversal(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	created := []*domain.POI{
		createTestPOI(t, poiRepo, ctx, city.ID, "Page POI 1", 41.6900, 44.8000),
		createTestPOI(t, poiRepo, ctx, city.ID, "Page POI 2", 41.6910, 44.8010),
		createTestPOI(t, poiRepo, ctx, city.ID, "Page POI 3", 41.6920, 44.8020),
	}

	firstPage, err := poiRepo.ListByCityID(ctx, city.ID, nil, nil, domain.PageRequest{Limit: 2}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list first page: %v", err)
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

	secondPage, err := poiRepo.ListByCityID(ctx, city.ID, nil, nil, domain.PageRequest{
		Cursor: firstPage.NextCursor,
		Limit:  2,
	}, repository.ListSort{})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}

	if len(secondPage.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(secondPage.Items))
	}
	if secondPage.HasMore {
		t.Fatal("expected second page to be terminal")
	}
	if secondPage.NextCursor != "" {
		t.Fatal("expected empty next cursor on terminal page")
	}

	seen := make(map[int]int)
	for _, poi := range firstPage.Items {
		seen[poi.ID]++
	}
	for _, poi := range secondPage.Items {
		seen[poi.ID]++
	}

	for _, poi := range created {
		if seen[poi.ID] != 1 {
			t.Fatalf("expected poi %d to appear exactly once, got %d", poi.ID, seen[poi.ID])
		}
	}
}

func TestPOIRepo_ListByCityID_InvalidCursor(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	_, err := poiRepo.ListByCityID(ctx, city.ID, nil, nil, domain.PageRequest{
		Cursor: "not-base64",
		Limit:  20,
	}, repository.ListSort{})
	if err == nil {
		t.Fatal("expected error for invalid cursor")
	}
	if got := err.Error(); !strings.Contains(got, "invalid cursor") {
		t.Fatalf("expected wrapped descriptive error, got %q", got)
	}
}

func TestPOIRepo_ListByCityID_SortsByInterestScoreDesc(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	low := createTestPOI(t, poiRepo, ctx, city.ID, "Low", 41.69, 44.80)
	medium := createTestPOI(t, poiRepo, ctx, city.ID, "Medium", 41.691, 44.801)
	high := createTestPOI(t, poiRepo, ctx, city.ID, "High", 41.692, 44.802)

	_, err := tp.Pool.Exec(ctx, `UPDATE poi SET interest_score = 10 WHERE id = $1`, low.ID)
	if err != nil {
		t.Fatalf("update low score: %v", err)
	}
	_, err = tp.Pool.Exec(ctx, `UPDATE poi SET interest_score = 50 WHERE id = $1`, medium.ID)
	if err != nil {
		t.Fatalf("update medium score: %v", err)
	}
	_, err = tp.Pool.Exec(ctx, `UPDATE poi SET interest_score = 90 WHERE id = $1`, high.ID)
	if err != nil {
		t.Fatalf("update high score: %v", err)
	}

	result, err := poiRepo.ListByCityID(ctx, city.ID, nil, nil, domain.PageRequest{Limit: 10}, repository.ListSort{
		By:  "interest_score",
		Dir: repository.SortDirDesc,
	})
	if err != nil {
		t.Fatalf("ListByCityID failed: %v", err)
	}

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 pois, got %d", len(result.Items))
	}
	if result.Items[0].Name != "High" || result.Items[1].Name != "Medium" || result.Items[2].Name != "Low" {
		t.Fatalf("unexpected order: %q, %q, %q", result.Items[0].Name, result.Items[1].Name, result.Items[2].Name)
	}
}

func TestPOIRepo_FindNearby_PaginationByDistanceCursor(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi1 := createTestPOI(t, poiRepo, ctx, city.ID, "Nearby Page 1", 41.6927, 44.8090)
	poi2 := createTestPOI(t, poiRepo, ctx, city.ID, "Nearby Page 2", 41.6932, 44.8078)
	poi3 := createTestPOI(t, poiRepo, ctx, city.ID, "Nearby Page 3", 41.6940, 44.8065)

	for _, poiID := range []int{poi1.ID, poi2.ID, poi3.ID} {
		if _, err := tp.Pool.Exec(ctx,
			`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'en', 'Nearby story', 'general', 'active')`,
			poiID,
		); err != nil {
			t.Fatalf("insert story for poi %d: %v", poiID, err)
		}
	}

	firstPage, err := poiRepo.FindNearby(ctx, 41.6927, 44.8090, 500, city.ID, "en", domain.PageRequest{Limit: 2})
	if err != nil {
		t.Fatalf("find nearby first page: %v", err)
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

	if firstPage.Items[0].DistanceM > firstPage.Items[1].DistanceM {
		t.Fatal("expected first page items sorted by distance ascending")
	}

	secondPage, err := poiRepo.FindNearby(ctx, 41.6927, 44.8090, 500, city.ID, "en", domain.PageRequest{
		Cursor: firstPage.NextCursor,
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("find nearby second page: %v", err)
	}

	if len(secondPage.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(secondPage.Items))
	}
	if secondPage.HasMore {
		t.Fatal("expected second page to be terminal")
	}

	lastFirst := firstPage.Items[len(firstPage.Items)-1]
	firstSecond := secondPage.Items[0]
	if firstSecond.DistanceM < lastFirst.DistanceM {
		t.Fatal("expected second page to continue distance ordering")
	}
	if firstSecond.DistanceM == lastFirst.DistanceM && firstSecond.ID <= lastFirst.ID {
		t.Fatal("expected second page to continue tie-break ordering by id")
	}
}

func TestPOIRepo_FindNearby_InvalidCursor(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	_, err := poiRepo.FindNearby(ctx, 41.6927, 44.8090, 500, city.ID, "en", domain.PageRequest{
		Cursor: "not-base64",
		Limit:  20,
	})
	if err == nil {
		t.Fatal("expected error for invalid cursor")
	}
	if got := err.Error(); !strings.Contains(got, "invalid cursor") {
		t.Fatalf("expected wrapped descriptive error, got %q", got)
	}
}

func TestPOIRepo_FindNearby(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	// Create 5 POIs around Tbilisi center (41.7151, 44.8271)
	// Rike Park - very close to center
	poi1 := createTestPOI(t, poiRepo, ctx, city.ID, "Rike Park", 41.6927, 44.8090)
	// Narikala - ~500m from Rike Park
	poi2 := createTestPOI(t, poiRepo, ctx, city.ID, "Narikala Fortress", 41.6875, 44.8074)
	// Peace Bridge - close to Rike Park
	poi3 := createTestPOI(t, poiRepo, ctx, city.ID, "Peace Bridge", 41.6932, 44.8078)
	// Metekhi Church - ~300m from Rike Park
	createTestPOI(t, poiRepo, ctx, city.ID, "Metekhi Church", 41.6909, 44.8114)
	// Mtatsminda - further away (~2km)
	createTestPOI(t, poiRepo, ctx, city.ID, "Mtatsminda Park", 41.6945, 44.7867)

	// Create stories for 3 POIs (only POIs with stories should appear)
	_, err := tp.Pool.Exec(ctx,
		`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'en', 'Test story 1', 'general', 'active')`,
		poi1.ID)
	if err != nil {
		t.Fatalf("insert story 1: %v", err)
	}
	_, err = tp.Pool.Exec(ctx,
		`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'en', 'Test story 2', 'atmosphere', 'active')`,
		poi2.ID)
	if err != nil {
		t.Fatalf("insert story 2: %v", err)
	}
	_, err = tp.Pool.Exec(ctx,
		`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'en', 'Test story 3', 'human_story', 'active')`,
		poi3.ID)
	if err != nil {
		t.Fatalf("insert story 3: %v", err)
	}

	// Search from Rike Park area with 500m radius
	results, err := poiRepo.FindNearby(ctx, 41.6927, 44.8090, 500, city.ID, "en", domain.PageRequest{Limit: domain.DefaultPageLimit})
	if err != nil {
		t.Fatalf("FindNearby failed: %v", err)
	}

	if len(results.Items) < 2 {
		t.Errorf("expected at least 2 nearby POIs with stories, got %d", len(results.Items))
	}

	// Verify distance_m is populated and reasonable
	for _, r := range results.Items {
		if r.DistanceM < 0 {
			t.Errorf("expected non-negative distance_m, got %f for %s", r.DistanceM, r.Name)
		}
	}
}

func TestPOIRepo_FindNearby_SmallRadius(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	// Close POI
	closePOI := createTestPOI(t, poiRepo, ctx, city.ID, "Close POI", 41.6927, 44.8090)
	// Far POI (~2km away)
	farPOI := createTestPOI(t, poiRepo, ctx, city.ID, "Far POI", 41.7100, 44.8090)

	// Stories for both
	_, _ = tp.Pool.Exec(ctx,
		`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'en', 'Close story', 'general', 'active')`,
		closePOI.ID)
	_, _ = tp.Pool.Exec(ctx,
		`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'en', 'Far story', 'general', 'active')`,
		farPOI.ID)

	// Search with very small radius (50m) from closePOI location
	results, err := poiRepo.FindNearby(ctx, 41.6927, 44.8090, 50, city.ID, "en", domain.PageRequest{Limit: domain.DefaultPageLimit})
	if err != nil {
		t.Fatalf("FindNearby small radius failed: %v", err)
	}

	// Should find only the close POI
	if len(results.Items) != 1 {
		t.Errorf("expected 1 nearby POI with small radius, got %d", len(results.Items))
	}
	if len(results.Items) > 0 && results.Items[0].Name != "Close POI" {
		t.Errorf("expected 'Close POI', got %s", results.Items[0].Name)
	}
}

func TestPOIRepo_FindNearby_LanguageFilter(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	poi := createTestPOI(t, poiRepo, ctx, city.ID, "Lang Test POI", 41.6927, 44.8090)

	// Only Russian story
	_, _ = tp.Pool.Exec(ctx,
		`INSERT INTO story (poi_id, language, text, layer_type, status) VALUES ($1, 'ru', 'Тестовая история', 'general', 'active')`,
		poi.ID)

	// Search for English stories — should find none
	results, err := poiRepo.FindNearby(ctx, 41.6927, 44.8090, 500, city.ID, "en", domain.PageRequest{Limit: domain.DefaultPageLimit})
	if err != nil {
		t.Fatalf("FindNearby EN failed: %v", err)
	}
	if len(results.Items) != 0 {
		t.Errorf("expected 0 results for EN, got %d", len(results.Items))
	}

	// Search for Russian stories — should find one
	results, err = poiRepo.FindNearby(ctx, 41.6927, 44.8090, 500, city.ID, "ru", domain.PageRequest{Limit: domain.DefaultPageLimit})
	if err != nil {
		t.Fatalf("FindNearby RU failed: %v", err)
	}
	if len(results.Items) != 1 {
		t.Errorf("expected 1 result for RU, got %d", len(results.Items))
	}
}

func TestPOIRepo_FullCRUDCycle(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	cityRepo := repository.NewCityRepo(tp.Pool)
	poiRepo := repository.NewPOIRepo(tp.Pool)
	ctx := context.Background()

	city := createTestCity(t, cityRepo, ctx)
	defer func() { _ = cityRepo.Delete(ctx, city.ID) }()

	// Create
	poi, err := poiRepo.Create(ctx, &domain.POI{
		CityID: city.ID, Name: "CyclePOI", Lat: 41.69, Lng: 44.80,
		Type: domain.POITypeMuseum, Tags: json.RawMessage(`{"floor": 2}`),
		InterestScore: 70, Status: domain.POIStatusActive,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Read
	got, err := poiRepo.GetByID(ctx, poi.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "CyclePOI" {
		t.Errorf("expected CyclePOI, got %s", got.Name)
	}

	// Update
	got.Name = "UpdatedCyclePOI"
	got.InterestScore = 85
	updated, err := poiRepo.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "UpdatedCyclePOI" {
		t.Errorf("expected UpdatedCyclePOI, got %s", updated.Name)
	}

	// Delete
	err = poiRepo.Delete(ctx, updated.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = poiRepo.GetByID(ctx, updated.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
