//go:build integration

package repository_test

import (
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/testutil/dbtest"
)

func TestMain(m *testing.M) {
	dbtest.Main(m)
}
