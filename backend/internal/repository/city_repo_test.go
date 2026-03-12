//go:build integration

package repository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/testutil/dbtest"
)

func TestCityRepo_Create(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
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
}

func TestCityRepo_GetByID(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	city := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{
		Name: "Batumi", Country: "Georgia",
		CenterLat: 41.6168, CenterLng: 41.6367, RadiusKm: 5.0,
	})

	got, err := repo.GetByID(ctx, city.ID, false)
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
	h := dbtest.Get(t)
	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999999, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCityRepo_GetAll(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Alpha City"})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Beta City"})

	cities, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(cities) < 2 {
		t.Errorf("expected at least 2 cities, got %d", len(cities))
	}
}

func TestCityRepo_List_PaginationTraversal(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	created := make([]*domain.City, 0, 3)
	for i := 0; i < 3; i++ {
		city := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{
			Name:      "Pagination City " + string(rune('A'+i)),
			CenterLat: 40.0 + float64(i),
			CenterLng: 44.0 + float64(i),
		})
		created = append(created, city)
	}

	page := domain.PageRequest{Limit: 2}

	firstPage, err := repo.List(ctx, page, false, repository.ListSort{})
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
	}, false, repository.ListSort{})
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
	h := dbtest.Get(t)
	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	_, err := repo.List(ctx, domain.PageRequest{Cursor: "not-base64", Limit: 20}, false, repository.ListSort{})
	if err == nil {
		t.Fatal("expected error for invalid cursor")
	}
	if got := err.Error(); !strings.Contains(got, "invalid cursor") {
		t.Fatalf("expected wrapped descriptive error, got %q", got)
	}
}

func TestCityRepo_List_SortsByNameDesc(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Alpha"})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Zulu"})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Bravo"})

	result, err := repo.List(ctx, domain.PageRequest{Limit: 10}, false, repository.ListSort{
		By:  "name",
		Dir: repository.SortDirDesc,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 cities, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Zulu" || result.Items[1].Name != "Bravo" || result.Items[2].Name != "Alpha" {
		t.Fatalf("unexpected order: %q, %q, %q", result.Items[0].Name, result.Items[1].Name, result.Items[2].Name)
	}
}

func TestCityRepo_Update(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	created := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{
		Name: "Old Name", Country: "OldCountry",
		CenterLat: 10.0, CenterLng: 20.0, RadiusKm: 5.0,
	})

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
	h := dbtest.Get(t)
	repo := repository.NewCityRepo(h.Pool)
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
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	created := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "ToDelete"})

	err := repo.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Soft-deleted city is still visible via GetByID (admin path).
	got, err := repo.GetByID(ctx, created.ID, true)
	if err != nil {
		t.Fatalf("expected soft-deleted city to be visible via GetByID, got %v", err)
	}
	if got.DeletedAt == nil {
		t.Error("expected deleted_at to be set after soft delete")
	}

	// Soft-deleted city is NOT visible via GetActiveByID (public path).
	_, err = repo.GetActiveByID(ctx, created.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound for soft-deleted city via GetActiveByID, got %v", err)
	}
}

func TestCityRepo_Delete_Idempotent(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	created := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "IdempotentDelete"})

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("first delete: %v", err)
	}

	got1, _ := repo.GetByID(ctx, created.ID, true)
	firstDeletedAt := got1.DeletedAt

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("second delete (idempotent): %v", err)
	}

	got2, _ := repo.GetByID(ctx, created.ID, true)
	if !got2.DeletedAt.Equal(*firstDeletedAt) {
		t.Error("expected deleted_at to remain unchanged on repeated delete")
	}
}

func TestCityRepo_Restore(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	created := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "ToRestore", IsActive: dbtest.BoolPtr(true)})

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	restored, err := repo.Restore(ctx, created.ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Error("expected deleted_at to be nil after restore")
	}
	if restored.Name != "ToRestore" {
		t.Errorf("expected name ToRestore, got %s", restored.Name)
	}

	got, err := repo.GetActiveByID(ctx, restored.ID)
	if err != nil {
		t.Fatalf("expected restored city visible via GetActiveByID: %v", err)
	}
	if got.Name != "ToRestore" {
		t.Errorf("expected name ToRestore, got %s", got.Name)
	}
}

func TestCityRepo_Restore_NotDeleted(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	created := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "NotDeleted"})

	_, err := repo.Restore(ctx, created.ID)
	if err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-deleted city, got %v", err)
	}
}

func TestCityRepo_Restore_NonExistent(t *testing.T) {
	h := dbtest.Get(t)
	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	_, err := repo.Restore(ctx, 999999)
	if err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCityRepo_SoftDeleted_HiddenFromListActive(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	city := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "WillBeDeleted", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "StaysAlive", IsActive: dbtest.BoolPtr(true)})

	if err := repo.Delete(ctx, city.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	result, err := repo.ListActive(ctx, domain.PageRequest{Limit: 100})
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	for _, c := range result.Items {
		if c.ID == city.ID {
			t.Errorf("soft-deleted city %q should not appear in ListActive", c.Name)
		}
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 active city, got %d", len(result.Items))
	}
}

