package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
)

// Placeholder is the fixed string that replaces sensitive values.
const Placeholder = "[REDACTED]"

// sensitiveKeys is the canonical set of keys whose values must never appear in
// logs. All comparisons are case-insensitive.
var sensitiveKeys = map[string]struct{}{
	"authorization":  {},
	"token":          {},
	"refresh_token":  {},
	"access_token":   {},
	"id_token":       {},
	"password":       {},
	"receipt":        {},
	"device_token":   {},
	"email":          {},
	"api_key":        {},
	"apikey":         {},
	"secret":         {},
	"client_secret":  {},
	"credentials":    {},
	"private_key":    {},
	"cookie":         {},
	"set-cookie":     {},
	"x-api-key":      {},
}

// isSensitive returns true when key (case-insensitive) is in the sensitive set.
func isSensitive(key string) bool {
	_, ok := sensitiveKeys[strings.ToLower(key)]
	return ok
}

// RedactMap returns a shallow copy of m with sensitive keys replaced by
// Placeholder. Nested maps are handled recursively.
func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isSensitive(k) {
			out[k] = Placeholder
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			out[k] = RedactMap(nested)
			continue
		}
		out[k] = v
	}
	return out
}

// RedactAny sanitises an arbitrary value for safe persistence (e.g. audit
// logs). Structs are converted to map[string]any via a JSON round-trip so that
// sensitive keys can be discovered. Maps and slices are walked recursively.
// Returns nil when v is nil or cannot be converted.
func RedactAny(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		return redactMapAny(val)
	case []any:
		return redactSlice(val)
	default:
		// Struct or other type: JSON round-trip to map[string]any.
		b, err := json.Marshal(val)
		if err != nil {
			return nil
		}
		var generic any
		if err := json.Unmarshal(b, &generic); err != nil {
			return nil
		}
		return RedactAny(generic)
	}
}

// redactMapAny is the recursive map walker used by RedactAny.
func redactMapAny(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isSensitive(k) {
			out[k] = Placeholder
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			out[k] = redactMapAny(val)
		case []any:
			out[k] = redactSlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

// redactSlice walks each element for maps or nested slices.
func redactSlice(s []any) []any {
	out := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			out[i] = redactMapAny(val)
		case []any:
			out[i] = redactSlice(val)
		default:
			out[i] = v
		}
	}
	return out
}

// RedactAttrs accepts slog key-value pairs (as passed to slog.Info, etc.) and
// returns a new slice with sensitive values replaced. This is useful for
// sanitising ad-hoc log calls:
//
//	logger.Info("event", logger.RedactAttrs("token", tok, "user_id", uid)...)
func RedactAttrs(args ...any) []any {
	out := make([]any, len(args))
	copy(out, args)
	for i := 0; i+1 < len(out); i += 2 {
		key, ok := out[i].(string)
		if !ok {
			continue
		}
		if isSensitive(key) {
			out[i+1] = Placeholder
		}
	}
	return out
}

// RedactHeaders returns a map of header name → value with sensitive headers
// replaced by Placeholder. Only the first value of each header is kept.
func RedactHeaders(headers map[string][]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, vals := range headers {
		if isSensitive(k) {
			out[k] = Placeholder
			continue
		}
		if len(vals) > 0 {
			out[k] = vals[0]
		}
	}
	return out
}

// --- slog Handler wrapper (defense-in-depth) ---

// RedactHandler wraps an slog.Handler and scrubs sensitive attributes before
// they reach the underlying handler. Install it as a safety net around the
// default JSON handler so that even if application code accidentally passes a
// sensitive key, the value is replaced.
type RedactHandler struct {
	inner slog.Handler
}

// NewRedactHandler creates a new RedactHandler wrapping inner.
func NewRedactHandler(inner slog.Handler) *RedactHandler {
	return &RedactHandler{inner: inner}
}

func (h *RedactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *RedactHandler) Handle(ctx context.Context, r slog.Record) error {
	var cleaned []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		cleaned = append(cleaned, redactAttr(a))
		return true
	})

	// Build a new record with cleaned attrs.
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	nr.AddAttrs(cleaned...)
	return h.inner.Handle(ctx, nr)
}

func (h *RedactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = redactAttr(a)
	}
	return NewRedactHandler(h.inner.WithAttrs(redacted))
}

func (h *RedactHandler) WithGroup(name string) slog.Handler {
	return NewRedactHandler(h.inner.WithGroup(name))
}

func redactAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		redacted := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			redacted[i] = redactAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(redacted...)}
	}
	if isSensitive(a.Key) {
		return slog.String(a.Key, Placeholder)
	}
	return a
}
