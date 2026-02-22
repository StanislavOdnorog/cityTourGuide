package domain

import "time"

// PurchaseType represents the kind of purchase.
type PurchaseType string

const (
	PurchaseTypeCityPack     PurchaseType = "city_pack"
	PurchaseTypeSubscription PurchaseType = "subscription"
	PurchaseTypeLifetime     PurchaseType = "lifetime"
)

// Purchase represents a user's in-app purchase record.
type Purchase struct {
	ID            int          `json:"id"`
	UserID        string       `json:"user_id"` // UUID
	Type          PurchaseType `json:"type"`
	CityID        *int         `json:"city_id"`
	Platform      string       `json:"platform"`
	TransactionID *string      `json:"transaction_id"`
	Price         float64      `json:"price"`
	IsLTD         bool         `json:"is_ltd"`
	ExpiresAt     *time.Time   `json:"expires_at"`
	CreatedAt     time.Time    `json:"created_at"`
}
