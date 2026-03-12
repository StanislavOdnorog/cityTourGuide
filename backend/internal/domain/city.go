package domain

import "time"

// City represents a city available in the application.
type City struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	NameRu         *string    `json:"name_ru"`
	Country        string     `json:"country"`
	CenterLat      float64    `json:"center_lat"`
	CenterLng      float64    `json:"center_lng"`
	RadiusKm       float64    `json:"radius_km"`
	IsActive       bool       `json:"is_active"`
	DownloadSizeMB float64    `json:"download_size_mb"`
	DeletedAt      *time.Time `json:"deleted_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// DisplayName returns the localized city name for the given language.
// For "ru", it returns NameRu if non-nil and non-empty, otherwise falls back to Name.
func (c *City) DisplayName(language string) string {
	if language == "ru" && c.NameRu != nil && *c.NameRu != "" {
		return *c.NameRu
	}
	return c.Name
}