func TestCityRepo_SoftDeleted_VisibleInAdminList(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	city := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "SoftDeleted"})
	if err := repo.Delete(ctx, city.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	result, err := repo.List(ctx, domain.PageRequest{Limit: 100}, true, repository.ListSort{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := false
	for _, c := range result.Items {
		if c.ID == city.ID {
			found = true
			if c.DeletedAt == nil {
				t.Error("expected deleted_at to be set")
			}
		}
	}
	if !found {
		t.Error("soft-deleted city should be visible in admin List")
	}
}

func TestCityRepo_Delete_NotFound(t *testing.T) {
	h := dbtest.Get(t)
	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	err := repo.Delete(ctx, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- Visibility boundary regression tests (integration) ---

func TestCityRepo_ListActive_ExcludesInactiveCities(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	active1 := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Active1", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Inactive1", IsActive: dbtest.BoolPtr(false)})
	active2 := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Active2", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Inactive2", IsActive: dbtest.BoolPtr(false)})

	result, err := repo.ListActive(ctx, domain.PageRequest{Limit: 100})
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 active cities, got %d", len(result.Items))
	}

	ids := map[int]bool{}
	for _, c := range result.Items {
		if !c.IsActive {
			t.Errorf("ListActive returned inactive city %q", c.Name)
		}
		ids[c.ID] = true
	}
	if !ids[active1.ID] || !ids[active2.ID] {
		t.Error("expected both active cities to be returned")
	}
}

func TestCityRepo_List_IncludesAllCities(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Active", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Inactive", IsActive: dbtest.BoolPtr(false)})

	result, err := repo.List(ctx, domain.PageRequest{Limit: 100}, false, repository.ListSort{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 total cities (active+inactive), got %d", len(result.Items))
	}
}

func TestCityRepo_ListActive_PaginationNeverLeaksInactive(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	// Insert 5 cities: alternating active/inactive by ID order.
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "A1", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "I1", IsActive: dbtest.BoolPtr(false)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "A2", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "I2", IsActive: dbtest.BoolPtr(false)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "A3", IsActive: dbtest.BoolPtr(true)})

	// Paginate through all pages with limit=1, collecting results.
	var allItems []domain.City
	cursor := ""
	for i := 0; i < 10; i++ { // safety limit to avoid infinite loop
		page, err := repo.ListActive(ctx, domain.PageRequest{Cursor: cursor, Limit: 1})
		if err != nil {
			t.Fatalf("ListActive page %d: %v", i, err)
		}

		for _, c := range page.Items {
			if !c.IsActive {
				t.Fatalf("page %d: inactive city %q leaked through ListActive pagination", i, c.Name)
			}
		}
		allItems = append(allItems, page.Items...)

		if !page.HasMore {
			break
		}
		cursor = page.NextCursor
	}

	if len(allItems) != 3 {
		t.Errorf("expected 3 active cities across all pages, got %d", len(allItems))
	}
}

func TestCityRepo_GetActiveByID_ReturnsNotFoundForInactive(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	inactive := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Hidden", IsActive: dbtest.BoolPtr(false)})

	_, err := repo.GetActiveByID(ctx, inactive.ID)
	if err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound for inactive city, got %v", err)
	}

	// Same city is visible via GetByID (admin path).
	got, err := repo.GetByID(ctx, inactive.ID, false)
	if err != nil {
		t.Fatalf("GetByID should find inactive city: %v", err)
	}
	if got.Name != "Hidden" {
		t.Errorf("expected name Hidden, got %s", got.Name)
	}
}

func TestCityRepo_ListVsListActive_CountDifference(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Pub1", IsActive: dbtest.BoolPtr(true)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Hid1", IsActive: dbtest.BoolPtr(false)})
	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Pub2", IsActive: dbtest.BoolPtr(true)})

	all, err := repo.List(ctx, domain.PageRequest{Limit: 100}, false, repository.ListSort{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	active, err := repo.ListActive(ctx, domain.PageRequest{Limit: 100})
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}

	if len(all.Items) != 3 {
		t.Errorf("List: expected 3, got %d", len(all.Items))
	}
	if len(active.Items) != 2 {
		t.Errorf("ListActive: expected 2, got %d", len(active.Items))
	}
	if len(active.Items) >= len(all.Items) {
		t.Error("ListActive should return fewer items than List when inactive cities exist")
	}
}

func TestCityRepo_FullCRUDCycle(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
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
	got, err := repo.GetByID(ctx, city.ID, false)
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

	// Soft-delete
	err = repo.Delete(ctx, updated.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify soft-deleted (still visible via GetByID, hidden from GetActiveByID)
	softDeleted, err := repo.GetByID(ctx, updated.ID, true)
	if err != nil {
		t.Fatalf("expected soft-deleted city visible via GetByID: %v", err)
	}
	if softDeleted.DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}

	_, err = repo.GetActiveByID(ctx, updated.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound for soft-deleted city via GetActiveByID, got %v", err)
	}

	// Restore
	restored, err := repo.Restore(ctx, updated.ID)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Error("expected deleted_at to be nil after restore")
	}
}

func TestCityRepo_Restore_Conflict(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	repo := repository.NewCityRepo(h.Pool)
	ctx := context.Background()

	original := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Conflict City", Country: "Georgia"})
	if err := repo.Delete(ctx, original.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "Conflict City", Country: "Georgia"})

	_, err := repo.Restore(ctx, original.ID)
	if err == nil {
		t.Fatal("expected restore conflict, got nil")
	}
	if err != repository.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}
