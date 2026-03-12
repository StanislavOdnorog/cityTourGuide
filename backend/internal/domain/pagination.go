package domain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidCursor is returned when a pagination cursor cannot be decoded.
var ErrInvalidCursor = errors.New("invalid cursor")

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// PageRequest holds cursor-based pagination parameters.
type PageRequest struct {
	Cursor string
	Limit  int
}

// PageResponse holds a paginated result set.
type PageResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// NormalizeLimit ensures the limit is within valid bounds.
// Returns an error if limit exceeds MaxPageLimit.
func (p *PageRequest) NormalizeLimit() error {
	if p.Limit <= 0 {
		p.Limit = DefaultPageLimit
		return nil
	}
	if p.Limit > MaxPageLimit {
		return fmt.Errorf("limit must not exceed %d", MaxPageLimit)
	}
	return nil
}

// EncodeCursor encodes an integer ID as an opaque base64 cursor.
func EncodeCursor(id int) string {
	return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("id:%d", id)))
}

// DecodeCursor decodes an opaque base64 cursor back to an integer ID.
func DecodeCursor(cursor string) (int, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("malformed encoding: %w", ErrInvalidCursor)
	}
	s := string(data)
	if !strings.HasPrefix(s, "id:") {
		return 0, fmt.Errorf("unexpected format: %w", ErrInvalidCursor)
	}
	id, err := strconv.Atoi(s[3:])
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("bad id value: %w", ErrInvalidCursor)
	}
	return id, nil
}

// EncodeCursor64 encodes an int64 ID as an opaque base64 cursor.
func EncodeCursor64(id int64) string {
	return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("id:%d", id)))
}

// DecodeCursor64 decodes an opaque base64 cursor back to an int64 ID.
func DecodeCursor64(cursor string) (int64, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("malformed encoding: %w", ErrInvalidCursor)
	}
	s := string(data)
	if !strings.HasPrefix(s, "id:") {
		return 0, fmt.Errorf("unexpected format: %w", ErrInvalidCursor)
	}
	id, err := strconv.ParseInt(s[3:], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("bad id value: %w", ErrInvalidCursor)
	}
	return id, nil
}

// EncodeDistanceCursor encodes a (distance, id) pair as an opaque base64 cursor
// for distance-ordered pagination.
func EncodeDistanceCursor(distance float64, id int) string {
	return base64.URLEncoding.EncodeToString(
		[]byte(fmt.Sprintf("dist:%f:%d", distance, id)),
	)
}

// DecodeDistanceCursor decodes a distance-based cursor back to (distance, id).
func DecodeDistanceCursor(cursor string) (float64, int, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, 0, fmt.Errorf("malformed encoding: %w", ErrInvalidCursor)
	}
	s := string(data)
	if !strings.HasPrefix(s, "dist:") {
		return 0, 0, fmt.Errorf("unexpected format: %w", ErrInvalidCursor)
	}
	parts := strings.SplitN(s[5:], ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected format: %w", ErrInvalidCursor)
	}
	distance, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("bad distance value: %w", ErrInvalidCursor)
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, 0, fmt.Errorf("bad id value: %w", ErrInvalidCursor)
	}
	return distance, id, nil
}
