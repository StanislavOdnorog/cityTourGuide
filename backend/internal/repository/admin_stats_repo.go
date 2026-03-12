package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminStats contains aggregate counters for the admin dashboard.
type AdminStats struct {
	CitiesCount     int `json:"cities_count"`
	POIsCount       int `json:"pois_count"`
	StoriesCount    int `json:"stories_count"`
	ReportsCount    int `json:"reports_count"`
	NewReportsCount int `json:"new_reports_count"`
}

// AdminStatsRepo loads aggregate admin dashboard counters.
type AdminStatsRepo struct {
	pool *pgxpool.Pool
}

// NewAdminStatsRepo creates a new AdminStatsRepo.
func NewAdminStatsRepo(pool *pgxpool.Pool) *AdminStatsRepo {
	return &AdminStatsRepo{pool: pool}
}

// Get returns aggregate admin dashboard counters in a single query.
func (r *AdminStatsRepo) Get(ctx context.Context) (*AdminStats, error) {
	const query = `
		SELECT
			(SELECT COUNT(*) FROM cities WHERE deleted_at IS NULL) AS cities_count,
			(SELECT COUNT(*) FROM poi) AS pois_count,
			(SELECT COUNT(*) FROM story) AS stories_count,
			(SELECT COUNT(*) FROM report) AS reports_count,
			(SELECT COUNT(*) FROM report WHERE status = 'new') AS new_reports_count`

	var stats AdminStats
	if err := r.pool.QueryRow(ctx, query).Scan(
		&stats.CitiesCount,
		&stats.POIsCount,
		&stats.StoriesCount,
		&stats.ReportsCount,
		&stats.NewReportsCount,
	); err != nil {
		return nil, fmt.Errorf("admin_stats_repo: get: %w", err)
	}

	return &stats, nil
}
