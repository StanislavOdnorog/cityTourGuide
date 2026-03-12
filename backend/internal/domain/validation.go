package domain

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a structured validation failure suitable for API responses.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// uuidRegex matches a canonical UUID v4 string (lowercase hex).
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidateUUID checks that s is a valid lowercase UUID string.
func ValidateUUID(s string) error {
	if !uuidRegex.MatchString(s) {
		return &ValidationError{Field: "id", Message: "must be a valid UUID"}
	}
	return nil
}

// ValidateEmail checks that s is a valid email address and does not exceed maxLen runes.
func ValidateEmail(s string, maxLen int) error {
	if err := RejectNullBytes(s, "email"); err != nil {
		return err
	}
	if utf8.RuneCountInString(s) > maxLen {
		return &ValidationError{Field: "email", Message: fmt.Sprintf("must not exceed %d characters", maxLen)}
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return &ValidationError{Field: "email", Message: "must be a valid email address"}
	}
	return nil
}

// ValidateStringLength checks that the rune length of s falls within [min, max].
func ValidateStringLength(s, fieldName string, min, max int) error {
	if err := RejectNullBytes(s, fieldName); err != nil {
		return err
	}
	n := utf8.RuneCountInString(s)
	if n < min {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("must be at least %d characters", min)}
	}
	if n > max {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("must not exceed %d characters", max)}
	}
	return nil
}

// ValidateCoordinate checks that lat and lng are within valid geographic ranges.
func ValidateCoordinate(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return &ValidationError{Field: "latitude", Message: "must be between -90 and 90"}
	}
	if lng < -180 || lng > 180 {
		return &ValidationError{Field: "longitude", Message: "must be between -180 and 180"}
	}
	return nil
}

// ValidateEnum checks that value is one of the allowed values.
func ValidateEnum(value, fieldName string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return &ValidationError{
		Field:   fieldName,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	}
}

// RejectNullBytes returns an error if s contains any null bytes.
func RejectNullBytes(s, fieldName string) error {
	if strings.ContainsRune(s, '\x00') {
		return &ValidationError{Field: fieldName, Message: "must not contain null bytes"}
	}
	return nil
}
