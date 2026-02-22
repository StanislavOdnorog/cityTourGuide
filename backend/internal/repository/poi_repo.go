package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// NearbyPOI extends POI with distance information from a spatial query.
type NearbyPOI struct {
	domain.POI
	DistanceM float64 `json:"distance_m"`
}

// POIRepo handles database operations for Points of Interest.
type POIRepo struct {
	pool *pgxpool.Pool
}

// NewPOIRepo creates a new POIRepo.
func NewPOIRepo(pool *pgxpool.Pool) *POIRepo {
	return &POIRepo{pool: pool}
}

// Create inserts a new POI using ST_MakePoint for the geography column.
func (r *POIRepo) Create(ctx context.Context, poi *domain.POI) (*domain.POI, error) {
	query := `
		INSERT INTO poi (city_id, name, name_ru, location, type, tags, address, interest_score, status)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, $6, $7, $8, $9, $10)
		RETURNING id, city_id, name, name_ru,
			ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng,
			type, tags, address, interest_score, status, created_at, updated_at`

	var p domain.POI
	err := r.pool.QueryRow(ctx, query,
		poi.CityID,
		poi.Name,
		poi.NameRu,
		poi.Lng, // ST_MakePoint takes (lng, lat)
		poi.Lat,
		poi.Type,
		poi.Tags,
		poi.Address,
		poi.InterestScore,
		poi.Status,
	).Scan(
		&p.ID, &p.CityID, &p.Name, &p.NameRu,
		&p.Lat, &p.Lng,
		&p.Type, &p.Tags, &p.Address, &p.InterestScore, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("poi_repo: create: %w", err)
	}

	return &p, nil
}

// GetByID returns a POI by its ID.
func (r *POIRepo) GetByID(ctx context.Context, id int) (*domain.POI, error) {
	query := `
		SELECT id, city_id, name, name_ru,
			ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng,
			type, tags, address, interest_score, status, created_at, updated_at
		FROM poi
		WHERE id = $1`

	var p domain.POI
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CityID, &p.Name, &p.NameRu,
		&p.Lat, &p.Lng,
		&p.Type, &p.Tags, &p.Address, &p.InterestScore, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("poi_repo: get by id: %w", err)
	}

	return &p, nil
}

// GetByCityID returns POIs for a given city with optional status and type filters.
func (r *POIRepo) GetByCityID(ctx context.Context, cityID int, status *domain.POIStatus, poiType *domain.POIType) ([]domain.POI, error) {
	query := `
		SELECT id, city_id, name, name_ru,
			ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng,
			type, tags, address, interest_score, status, created_at, updated_at
		FROM poi
		WHERE city_id = $1`

	args := []interface{}{cityID}
	argIdx := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}
	if poiType != nil {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, *poiType)
	}

	query += " ORDER BY interest_score DESC, name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("poi_repo: get by city id: %w", err)
	}
	defer rows.Close()

	var pois []domain.POI
	for rows.Next() {
		var p domain.POI
		if err := rows.Scan(
			&p.ID, &p.CityID, &p.Name, &p.NameRu,
			&p.Lat, &p.Lng,
			&p.Type, &p.Tags, &p.Address, &p.InterestScore, &p.Status,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("poi_repo: get by city id scan: %w", err)
		}
		pois = append(pois, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("poi_repo: get by city id rows: %w", err)
	}

	return pois, nil
}

// Update modifies an existing POI and returns the updated record.
func (r *POIRepo) Update(ctx context.Context, poi *domain.POI) (*domain.POI, error) {
	query := `
		UPDATE poi
		SET city_id = $2, name = $3, name_ru = $4,
			location = ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography,
			type = $7, tags = $8, address = $9, interest_score = $10, status = $11,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, city_id, name, name_ru,
			ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng,
			type, tags, address, interest_score, status, created_at, updated_at`

	var p domain.POI
	err := r.pool.QueryRow(ctx, query,
		poi.ID,
		poi.CityID,
		poi.Name,
		poi.NameRu,
		poi.Lng, // ST_MakePoint takes (lng, lat)
		poi.Lat,
		poi.Type,
		poi.Tags,
		poi.Address,
		poi.InterestScore,
		poi.Status,
	).Scan(
		&p.ID, &p.CityID, &p.Name, &p.NameRu,
		&p.Lat, &p.Lng,
		&p.Type, &p.Tags, &p.Address, &p.InterestScore, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("poi_repo: update: %w", err)
	}

	return &p, nil
}

// Delete removes a POI by its ID.
func (r *POIRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM poi WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("poi_repo: delete: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// FindNearby returns POIs within a given radius of a point, joined with active stories
// for the specified language. Results are sorted by interest_score DESC, distance ASC.
func (r *POIRepo) FindNearby(ctx context.Context, lat, lng, radiusM float64, cityID int, language string) ([]NearbyPOI, error) {
	query := `
		SELECT DISTINCT ON (p.id)
			p.id, p.city_id, p.name, p.name_ru,
			ST_Y(p.location::geometry) AS lat, ST_X(p.location::geometry) AS lng,
			p.type, p.tags, p.address, p.interest_score, p.status,
			p.created_at, p.updated_at,
			ST_Distance(p.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_m
		FROM poi p
		INNER JOIN story s ON s.poi_id = p.id AND s.status = 'active' AND s.language = $5
		WHERE p.status = 'active'
			AND p.city_id = $4
			AND ST_DWithin(p.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY p.id, p.interest_score DESC, distance_m ASC`

	// Wrap with an outer query to apply the final sort order after DISTINCT ON
	wrappedQuery := `
		SELECT id, city_id, name, name_ru, lat, lng, type, tags, address,
			interest_score, status, created_at, updated_at, distance_m
		FROM (` + query + `) sub
		ORDER BY interest_score DESC, distance_m ASC
		LIMIT 20`

	rows, err := r.pool.Query(ctx, wrappedQuery, lng, lat, radiusM, cityID, language)
	if err != nil {
		return nil, fmt.Errorf("poi_repo: find nearby: %w", err)
	}
	defer rows.Close()

	var results []NearbyPOI
	for rows.Next() {
		var np NearbyPOI
		if err := rows.Scan(
			&np.ID, &np.CityID, &np.Name, &np.NameRu,
			&np.Lat, &np.Lng,
			&np.Type, &np.Tags, &np.Address, &np.InterestScore, &np.Status,
			&np.CreatedAt, &np.UpdatedAt,
			&np.DistanceM,
		); err != nil {
			return nil, fmt.Errorf("poi_repo: find nearby scan: %w", err)
		}
		results = append(results, np)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("poi_repo: find nearby rows: %w", err)
	}

	return results, nil
}
