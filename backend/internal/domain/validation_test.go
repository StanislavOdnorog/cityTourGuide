package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateUUID(t *testing.T) {
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
	}
	for _, s := range valid {
		if err := ValidateUUID(s); err != nil {
			t.Errorf("ValidateUUID(%q) = %v, want nil", s, err)
		}
	}

	invalid := []string{
		"",
		"not-a-uuid",
		"550E8400-E29B-41D4-A716-446655440000", // uppercase
		"550e8400e29b41d4a716446655440000",       // no dashes
		"550e8400-e29b-41d4-a716-44665544000",    // too short
	}
	for _, s := range invalid {
		err := ValidateUUID(s)
		if err == nil {
			t.Errorf("ValidateUUID(%q) = nil, want error", s)
			continue
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("ValidateUUID(%q) error type = %T, want *ValidationError", s, err)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	if err := ValidateEmail("user@example.com", 254); err != nil {
		t.Errorf("ValidateEmail valid = %v", err)
	}

	tests := []struct {
		email  string
		maxLen int
		field  string
		substr string
	}{
		{"not-an-email", 254, "email", "valid email"},
		{"a@b.c", 3, "email", "exceed"},
		{"has\x00null@example.com", 254, "email", "null bytes"},
	}
	for _, tt := range tests {
		err := ValidateEmail(tt.email, tt.maxLen)
		if err == nil {
			t.Errorf("ValidateEmail(%q, %d) = nil, want error", tt.email, tt.maxLen)
			continue
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("error type = %T, want *ValidationError", err)
			continue
		}
		if ve.Field != tt.field {
			t.Errorf("field = %q, want %q", ve.Field, tt.field)
		}
		if !strings.Contains(ve.Message, tt.substr) {
			t.Errorf("message = %q, want substring %q", ve.Message, tt.substr)
		}
	}
}

func TestValidateStringLength(t *testing.T) {
	// Valid
	if err := ValidateStringLength("hello", "name", 1, 10); err != nil {
		t.Errorf("valid string = %v", err)
	}

	// Unicode: 4 runes
	if err := ValidateStringLength("café", "name", 1, 4); err != nil {
		t.Errorf("unicode string = %v", err)
	}

	// Too short
	err := ValidateStringLength("", "name", 1, 10)
	if err == nil {
		t.Fatal("empty string should fail min=1")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "name" {
		t.Errorf("unexpected error: %v", err)
	}

	// Too long
	err = ValidateStringLength("toolong", "name", 1, 3)
	if err == nil {
		t.Fatal("long string should fail max=3")
	}

	// Null byte
	err = ValidateStringLength("has\x00null", "name", 1, 100)
	if err == nil {
		t.Fatal("null byte should fail")
	}
}

func TestValidateCoordinate(t *testing.T) {
	if err := ValidateCoordinate(48.8566, 2.3522); err != nil {
		t.Errorf("Paris coords = %v", err)
	}

	// Boundary values
	if err := ValidateCoordinate(90, 180); err != nil {
		t.Errorf("max boundary = %v", err)
	}
	if err := ValidateCoordinate(-90, -180); err != nil {
		t.Errorf("min boundary = %v", err)
	}

	// Out of range
	err := ValidateCoordinate(91, 0)
	if err == nil {
		t.Fatal("lat 91 should fail")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "latitude" {
		t.Errorf("unexpected error: %v", err)
	}

	err = ValidateCoordinate(0, 181)
	if err == nil {
		t.Fatal("lng 181 should fail")
	}
	if !errors.As(err, &ve) || ve.Field != "longitude" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateEnum(t *testing.T) {
	allowed := []string{"draft", "published", "archived"}

	if err := ValidateEnum("published", "status", allowed); err != nil {
		t.Errorf("valid enum = %v", err)
	}

	err := ValidateEnum("deleted", "status", allowed)
	if err == nil {
		t.Fatal("invalid enum should fail")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "status" {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(ve.Message, "draft") {
		t.Errorf("message should list allowed values: %q", ve.Message)
	}
}

func TestRejectNullBytes(t *testing.T) {
	if err := RejectNullBytes("clean", "field"); err != nil {
		t.Errorf("clean string = %v", err)
	}

	err := RejectNullBytes("has\x00null", "field")
	if err == nil {
		t.Fatal("null byte should fail")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "field" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidationErrorMessage(t *testing.T) {
	ve := &ValidationError{Field: "name", Message: "is required"}
	if ve.Error() != "name: is required" {
		t.Errorf("Error() = %q", ve.Error())
	}
}
