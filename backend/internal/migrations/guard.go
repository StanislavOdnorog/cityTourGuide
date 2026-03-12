package migrations

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const metadataTable = "schema_migrations"

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Verify checks that the connected database schema matches the latest checked-in migration.
func Verify(ctx context.Context, db QueryRower, migrationsDir string) error {
	expectedVersion, err := LatestVersionFromDir(migrationsDir)
	if err != nil {
		return err
	}

	currentVersion, dirty, err := readDatabaseVersion(ctx, db)
	if err != nil {
		return schemaStateError{
			message:         err.Error(),
			currentVersion:  "unknown",
			expectedVersion: expectedVersion,
		}
	}

	if dirty {
		return schemaStateError{
			message:         "database schema is marked dirty; fix or force the failed migration before starting",
			currentVersion:  fmt.Sprintf("%d (dirty)", currentVersion),
			expectedVersion: expectedVersion,
		}
	}

	if currentVersion < expectedVersion {
		return schemaStateError{
			message:         "database schema is behind the checked-in migrations; run migrate-up before starting",
			currentVersion:  strconv.FormatUint(currentVersion, 10),
			expectedVersion: expectedVersion,
		}
	}

	if currentVersion > expectedVersion {
		return schemaStateError{
			message:         "database schema is ahead of the checked-in migrations in an unexpected way; confirm the deployed code and migration set match",
			currentVersion:  strconv.FormatUint(currentVersion, 10),
			expectedVersion: expectedVersion,
		}
	}

	return nil
}

// LatestVersionFromDir returns the highest checked-in migration version from *.up.sql files.
func LatestVersionFromDir(migrationsDir string) (uint64, error) {
	migrationFiles, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return 0, fmt.Errorf("migration guard: list migration files in %q: %w", migrationsDir, err)
	}
	if len(migrationFiles) == 0 {
		return 0, fmt.Errorf("migration guard: no up migration files found in %q", migrationsDir)
	}

	var latest uint64
	for _, path := range migrationFiles {
		version, err := parseVersion(filepath.Base(path))
		if err != nil {
			return 0, fmt.Errorf("migration guard: parse migration file %q: %w", path, err)
		}
		if version > latest {
			latest = version
		}
	}

	return latest, nil
}

// ResolveDir finds the checked-in migrations directory in common local and container layouts.
func ResolveDir() (string, error) {
	candidates := []string{
		filepath.Join(".", "migrations"),
		filepath.Join(".", "backend", "migrations"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "migrations"),
			filepath.Join(exeDir, "..", "migrations"),
		)
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		sourceDir := filepath.Dir(sourceFile)
		candidates = append(candidates, filepath.Join(sourceDir, "..", "..", "migrations"))
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("migration guard: stat migrations dir %q: %w", candidate, err)
		}
	}

	return "", fmt.Errorf("migration guard: could not locate migrations directory")
}

func readDatabaseVersion(ctx context.Context, db QueryRower) (uint64, bool, error) {
	var version uint64
	var dirty bool

	err := db.QueryRow(ctx, "SELECT version, dirty FROM "+metadataTable+" LIMIT 1").Scan(&version, &dirty)
	if err == nil {
		return version, dirty, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return 0, false, fmt.Errorf("migration metadata table %q is missing; run migrate-up to initialize the schema", metadataTable)
	}

	return 0, false, fmt.Errorf("read migration metadata from %q: %w", metadataTable, err)
}

func parseVersion(filename string) (uint64, error) {
	versionPart, _, found := strings.Cut(filename, "_")
	if !found {
		return 0, fmt.Errorf("missing version prefix")
	}

	version, err := strconv.ParseUint(versionPart, 10, 64)
	if err != nil {
		return 0, err
	}

	return version, nil
}

type schemaStateError struct {
	message         string
	currentVersion  string
	expectedVersion uint64
}

func (e schemaStateError) Error() string {
	return fmt.Sprintf(
		"migration guard: %s (database version=%s, expected version=%d)",
		e.message,
		e.currentVersion,
		e.expectedVersion,
	)
}
