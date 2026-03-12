package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// cityColumns is the standard SELECT list for cities.
const cityColumns = `id, name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb, deleted_at, created_at, updated_at`

// CityRepo handles database operations for cities.
type CityRepo struct {
	pool *pgxpool.Pool
}

// NewCityRepo creates a new CityRepo.
func NewCityRepo(pool *pgxpool.Pool) *CityRepo {
	return &CityRepo{pool: pool}
}

func scanCity(row pgx.Row) (*domain.City, error) {
	var c domain.City
	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.NameRu,
		&c.Country,
		&c.CenterLat,
		&c.CenterLng,
		&c.RadiusKm,
		&c.IsActive,
		&c.DownloadSizeMB,
		&c.DeletedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	return &c, err
}

func scanCities(rows pgx.Rows) ([]domain.City, error) {
	var cities []domain.City
	for rows.Next() {
		var c domain.City
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.NameRu,
			&c.Country,
			&c.CenterLat,
			&c.CenterLng,
			&c.RadiusKm,
			&c.IsActive,
			&c.DownloadSizeMB,
			&c.DeletedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cities = append(cities, c)
	}
	return cities, rows.Err()
}

// Create inserts a new city and returns it with generated fields.
func (r *CityRepo) Create(ctx context.Context, city *domain.City) (*domain.City, error) {
	query := `
		INSERT INTO cities (name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + cityColumns

	c, err := scanCity(r.pool.QueryRow(ctx, query,
		city.Name,
		city.NameRu,
		city.Country,
		city.CenterLat,
		city.CenterLng,
		city.RadiusKm,
		city.IsActive,
		city.DownloadSizeMB,
	))
	if err != nil {
		return nil, fmt.Errorf("city_repo: create: %w", err)
	}

	return c, nil
}

// GetByID returns a city by its ID, optionally including soft-deleted rows.
func (r *CityRepo) GetByID(ctx context.Context, id int, includeDeleted bool) (*domain.City, error) {
	query := `SELECT ` + cityColumns + ` FROM cities WHERE id = $1`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}

	c, err := scanCity(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("city_repo: get by id: %w", err)
	}

	return c, nil
}

// GetActiveByID returns a city by its ID only if it is active and not soft-deleted.
func (r *CityRepo) GetActiveByID(ctx context.Context, id int) (*domain.City, error) {
	query := `SELECT ` + cityColumns + ` FROM cities WHERE id = $1 AND is_active = true AND deleted_at IS NULL`

	c, err := scanCity(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("city_repo: get active by id: %w", err)
	}

	return c, nil
}

// GetAll returns all non-deleted cities ordered by name (unpaginated, for internal use).
func (r *CityRepo) GetAll(ctx context.Context) ([]domain.City, error) {
	query := `SELECT ` + cityColumns + ` FROM cities WHERE deleted_at IS NULL ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("city_repo: get all: %w", err)
	}
	defer rows.Close()

	cities, err := scanCities(rows)
	if err != nil {
		return nil, fmt.Errorf("city_repo: get all scan: %w", err)
	}

	return cities, nil
}

