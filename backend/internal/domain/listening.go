package domain

import "time"

// UserListening represents a record of a user listening to a story.
type UserListening struct {
	ID         int       `json:"id"`
	UserID     string    `json:"user_id"` // UUID
	StoryID    int       `json:"story_id"`
	ListenedAt time.Time `json:"listened_at"`
	Completed  bool      `json:"completed"`
	Lat        *float64  `json:"lat"`
	Lng        *float64  `json:"lng"`
}
