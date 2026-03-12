package repository

import (
	"context"
	"sync"
	"time"
)

// CachedAdminStatsRepo wraps an AdminStatsRepo with a simple in-memory TTL cache
// to avoid repeated full-table COUNT scans on every dashboard page load.
type CachedAdminStatsRepo struct {
	inner *AdminStatsRepo
	ttl   time.Duration

	mu        sync.Mutex
	cached    *AdminStats
	expiresAt time.Time
}

// NewCachedAdminStatsRepo creates a caching wrapper around an AdminStatsRepo.
func NewCachedAdminStatsRepo(inner *AdminStatsRepo, ttl time.Duration) *CachedAdminStatsRepo {
	return &CachedAdminStatsRepo{
		inner: inner,
		ttl:   ttl,
	}
}

// Get returns cached stats if still valid, otherwise fetches fresh data.
func (c *CachedAdminStatsRepo) Get(ctx context.Context) (*AdminStats, error) {
	c.mu.Lock()
	if c.cached != nil && time.Now().Before(c.expiresAt) {
		stats := *c.cached
		c.mu.Unlock()
		return &stats, nil
	}
	c.mu.Unlock()

	stats, err := c.inner.Get(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cached = stats
	c.expiresAt = time.Now().Add(c.ttl)
	c.mu.Unlock()

	// Return a copy so callers can't mutate the cached value.
	out := *stats
	return &out, nil
}
