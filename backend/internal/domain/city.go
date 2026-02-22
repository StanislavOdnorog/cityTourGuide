package domain

import "time"

// City represents a city available in the application.
type City struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	NameRu         *string   `json:"name_ru"`
	Country        string    `json:"country"`
	CenterLat      float64   `json:"center_lat"`
	CenterLng      float64   `json:"center_lng"`
	RadiusKm       float64   `json:"radius_km"`
	IsActive       bool      `json:"is_active"`
	DownloadSizeMB float64   `json:"download_size_mb"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
