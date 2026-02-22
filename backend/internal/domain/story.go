package domain

import (
	"encoding/json"
	"time"
)

// StoryLayerType represents the narrative layer of a story.
type StoryLayerType string

const (
	StoryLayerAtmosphere   StoryLayerType = "atmosphere"
	StoryLayerHumanStory   StoryLayerType = "human_story"
	StoryLayerHiddenDetail StoryLayerType = "hidden_detail"
	StoryLayerTimeShift    StoryLayerType = "time_shift"
	StoryLayerGeneral      StoryLayerType = "general"
)

// StoryStatus represents the moderation status of a story.
type StoryStatus string

const (
	StoryStatusActive        StoryStatus = "active"
	StoryStatusDisabled      StoryStatus = "disabled"
	StoryStatusReported      StoryStatus = "reported"
	StoryStatusPendingReview StoryStatus = "pending_review"
)

// Story represents an audio story linked to a POI.
type Story struct {
	ID          int             `json:"id"`
	POIID       int             `json:"poi_id"`
	Language    string          `json:"language"`
	Text        string          `json:"text"`
	AudioURL    *string         `json:"audio_url"`
	DurationSec *int16          `json:"duration_sec"`
	LayerType   StoryLayerType  `json:"layer_type"`
	OrderIndex  int16           `json:"order_index"`
	IsInflation bool            `json:"is_inflation"`
	Confidence  int16           `json:"confidence"`
	Sources     json.RawMessage `json:"sources"`
	Status      StoryStatus     `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
