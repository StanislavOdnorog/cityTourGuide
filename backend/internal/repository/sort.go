package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	SortDirAsc  = "asc"
	SortDirDesc = "desc"
)

type ListSort struct {
	By  string
	Dir string
}

type SortValueType string

const (
	SortValueInt    SortValueType = "int"
	SortValueInt16  SortValueType = "int16"
	SortValueInt64  SortValueType = "int64"
	SortValueString SortValueType = "string"
	SortValueTime   SortValueType = "time"
	SortValueBool   SortValueType = "bool"
)

type SortColumn struct {
	Key    string
	Column string
	Type   SortValueType
}

type ResolvedSort struct {
	Key    string
	Column string
	Dir    string
	Type   SortValueType
}

type orderedCursor struct {
	SortBy  string `json:"sort_by"`
	SortDir string `json:"sort_dir"`
	Kind    string `json:"kind"`
	Value   string `json:"value"`
	ID      int64  `json:"id"`
}

func ResolveSort(sort ListSort, allowed map[string]SortColumn, defaultKey, defaultDir string) (ResolvedSort, error) {
	key := sort.By
	if key == "" {
		key = defaultKey
	}

	col, ok := allowed[key]
	if !ok {
		return ResolvedSort{}, fmt.Errorf("invalid sort: unsupported sort_by %q", key)
	}

	dir := strings.ToLower(sort.Dir)
	if dir == "" {
		dir = strings.ToLower(defaultDir)
	}
	if dir != SortDirAsc && dir != SortDirDesc {
		return ResolvedSort{}, fmt.Errorf("invalid sort: unsupported sort_dir %q", sort.Dir)
	}

	return ResolvedSort{
		Key:    col.Key,
		Column: col.Column,
		Dir:    dir,
		Type:   col.Type,
	}, nil
}

func (s ResolvedSort) OrderBy() string {
	sqlDir := strings.ToUpper(s.Dir)
	return fmt.Sprintf("%s %s, id %s", s.Column, sqlDir, sqlDir)
}

func (s ResolvedSort) CursorCondition(cursor string, argIdx int) (string, []interface{}, error) {
	if cursor == "" {
		return "", nil, nil
	}

	decoded, err := decodeOrderedCursor(cursor)
	if err != nil {
		return "", nil, err
	}
	if decoded.SortBy != s.Key || decoded.SortDir != s.Dir || decoded.Kind != string(s.Type) {
		return "", nil, fmt.Errorf("invalid cursor: sort does not match current sort")
	}

	value, err := parseCursorValue(s.Type, decoded.Value)
	if err != nil {
		return "", nil, err
	}

	op := ">"
	if s.Dir == SortDirDesc {
		op = "<"
	}

	return fmt.Sprintf("(%s %s $%d OR (%s = $%d AND id %s $%d))", s.Column, op, argIdx, s.Column, argIdx, op, argIdx+1),
		[]interface{}{value, int(decoded.ID)}, nil
}

func EncodeOrderedCursor(sort ResolvedSort, value interface{}, id int) (string, error) {
	encodedValue, err := formatCursorValue(sort.Type, value)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(orderedCursor{
		SortBy:  sort.Key,
		SortDir: sort.Dir,
		Kind:    string(sort.Type),
		Value:   encodedValue,
		ID:      int64(id),
	})
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}

	return base64.URLEncoding.EncodeToString(payload), nil
}

func EncodeOrderedCursor64(sort ResolvedSort, value interface{}, id int64) (string, error) {
	encodedValue, err := formatCursorValue(sort.Type, value)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(orderedCursor{
		SortBy:  sort.Key,
		SortDir: sort.Dir,
		Kind:    string(sort.Type),
		Value:   encodedValue,
		ID:      id,
	})
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}

	return base64.URLEncoding.EncodeToString(payload), nil
}

func decodeOrderedCursor(cursor string) (orderedCursor, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return orderedCursor{}, fmt.Errorf("invalid cursor: malformed encoding")
	}

	var decoded orderedCursor
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return orderedCursor{}, fmt.Errorf("invalid cursor: unexpected format")
	}
	if decoded.SortBy == "" || decoded.SortDir == "" || decoded.Kind == "" || decoded.ID <= 0 {
		return orderedCursor{}, fmt.Errorf("invalid cursor: unexpected format")
	}

	return decoded, nil
}

func formatCursorValue(kind SortValueType, value interface{}) (string, error) {
	switch kind {
	case SortValueInt:
		v, ok := value.(int)
		if !ok {
			return "", fmt.Errorf("encode cursor: expected int sort value")
		}
		return strconv.Itoa(v), nil
	case SortValueInt16:
		v, ok := value.(int16)
		if !ok {
			return "", fmt.Errorf("encode cursor: expected int16 sort value")
		}
		return strconv.FormatInt(int64(v), 10), nil
	case SortValueInt64:
		v, ok := value.(int64)
		if !ok {
			return "", fmt.Errorf("encode cursor: expected int64 sort value")
		}
		return strconv.FormatInt(v, 10), nil
	case SortValueString:
		v, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("encode cursor: expected string sort value")
		}
		return v, nil
	case SortValueTime:
		v, ok := value.(time.Time)
		if !ok {
			return "", fmt.Errorf("encode cursor: expected time sort value")
		}
		return v.UTC().Format(time.RFC3339Nano), nil
	case SortValueBool:
		v, ok := value.(bool)
		if !ok {
			return "", fmt.Errorf("encode cursor: expected bool sort value")
		}
		if v {
			return "1", nil
		}
		return "0", nil
	default:
		return "", fmt.Errorf("encode cursor: unsupported sort value type %q", kind)
	}
}

func parseCursorValue(kind SortValueType, value string) (interface{}, error) {
	switch kind {
	case SortValueInt:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: bad sort value")
		}
		return parsed, nil
	case SortValueInt16:
		parsed, err := strconv.ParseInt(value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: bad sort value")
		}
		return int16(parsed), nil
	case SortValueInt64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: bad sort value")
		}
		return parsed, nil
	case SortValueString:
		return value, nil
	case SortValueTime:
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: bad sort value")
		}
		return parsed.UTC(), nil
	case SortValueBool:
		switch value {
		case "1":
			return true, nil
		case "0":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid cursor: bad sort value")
		}
	default:
		return nil, fmt.Errorf("invalid cursor: unsupported sort type")
	}
}
