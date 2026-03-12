//go:build integration

package dbtest_test

import (
	"context"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/testutil/dbtest"
)

func TestMain(m *testing.M) {
	dbtest.Main(m)
}

func TestHarness_PoolIsUsable(t *testing.T) {
	h := dbtest.Get(t)

	var result int
	err := h.Pool.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}

func TestHarness_PostGISAvailable(t *testing.T) {
	h := dbtest.Get(t)

	var version string
	err := h.Pool.QueryRow(context.Background(), "SELECT PostGIS_Version()").Scan(&version)
	if err != nil {
		t.Fatalf("PostGIS not available: %v", err)
	}
	if version == "" {
		t.Fatal("expected non-empty PostGIS version")
	}
}

func TestHarness_TruncateAll(t *testing.T) {
	h := dbtest.Get(t)

	// Insert a city, truncate, verify it's gone.
	city := dbtest.InsertCity(t, h.Pool, dbtest.CityOpts{Name: "TruncateTest"})
	if city.ID == 0 {
		t.Fatal("expected non-zero city ID")
	}

	h.TruncateAll(t)

	var count int
	err := h.Pool.QueryRow(context.Background(), "SELECT count(*) FROM cities").Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 cities after truncate, got %d", count)
	}
}

func TestFixtures_InsertCity(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	city := dbtest.InsertCity(t, h.Pool)
	if city.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if city.Name != "Test City" {
		t.Errorf("expected default name 'Test City', got %q", city.Name)
	}
}

func TestFixtures_InsertStoryChain(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	// InsertStory should auto-create POI and City
	story := dbtest.InsertStory(t, h.Pool)
	if story.ID == 0 {
		t.Error("expected non-zero story ID")
	}
	if story.POIID == 0 {
		t.Error("expected non-zero POI ID")
	}
}

func TestFixtures_InsertReport(t *testing.T) {
	h := dbtest.Get(t)
	h.TruncateAll(t)

	report := dbtest.InsertReport(t, h.Pool)
	if report.ID == 0 {
		t.Error("expected non-zero report ID")
	}
}
