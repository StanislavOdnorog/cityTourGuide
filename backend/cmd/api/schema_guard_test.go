package main

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/saas/city-stories-guide/backend/internal/migrations"
)

type guardTestDB struct {
	row guardTestRow
}

func (db guardTestDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return db.row
}

type guardTestRow struct {
	version uint64
	dirty   bool
	err     error
}

func (r guardTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	*dest[0].(*uint64) = r.version
	*dest[1].(*bool) = r.dirty
	return nil
}

func TestVerifyDatabaseSchema_SucceedsWhenCurrent(t *testing.T) {
	expectedVersion := repoMigrationVersion(t)

	err := verifyDatabaseSchema(context.Background(), guardTestDB{
		row: guardTestRow{version: expectedVersion},
	})
	if err != nil {
		t.Fatalf("verifyDatabaseSchema() error = %v, want nil", err)
	}
}

func TestVerifyDatabaseSchema_FailsWhenBehind(t *testing.T) {
	expectedVersion := repoMigrationVersion(t)

	err := verifyDatabaseSchema(context.Background(), guardTestDB{
		row: guardTestRow{version: expectedVersion - 1},
	})
	if err == nil {
		t.Fatal("verifyDatabaseSchema() error = nil, want failure")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version="+itoa(expectedVersion-1)) || !strings.Contains(msg, "expected version="+itoa(expectedVersion)) {
		t.Fatalf("verifyDatabaseSchema() error = %q, want version details", msg)
	}
}

func TestVerifyDatabaseSchema_FailsWhenDirty(t *testing.T) {
	expectedVersion := repoMigrationVersion(t)

	err := verifyDatabaseSchema(context.Background(), guardTestDB{
		row: guardTestRow{version: expectedVersion - 1, dirty: true},
	})
	if err == nil {
		t.Fatal("verifyDatabaseSchema() error = nil, want failure")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version="+itoa(expectedVersion-1)+" (dirty)") || !strings.Contains(msg, "expected version="+itoa(expectedVersion)) {
		t.Fatalf("verifyDatabaseSchema() error = %q, want version details", msg)
	}
}

func TestVerifyDatabaseSchema_FailsWhenMigrationTableMissing(t *testing.T) {
	expectedVersion := repoMigrationVersion(t)

	err := verifyDatabaseSchema(context.Background(), guardTestDB{
		row: guardTestRow{err: &pgconn.PgError{Code: "42P01"}},
	})
	if err == nil {
		t.Fatal("verifyDatabaseSchema() error = nil, want failure")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version=unknown") || !strings.Contains(msg, "expected version="+itoa(expectedVersion)) {
		t.Fatalf("verifyDatabaseSchema() error = %q, want version details", msg)
	}
}

func repoMigrationVersion(t *testing.T) uint64 {
	t.Helper()

	migrationsDir, err := migrations.ResolveDir()
	if err != nil {
		t.Fatalf("ResolveDir() error = %v", err)
	}

	version, err := migrations.LatestVersionFromDir(migrationsDir)
	if err != nil {
		t.Fatalf("LatestVersionFromDir() error = %v", err)
	}

	return version
}

func itoa(v uint64) string {
	return strconv.FormatUint(v, 10)
}
