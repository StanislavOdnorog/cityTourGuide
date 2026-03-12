package domain

import "time"

// ReportType represents the kind of issue being reported.
type ReportType string

const (
	ReportTypeWrongLocation        ReportType = "wrong_location"
	ReportTypeWrongFact            ReportType = "wrong_fact"
	ReportTypeInappropriateContent ReportType = "inappropriate_content"
)

// ReportStatus represents the resolution status of a report.
type ReportStatus string

const (
	ReportStatusNew       ReportStatus = "new"
	ReportStatusReviewed  ReportStatus = "reviewed"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

// Report represents a user-submitted report about a story.
type Report struct {
	ID         int          `json:"id"`
	StoryID    int          `json:"story_id"`
	UserID     string       `json:"user_id"` // UUID
	Type       ReportType   `json:"type"`
	Comment    *string      `json:"comment"`
	UserLat    *float64     `json:"user_lat"`
	UserLng    *float64     `json:"user_lng"`
	Status     ReportStatus `json:"status"`
	ResolvedAt *time.Time   `json:"resolved_at"`
	CreatedAt  time.Time    `json:"created_at"`
}

// AdminReportListItem extends Report with joined POI and story context
// for the admin report listing endpoint.
type AdminReportListItem struct {
	Report
	POIID         *int    `json:"poi_id"`
	POIName       *string `json:"poi_name"`
	StoryLanguage *string `json:"story_language"`
	StoryStatus   *string `json:"story_status"`
}

// ModeratedStory contains story metadata returned from atomic report moderation.
type ModeratedStory struct {
	ID       int         `json:"id"`
	POIID    int         `json:"poi_id"`
	Language string      `json:"language"`
	Status   StoryStatus `json:"status"`
}

// ModeratedReportResult contains the report and story state after moderation.
type ModeratedReportResult struct {
	Report Report         `json:"report"`
	Story  ModeratedStory `json:"story"`
}
