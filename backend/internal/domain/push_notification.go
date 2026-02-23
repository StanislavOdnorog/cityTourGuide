package domain

import "time"

// NotificationType represents the type of push notification.
type NotificationType string

const (
	NotificationTypeGeo     NotificationType = "geo"
	NotificationTypeContent NotificationType = "content"
)

// PushNotification represents a sent push notification record.
type PushNotification struct {
	ID            int              `json:"id"`
	UserID        string           `json:"user_id"`
	DeviceTokenID int              `json:"device_token_id"`
	Type          NotificationType `json:"type"`
	Title         string           `json:"title"`
	Body          string           `json:"body"`
	Data          map[string]any   `json:"data,omitempty"`
	SentAt        *time.Time       `json:"sent_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

// NotificationPreferences holds a user's notification preferences.
type NotificationPreferences struct {
	GeoEnabled     bool `json:"geo_enabled"`
	ContentEnabled bool `json:"content_enabled"`
}
