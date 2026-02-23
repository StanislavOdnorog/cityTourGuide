package domain

import "time"

// AuthProvider represents the authentication method used by a user.
type AuthProvider string

const (
	AuthProviderEmail  AuthProvider = "email"
	AuthProviderGoogle AuthProvider = "google"
	AuthProviderApple  AuthProvider = "apple"
)

// User represents an application user.
type User struct {
	ID                  string       `json:"id"` // UUID
	Email               *string      `json:"email"`
	Name                *string      `json:"name"`
	PasswordHash        *string      `json:"-"` // never expose in JSON
	AuthProvider        AuthProvider `json:"auth_provider"`
	ProviderID          *string      `json:"provider_id,omitempty"` // OAuth provider user ID (Google sub, Apple sub)
	LanguagePref        string       `json:"language_pref"`
	IsAnonymous         bool         `json:"is_anonymous"`
	IsAdmin             bool         `json:"is_admin"`
	DeletedAt           *time.Time   `json:"deleted_at,omitempty"`
	DeletionScheduledAt *time.Time   `json:"deletion_scheduled_at,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// IsScheduledForDeletion returns true if the account is marked for deletion.
func (u *User) IsScheduledForDeletion() bool {
	return u.DeletionScheduledAt != nil && u.DeletedAt == nil
}
