package migrations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeDB struct {
	row fakeRow
}

func (db fakeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return db.row
}

type fakeRow struct {
	version uint64
	dirty   bool
	err     error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	*dest[0].(*uint64) = r.version
	*dest[1].(*bool) = r.dirty
	return nil
}

func TestVerify_UpToDateSchema(t *testing.T) {
	migrationsDir := writeMigrationFiles(t, "000001_init.up.sql", "000002_users.up.sql", "000003_stories.up.sql")

	err := Verify(context.Background(), fakeDB{
		row: fakeRow{version: 3},
	}, migrationsDir)
	if err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
}

func TestVerify_OutdatedSchema(t *testing.T) {
	migrationsDir := writeMigrationFiles(t, "000001_init.up.sql", "000002_users.up.sql", "000003_stories.up.sql")

	err := Verify(context.Background(), fakeDB{
		row: fakeRow{version: 2},
	}, migrationsDir)
	if err == nil {
		t.Fatal("Verify() error = nil, want schema mismatch")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version=2") || !strings.Contains(msg, "expected version=3") {
		t.Fatalf("Verify() error = %q, want both current and expected versions", msg)
	}
	if !strings.Contains(msg, "behind") {
		t.Fatalf("Verify() error = %q, want behind message", msg)
	}
}

func TestVerify_DirtySchema(t *testing.T) {
	migrationsDir := writeMigrationFiles(t, "000001_init.up.sql", "000002_users.up.sql", "000003_stories.up.sql")

	err := Verify(context.Background(), fakeDB{
		row: fakeRow{version: 2, dirty: true},
	}, migrationsDir)
	if err == nil {
		t.Fatal("Verify() error = nil, want dirty schema failure")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version=2 (dirty)") || !strings.Contains(msg, "expected version=3") {
		t.Fatalf("Verify() error = %q, want both current and expected versions", msg)
	}
	if !strings.Contains(msg, "dirty") {
		t.Fatalf("Verify() error = %q, want dirty message", msg)
	}
}

func TestVerify_AheadSchema(t *testing.T) {
	migrationsDir := writeMigrationFiles(t, "000001_init.up.sql", "000002_users.up.sql", "000003_stories.up.sql")

	err := Verify(context.Background(), fakeDB{
		row: fakeRow{version: 4},
	}, migrationsDir)
	if err == nil {
		t.Fatal("Verify() error = nil, want ahead schema failure")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version=4") || !strings.Contains(msg, "expected version=3") {
		t.Fatalf("Verify() error = %q, want both current and expected versions", msg)
	}
	if !strings.Contains(msg, "ahead") {
		t.Fatalf("Verify() error = %q, want ahead message", msg)
	}
}

func TestVerify_MissingMigrationTable(t *testing.T) {
	migrationsDir := writeMigrationFiles(t, "000001_init.up.sql")

	err := Verify(context.Background(), fakeDB{
		row: fakeRow{err: &pgconn.PgError{Code: "42P01"}},
	}, migrationsDir)
	if err == nil {
		t.Fatal("Verify() error = nil, want missing table failure")
	}

	msg := err.Error()
	if !strings.Contains(msg, "database version=unknown") || !strings.Contains(msg, "expected version=1") {
		t.Fatalf("Verify() error = %q, want both current and expected versions", msg)
	}
	if !strings.Contains(msg, "missing") {
		t.Fatalf("Verify() error = %q, want missing table message", msg)
	}
}

func TestResolveDir_FindsBackendMigrationsFromRepoRoot(t *testing.T) {
	dir, err := ResolveDir()
	if err != nil {
		t.Fatalf("ResolveDir() error = %v", err)
	}

	if filepath.Base(dir) != "migrations" {
		t.Fatalf("ResolveDir() = %q, want a migrations directory", dir)
	}
}

func writeMigrationFiles(t *testing.T, names ...string) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- test migration\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	return dir
}
