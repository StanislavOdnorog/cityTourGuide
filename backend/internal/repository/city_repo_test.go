//go:build integration

package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

func setupTestPool(t *testing.T) *repository.TestPool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://citystories:citystories_secret@localhost:5433/citystories?sslmode=disable"
	}

	tp, err := repository.NewTestPool(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to create test pool: %v", err)
	}

	return tp
}

func TestCityRepo_Create(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	city := &domain.City{
		Name:           "Tbilisi",
		Country:        "Georgia",
		CenterLat:      41.7151,
		CenterLng:      44.8271,
		RadiusKm:       10.0,
		IsActive:       true,
		DownloadSizeMB: 50.5,
	}

	created, err := repo.Create(ctx, city)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.Name != "Tbilisi" {
		t.Errorf("expected name Tbilisi, got %s", created.Name)
	}
	if created.Country != "Georgia" {
		t.Errorf("expected country Georgia, got %s", created.Country)
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}

	// Clean up
	_ = repo.Delete(ctx, created.ID)
}

func TestCityRepo_GetByID(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	city := &domain.City{
		Name:      "Batumi",
		Country:   "Georgia",
		CenterLat: 41.6168,
		CenterLng: 41.6367,
		RadiusKm:  5.0,
		IsActive:  true,
	}

	created, err := repo.Create(ctx, city)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = repo.Delete(ctx, created.ID) }()

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != "Batumi" {
		t.Errorf("expected name Batumi, got %s", got.Name)
	}
	if got.CenterLat != 41.6168 {
		t.Errorf("expected center_lat 41.6168, got %f", got.CenterLat)
	}
}

func TestCityRepo_GetByID_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCityRepo_GetAll(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	// Create two cities
	c1, err := repo.Create(ctx, &domain.City{
		Name: "Alpha City", Country: "TestLand",
		CenterLat: 1.0, CenterLng: 2.0, RadiusKm: 5.0, IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create c1 failed: %v", err)
	}
	defer func() { _ = repo.Delete(ctx, c1.ID) }()

	c2, err := repo.Create(ctx, &domain.City{
		Name: "Beta City", Country: "TestLand",
		CenterLat: 3.0, CenterLng: 4.0, RadiusKm: 8.0, IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create c2 failed: %v", err)
	}
	defer func() { _ = repo.Delete(ctx, c2.ID) }()

	cities, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(cities) < 2 {
		t.Errorf("expected at least 2 cities, got %d", len(cities))
	}
}

func TestCityRepo_List_PaginationTraversal(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	var baselineID int
	if err := tp.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM cities`).Scan(&baselineID); err != nil {
		t.Fatalf("query baseline id: %v", err)
	}

	created := make([]*domain.City, 0, 3)
	for i := 0; i < 3; i++ {
		city, err := repo.Create(ctx, &domain.City{
			Name:      "Pagination City " + string(rune('A'+i)),
			Country:   "TestLand",
			CenterLat: 40.0 + float64(i),
			CenterLng: 44.0 + float64(i),
			RadiusKm:  5.0,
			IsActive:  true,
		})
		if err != nil {
			t.Fatalf("create city %d: %v", i, err)
		}
		created = append(created, city)
		defer func(id int) { _ = repo.Delete(ctx, id) }(city.ID)
	}

	page := domain.PageRequest{
		Cursor: domain.EncodeCursor(baselineID),
		Limit:  2,
	}

	firstPage, err := repo.List(ctx, page)
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

	secondPage, err := repo.List(ctx, domain.PageRequest{
		Cursor: firstPage.NextCursor,
		Limit:  2,
	})
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
	for _, city := range firstPage.Items {
		seen[city.ID]++
	}
	for _, city := range secondPage.Items {
		seen[city.ID]++
	}

	for _, city := range created {
		if seen[city.ID] != 1 {
			t.Fatalf("expected city %d to appear exactly once, got %d", city.ID, seen[city.ID])
		}
	}
}

func TestCityRepo_List_InvalidCursor(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	_, err := repo.List(ctx, domain.PageRequest{Cursor: "not-base64", Limit: 20})
	if err == nil {
		t.Fatal("expected error for invalid cursor")
	}
	if got := err.Error(); !strings.Contains(got, "invalid cursor") {
		t.Fatalf("expected wrapped descriptive error, got %q", got)
	}
}

func TestCityRepo_Update(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, &domain.City{
		Name: "Old Name", Country: "OldCountry",
		CenterLat: 10.0, CenterLng: 20.0, RadiusKm: 5.0, IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = repo.Delete(ctx, created.ID) }()

	nameRu := "Новое Имя"
	created.Name = "New Name"
	created.NameRu = &nameRu
	created.Country = "NewCountry"

	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("expected name New Name, got %s", updated.Name)
	}
	if updated.NameRu == nil || *updated.NameRu != "Новое Имя" {
		t.Error("expected name_ru to be 'Новое Имя'")
	}
	if updated.Country != "NewCountry" {
		t.Errorf("expected country NewCountry, got %s", updated.Country)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) {
		t.Error("expected updated_at to be after created_at")
	}
}

func TestCityRepo_Update_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	_, err := repo.Update(ctx, &domain.City{
		ID: 999999, Name: "Ghost", Country: "Nowhere",
		CenterLat: 0, CenterLng: 0, RadiusKm: 1, IsActive: false,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCityRepo_Delete(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, &domain.City{
		Name: "ToDelete", Country: "TestLand",
		CenterLat: 5.0, CenterLng: 6.0, RadiusKm: 3.0, IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = repo.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = repo.GetByID(ctx, created.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCityRepo_Delete_NotFound(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	err := repo.Delete(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCityRepo_FullCRUDCycle(t *testing.T) {
	tp := setupTestPool(t)
	defer tp.Close()

	repo := repository.NewCityRepo(tp.Pool)
	ctx := context.Background()

	// Create
	city, err := repo.Create(ctx, &domain.City{
		Name: "CycleCity", Country: "TestLand",
		CenterLat: 41.7151, CenterLng: 44.8271, RadiusKm: 10.0, IsActive: true,
		DownloadSizeMB: 25.0,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Read
	got, err := repo.GetByID(ctx, city.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "CycleCity" {
		t.Errorf("expected CycleCity, got %s", got.Name)
	}

	// Update
	got.Name = "UpdatedCycleCity"
	updated, err := repo.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "UpdatedCycleCity" {
		t.Errorf("expected UpdatedCycleCity, got %s", updated.Name)
	}

	// Delete
	err = repo.Delete(ctx, updated.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = repo.GetByID(ctx, updated.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
