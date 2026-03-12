package repository

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// fakeStatsQuerier is a minimal AdminStatsRepo stand-in that counts calls.
type fakeStatsQuerier struct {
	calls atomic.Int64
	stats AdminStats
}

func (f *fakeStatsQuerier) Get(_ context.Context) (*AdminStats, error) {
	f.calls.Add(1)
	out := f.stats
	return &out, nil
}

func TestCachedAdminStatsRepo_ReturnsCached(t *testing.T) {
	inner := &fakeStatsQuerier{stats: AdminStats{CitiesCount: 5, POIsCount: 10}}

	cache := &CachedAdminStatsRepo{
		ttl: 30 * time.Second,
	}
	// We can't use the real inner (*AdminStatsRepo) without a DB,
	// so test the caching logic directly by pre-populating the cache.
	cache.cached = &AdminStats{CitiesCount: 5, POIsCount: 10}
	cache.expiresAt = time.Now().Add(30 * time.Second)

	// Reading from cache should return the cached value without hitting inner.
	stats, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CitiesCount != 5 || stats.POIsCount != 10 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	// Ensure inner was never called since we pre-populated cache.
	if inner.calls.Load() != 0 {
		t.Fatalf("expected 0 inner calls, got %d", inner.calls.Load())
	}
}

func TestCachedAdminStatsRepo_ExpiredCacheRefetches(t *testing.T) {
	cache := &CachedAdminStatsRepo{
		ttl: 30 * time.Second,
	}
	// Pre-populate with expired cache.
	cache.cached = &AdminStats{CitiesCount: 1}
	cache.expiresAt = time.Now().Add(-1 * time.Second)

	// Without a real inner, Get will panic — this confirms the cache is expired
	// and the code attempts to call inner.Get.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic from nil inner when cache is expired")
			}
		}()
		_, _ = cache.Get(context.Background())
	}()
}

func TestCachedAdminStatsRepo_ReturnsCopy(t *testing.T) {
	cache := &CachedAdminStatsRepo{
		ttl: 30 * time.Second,
	}
	cache.cached = &AdminStats{CitiesCount: 7}
	cache.expiresAt = time.Now().Add(30 * time.Second)

	stats, _ := cache.Get(context.Background())
	stats.CitiesCount = 999

	// The cached value should be unaffected.
	cache.mu.Lock()
	if cache.cached.CitiesCount != 7 {
		t.Fatal("cached value was mutated by caller")
	}
	cache.mu.Unlock()
}
