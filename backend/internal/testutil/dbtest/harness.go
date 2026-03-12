// Package dbtest provides a reusable integration test harness for PostgreSQL/PostGIS.
//
// It connects to a test database, applies all migrations from the backend/migrations
// directory, and provides helpers for table truncation and fixture creation.
//
// Usage:
//
//	//go:build integration
//
//	func TestMain(m *testing.M) {
//	    dbtest.Main(m)
//	}
//
//	func TestSomething(t *testing.T) {
//	    h := dbtest.Get(t)
//	    // h.Pool is a *pgxpool.Pool ready for use
//	    // h.TruncateAll(t) clears all application tables
//	}
package dbtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDatabaseURL = "postgres://citystories:citystories_secret@localhost:5433/citystories_test?sslmode=disable"

// applicationTables lists all tables in FK-safe truncation order (leaves first).
var applicationTables = []string{
	"admin_audit_logs",
	"push_notifications",
	"device_tokens",
	"inflation_job",
	"report",
	"purchase",
	"user_listening",
	"story",
	"poi",
	"cities",
	"users",
}

// Harness holds a connection pool to a migrated test database.
type Harness struct {
	Pool *pgxpool.Pool
}

var shared *Harness

// Main should be called from TestMain in integration test packages.
// It sets up the shared database harness and runs the test suite.
func Main(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, err := setup(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbtest: setup failed: %v\n", err)
		os.Exit(1)
	}
	shared = h

	code := m.Run()

	h.Pool.Close()
	os.Exit(code)
}

// Get returns the shared harness for the current test.
// It calls t.Fatal if the harness was not initialized (TestMain not wired).
func Get(t *testing.T) *Harness {
	t.Helper()
	if shared == nil {
		t.Fatal("dbtest: harness not initialized — add dbtest.Main(m) to TestMain")
	}
	return shared
}

// TruncateAll removes all rows from application tables.
// Call this at the start of tests that need a clean database state.
func (h *Harness) TruncateAll(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// TRUNCATE CASCADE handles FK ordering, but we list tables explicitly
	// to be safe and avoid touching system tables.
	query := "TRUNCATE TABLE " + strings.Join(applicationTables, ", ") + " CASCADE"
	if _, err := h.Pool.Exec(ctx, query); err != nil {
		t.Fatalf("dbtest: truncate tables: %v", err)
	}
}

func setup(ctx context.Context) (*Harness, error) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	migrationsDir, err := findMigrationsDir()
	if err != nil {
		pool.Close()
		return nil, err
	}

	if err := runMigrations(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &Harness{Pool: pool}, nil
}

func findMigrationsDir() (string, error) {
	// Walk up from this source file to find backend/migrations.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot determine source file location")
	}

	// thisFile is .../backend/internal/testutil/dbtest/harness.go
	// We need .../backend/migrations
	dir := filepath.Dir(thisFile)
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		dir = filepath.Dir(dir)
	}

	return "", fmt.Errorf("cannot find migrations directory from %s", thisFile)
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	sort.Strings(files)

	// Drop and recreate all tables for a clean slate.
	// We use a simple approach: drop schema public cascade and recreate.
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE"); err != nil {
		return fmt.Errorf("drop schema: %w", err)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA public"); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", filepath.Base(f), err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("execute %s: %w", filepath.Base(f), err)
		}
	}

	return nil
}
