package domain

import "time"

// DevicePlatform represents the mobile platform of a device.
type DevicePlatform string

const (
	DevicePlatformIOS     DevicePlatform = "ios"
	DevicePlatformAndroid DevicePlatform = "android"
)

// DeviceToken represents a push notification token for a user's device.
type DeviceToken struct {
	ID        int            `json:"id"`
	UserID    string         `json:"user_id"`
	Token     string         `json:"token"`
	Platform  DevicePlatform `json:"platform"`
	IsActive  bool           `json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
