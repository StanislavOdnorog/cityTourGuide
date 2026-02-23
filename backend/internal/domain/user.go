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
	ID           string       `json:"id"` // UUID
	Email        *string      `json:"email"`
	Name         *string      `json:"name"`
	PasswordHash *string      `json:"-"` // never expose in JSON
	AuthProvider AuthProvider `json:"auth_provider"`
	LanguagePref string       `json:"language_pref"`
	IsAnonymous  bool         `json:"is_anonymous"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
