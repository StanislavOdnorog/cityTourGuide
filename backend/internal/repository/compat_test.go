//go:build integration

package repository_test

import (
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/testutil/dbtest"
)

// setupTestPool bridges existing integration tests to the shared dbtest harness.
// Close() calls are safe because TestPool.Close() on a shared pool is handled
// by the harness owning the connection lifecycle via TestMain.
//
// New tests should use dbtest.Get(t) directly instead.
func setupTestPool(t *testing.T) *repository.TestPool {
	t.Helper()
	h := dbtest.Get(t)
	return &repository.TestPool{Pool: h.Pool, Shared: true}
}