// ListActive returns only active, non-deleted cities with cursor-based pagination, ordered by id ASC.
func (r *CityRepo) ListActive(ctx context.Context, page domain.PageRequest) (*domain.PageResponse[domain.City], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("city_repo: list active: %w", err)
	}

	query := `SELECT ` + cityColumns + ` FROM cities WHERE is_active = true AND deleted_at IS NULL`

	args := []interface{}{}
	argIdx := 1

	if page.Cursor != "" {
		cursorID, err := domain.DecodeCursor(page.Cursor)
		if err != nil {
			return nil, fmt.Errorf("city_repo: list active: %w", err)
		}
		query += fmt.Sprintf(" AND id > $%d", argIdx)
		args = append(args, cursorID)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("city_repo: list active: %w", err)
	}
	defer rows.Close()

	cities, err := scanCities(rows)
	if err != nil {
		return nil, fmt.Errorf("city_repo: list active scan: %w", err)
	}

	hasMore := len(cities) > page.Limit
	if hasMore {
		cities = cities[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(cities) > 0 {
		nextCursor = domain.EncodeCursor(cities[len(cities)-1].ID)
	}

	return &domain.PageResponse[domain.City]{
		Items:      cities,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// List returns cities with cursor-based pagination, ordered by id ASC.
// Soft-deleted cities are included only when includeDeleted is true.
func (r *CityRepo) List(ctx context.Context, page domain.PageRequest, includeDeleted bool, sort ListSort) (*domain.PageResponse[domain.City], error) {
	if err := page.NormalizeLimit(); err != nil {
		return nil, fmt.Errorf("city_repo: list: %w", err)
	}

	resolvedSort, err := ResolveSort(sort, map[string]SortColumn{
		"id":         {Key: "id", Column: "id", Type: SortValueInt},
		"name":       {Key: "name", Column: "name", Type: SortValueString},
		"country":    {Key: "country", Column: "country", Type: SortValueString},
		"is_active":  {Key: "is_active", Column: "is_active", Type: SortValueBool},
		"created_at": {Key: "created_at", Column: "created_at", Type: SortValueTime},
		"updated_at": {Key: "updated_at", Column: "updated_at", Type: SortValueTime},
	}, "id", SortDirAsc)
	if err != nil {
		return nil, fmt.Errorf("city_repo: list: %w", err)
	}

	query := `SELECT ` + cityColumns + ` FROM cities`
	if !includeDeleted {
		query += ` WHERE deleted_at IS NULL`
	}

	args := []interface{}{}
	argIdx := 1

	cursorCondition, cursorArgs, err := resolvedSort.CursorCondition(page.Cursor, argIdx)
	if err != nil {
		return nil, fmt.Errorf("city_repo: list: %w", err)
	}
	if cursorCondition != "" {
		if includeDeleted {
			query += " WHERE " + cursorCondition
		} else {
			query += " AND " + cursorCondition
		}
		args = append(args, cursorArgs...)
		argIdx += len(cursorArgs)
	}

	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d", resolvedSort.OrderBy(), argIdx)
	args = append(args, page.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("city_repo: list: %w", err)
	}
	defer rows.Close()

	cities, err := scanCities(rows)
	if err != nil {
		return nil, fmt.Errorf("city_repo: list scan: %w", err)
	}

	hasMore := len(cities) > page.Limit
	if hasMore {
		cities = cities[:page.Limit]
	}

	var nextCursor string
	if hasMore && len(cities) > 0 {
		nextCursor, err = EncodeOrderedCursor(resolvedSort, citySortValue(cities[len(cities)-1], resolvedSort.Key), cities[len(cities)-1].ID)
		if err != nil {
			return nil, fmt.Errorf("city_repo: list: %w", err)
		}
	}

	return &domain.PageResponse[domain.City]{
		Items:      cities,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func citySortValue(city domain.City, key string) interface{} {
	switch key {
	case "name":
		return city.Name
	case "country":
		return city.Country
	case "is_active":
		return city.IsActive
	case "created_at":
		return city.CreatedAt
	case "updated_at":
		return city.UpdatedAt
	default:
		return city.ID
	}
}

// Update modifies an existing city and returns the updated record.
func (r *CityRepo) Update(ctx context.Context, city *domain.City) (*domain.City, error) {
	query := `
		UPDATE cities
		SET name = $2, name_ru = $3, country = $4, center_lat = $5, center_lng = $6,
		    radius_km = $7, is_active = $8, download_size_mb = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + cityColumns

	c, err := scanCity(r.pool.QueryRow(ctx, query,
		city.ID,
		city.Name,
		city.NameRu,
		city.Country,
		city.CenterLat,
		city.CenterLng,
		city.RadiusKm,
		city.IsActive,
		city.DownloadSizeMB,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("city_repo: update: %w", err)
	}

	return c, nil
}

// Delete soft-deletes a city by setting deleted_at. Idempotent for already-deleted cities.
func (r *CityRepo) Delete(ctx context.Context, id int) error {
	query := `UPDATE cities SET deleted_at = COALESCE(deleted_at, NOW()), updated_at = NOW() WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("city_repo: delete: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Restore clears the soft-delete timestamp, making the city visible again.
func (r *CityRepo) Restore(ctx context.Context, id int) (*domain.City, error) {
	existing, err := r.GetByID(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("city_repo: restore: %w", err)
	}
	if existing.DeletedAt == nil {
		return nil, ErrNotFound
	}

	var conflictingID int
	conflictErr := r.pool.QueryRow(ctx,
		`SELECT id FROM cities WHERE id <> $1 AND name = $2 AND country = $3 AND deleted_at IS NULL LIMIT 1`,
		id,
		existing.Name,
		existing.Country,
	).Scan(&conflictingID)
	if conflictErr == nil {
		return nil, ErrConflict
	}
	if conflictErr != nil && !errors.Is(conflictErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("city_repo: restore conflict check: %w", conflictErr)
	}

	query := `
		UPDATE cities SET deleted_at = NULL, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NOT NULL
		RETURNING ` + cityColumns

	c, err := scanCity(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("city_repo: restore: %w", err)
	}

	return c, nil
}
