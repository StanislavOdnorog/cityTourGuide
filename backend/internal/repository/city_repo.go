package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// CityRepo handles database operations for cities.
type CityRepo struct {
	pool *pgxpool.Pool
}

// NewCityRepo creates a new CityRepo.
func NewCityRepo(pool *pgxpool.Pool) *CityRepo {
	return &CityRepo{pool: pool}
}

// Create inserts a new city and returns it with generated fields.
func (r *CityRepo) Create(ctx context.Context, city *domain.City) (*domain.City, error) {
	query := `
		INSERT INTO cities (name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb, created_at, updated_at`

	var c domain.City
	err := r.pool.QueryRow(ctx, query,
		city.Name,
		city.NameRu,
		city.Country,
		city.CenterLat,
		city.CenterLng,
		city.RadiusKm,
		city.IsActive,
		city.DownloadSizeMB,
	).Scan(
		&c.ID,
		&c.Name,
		&c.NameRu,
		&c.Country,
		&c.CenterLat,
		&c.CenterLng,
		&c.RadiusKm,
		&c.IsActive,
		&c.DownloadSizeMB,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("city_repo: create: %w", err)
	}

	return &c, nil
}

// GetByID returns a city by its ID.
func (r *CityRepo) GetByID(ctx context.Context, id int) (*domain.City, error) {
	query := `
		SELECT id, name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb, created_at, updated_at
		FROM cities
		WHERE id = $1`

	var c domain.City
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.NameRu,
		&c.Country,
		&c.CenterLat,
		&c.CenterLng,
		&c.RadiusKm,
		&c.IsActive,
		&c.DownloadSizeMB,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("city_repo: get by id: %w", err)
	}

	return &c, nil
}

// GetAll returns all cities ordered by name.
func (r *CityRepo) GetAll(ctx context.Context) ([]domain.City, error) {
	query := `
		SELECT id, name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb, created_at, updated_at
		FROM cities
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("city_repo: get all: %w", err)
	}
	defer rows.Close()

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
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("city_repo: get all scan: %w", err)
		}
		cities = append(cities, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("city_repo: get all rows: %w", err)
	}

	return cities, nil
}

// Update modifies an existing city and returns the updated record.
func (r *CityRepo) Update(ctx context.Context, city *domain.City) (*domain.City, error) {
	query := `
		UPDATE cities
		SET name = $2, name_ru = $3, country = $4, center_lat = $5, center_lng = $6,
		    radius_km = $7, is_active = $8, download_size_mb = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb, created_at, updated_at`

	var c domain.City
	err := r.pool.QueryRow(ctx, query,
		city.ID,
		city.Name,
		city.NameRu,
		city.Country,
		city.CenterLat,
		city.CenterLng,
		city.RadiusKm,
		city.IsActive,
		city.DownloadSizeMB,
	).Scan(
		&c.ID,
		&c.Name,
		&c.NameRu,
		&c.Country,
		&c.CenterLat,
		&c.CenterLng,
		&c.RadiusKm,
		&c.IsActive,
		&c.DownloadSizeMB,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("city_repo: update: %w", err)
	}

	return &c, nil
}

// Delete removes a city by its ID.
func (r *CityRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM cities WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("city_repo: delete: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
