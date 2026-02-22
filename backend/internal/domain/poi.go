package domain

import (
	"encoding/json"
	"time"
)

// POIType represents the type of a Point of Interest.
type POIType string

const (
	POITypeBuilding POIType = "building"
	POITypeStreet   POIType = "street"
	POITypePark     POIType = "park"
	POITypeMonument POIType = "monument"
	POITypeChurch   POIType = "church"
	POITypeBridge   POIType = "bridge"
	POITypeSquare   POIType = "square"
	POITypeMuseum   POIType = "museum"
	POITypeDistrict POIType = "district"
	POITypeOther    POIType = "other"
)

// POIStatus represents the moderation status of a POI.
type POIStatus string

const (
	POIStatusActive        POIStatus = "active"
	POIStatusDisabled      POIStatus = "disabled"
	POIStatusPendingReview POIStatus = "pending_review"
)

// POI represents a Point of Interest with a geographic location.
type POI struct {
	ID            int             `json:"id"`
	CityID        int             `json:"city_id"`
	Name          string          `json:"name"`
	NameRu        *string         `json:"name_ru"`
	Lat           float64         `json:"lat"`
	Lng           float64         `json:"lng"`
	Type          POIType         `json:"type"`
	Tags          json.RawMessage `json:"tags"`
	Address       *string         `json:"address"`
	InterestScore int16           `json:"interest_score"`
	Status        POIStatus       `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
