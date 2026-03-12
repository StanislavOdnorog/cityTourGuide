package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

// --- RedactMap tests ---

func TestRedactMap_SensitiveKeysRedacted(t *testing.T) {
	in := map[string]any{
		"authorization": "Bearer secret-jwt",
		"token":         "tok_abc",
		"refresh_token": "rt_xyz",
		"password":      "hunter2",
		"receipt":       "long-apple-receipt",
		"device_token":  "fcm:abc123",
		"email":         "user@example.com",
		"api_key":       "sk-key",
		"user_id":       "u-123",
		"status":        "ok",
	}

	out := RedactMap(in)

	for _, key := range []string{"authorization", "token", "refresh_token", "password", "receipt", "device_token", "email", "api_key"} {
		if out[key] != Placeholder {
			t.Errorf("expected %q to be redacted, got %v", key, out[key])
		}
	}
	// Non-sensitive fields preserved.
	if out["user_id"] != "u-123" {
		t.Errorf("expected user_id preserved, got %v", out["user_id"])
	}
	if out["status"] != "ok" {
		t.Errorf("expected status preserved, got %v", out["status"])
	}
}

func TestRedactMap_CaseInsensitive(t *testing.T) {
	in := map[string]any{
		"Authorization": "Bearer xyz",
		"TOKEN":         "tok",
		"Email":         "a@b.com",
	}
	out := RedactMap(in)
	for k, v := range out {
		if v != Placeholder {
			t.Errorf("expected %q (original key %q) to be redacted, got %v", k, k, v)
		}
	}
}

func TestRedactMap_NestedMaps(t *testing.T) {
	in := map[string]any{
		"request": map[string]any{
			"authorization": "Bearer nested",
			"method":        "POST",
		},
		"trace_id": "abc-123",
	}
	out := RedactMap(in)

	nested, ok := out["request"].(map[string]any)
	if !ok {
		t.Fatal("expected nested map")
	}
	if nested["authorization"] != Placeholder {
		t.Errorf("nested authorization not redacted: %v", nested["authorization"])
	}
	if nested["method"] != "POST" {
		t.Errorf("nested method should be preserved: %v", nested["method"])
	}
	if out["trace_id"] != "abc-123" {
		t.Errorf("trace_id should be preserved: %v", out["trace_id"])
	}
}

func TestRedactMap_NilInput(t *testing.T) {
	if RedactMap(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestRedactMap_EmptyInput(t *testing.T) {
	out := RedactMap(map[string]any{})
	if len(out) != 0 {
		t.Error("expected empty map for empty input")
	}
}

func TestRedactMap_OriginalUnmodified(t *testing.T) {
	in := map[string]any{"password": "secret", "user_id": "u1"}
	RedactMap(in)
	if in["password"] != "secret" {
		t.Error("original map was mutated")
	}
}

// --- RedactAttrs tests ---

func TestRedactAttrs_Mixed(t *testing.T) {
	args := RedactAttrs("token", "secret", "user_id", "u-1", "email", "a@b.com", "trace_id", "t-1")
	// token → redacted, user_id → preserved, email → redacted, trace_id → preserved
	expected := []any{"token", Placeholder, "user_id", "u-1", "email", Placeholder, "trace_id", "t-1"}
	for i, v := range args {
		if v != expected[i] {
			t.Errorf("index %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestRedactAttrs_OddLength(t *testing.T) {
	// Odd-length args should not panic; trailing element left as-is.
	args := RedactAttrs("token", "secret", "extra")
	if args[1] != Placeholder {
		t.Errorf("expected token value redacted, got %v", args[1])
	}
	if args[2] != "extra" {
		t.Errorf("trailing element should be unchanged, got %v", args[2])
	}
}

// --- RedactHeaders tests ---

func TestRedactHeaders(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Bearer xyz"},
		"Content-Type":  {"application/json"},
		"Cookie":        {"session=abc"},
		"X-Request-ID":  {"req-1"},
	}
	out := RedactHeaders(headers)

	if out["Authorization"] != Placeholder {
		t.Errorf("expected Authorization redacted, got %v", out["Authorization"])
	}
	if out["Cookie"] != Placeholder {
		t.Errorf("expected Cookie redacted, got %v", out["Cookie"])
	}
	if out["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type preserved, got %v", out["Content-Type"])
	}
	if out["X-Request-ID"] != "req-1" {
		t.Errorf("expected X-Request-ID preserved, got %v", out["X-Request-ID"])
	}
}

// --- RedactHandler (slog wrapper) tests ---

func TestRedactHandler_RedactsFlatAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewRedactHandler(inner))

	logger.Info("auth attempt",
		"token", "secret-jwt",
		"user_id", "u-42",
		"email", "test@example.com",
		"trace_id", "t-abc",
	)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", buf.String())
	}

	if entry["token"] != Placeholder {
		t.Errorf("token not redacted: %v", entry["token"])
	}
	if entry["email"] != Placeholder {
		t.Errorf("email not redacted: %v", entry["email"])
	}
	if entry["user_id"] != "u-42" {
		t.Errorf("user_id should be preserved: %v", entry["user_id"])
	}
	if entry["trace_id"] != "t-abc" {
		t.Errorf("trace_id should be preserved: %v", entry["trace_id"])
	}
}

func TestRedactHandler_RedactsGroupedAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewRedactHandler(inner))

	logger.Info("event",
		slog.Group("request",
			slog.String("authorization", "Bearer xyz"),
			slog.String("method", "POST"),
		),
	)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", buf.String())
	}

	req, ok := entry["request"].(map[string]any)
	if !ok {
		t.Fatalf("expected request group, got %v", entry["request"])
	}
	if req["authorization"] != Placeholder {
		t.Errorf("nested authorization not redacted: %v", req["authorization"])
	}
	if req["method"] != "POST" {
		t.Errorf("nested method should be preserved: %v", req["method"])
	}
}

func TestRedactHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewRedactHandler(inner)).With("password", "should-be-hidden", "trace_id", "t-1")

	logger.Info("test")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %s", buf.String())
	}
	if entry["password"] != Placeholder {
		t.Errorf("password not redacted in WithAttrs: %v", entry["password"])
	}
	if entry["trace_id"] != "t-1" {
		t.Errorf("trace_id should be preserved: %v", entry["trace_id"])
	}
}

func TestRedactHandler_Enabled(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewRedactHandler(inner)

	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("should not be enabled for Info when level is Warn")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("should be enabled for Error when level is Warn")
	}
}

// --- isSensitive tests ---

func TestIsSensitive_AllKnownKeys(t *testing.T) {
	keys := []string{
		"authorization", "token", "refresh_token", "access_token",
		"id_token", "password", "receipt", "device_token",
		"email", "api_key", "apikey", "secret", "client_secret",
		"private_key", "cookie", "set-cookie", "x-api-key",
	}
	for _, k := range keys {
		if !isSensitive(k) {
			t.Errorf("expected %q to be sensitive", k)
		}
	}
}

func TestIsSensitive_Credentials(t *testing.T) {
	if !isSensitive("credentials") {
		t.Error("expected credentials to be sensitive")
	}
	if !isSensitive("Credentials") {
		t.Error("expected Credentials to be sensitive (case-insensitive)")
	}
}

func TestRedactAny_Nil(t *testing.T) {
	if RedactAny(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestRedactAny_FlatMap(t *testing.T) {
	in := map[string]any{"name": "Tbilisi", "token": "secret"}
	out, ok := RedactAny(in).(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any")
	}
	if out["name"] != "Tbilisi" {
		t.Errorf("expected name preserved, got %v", out["name"])
	}
	if out["token"] != Placeholder {
		t.Errorf("expected token redacted, got %v", out["token"])
	}
}

func TestRedactAny_NestedMap(t *testing.T) {
	in := map[string]any{
		"user": map[string]any{
			"id":       "u-1",
			"password": "hunter2",
		},
	}
	out := RedactAny(in).(map[string]any)
	nested := out["user"].(map[string]any)
	if nested["id"] != "u-1" {
		t.Errorf("expected id preserved, got %v", nested["id"])
	}
	if nested["password"] != Placeholder {
		t.Errorf("expected password redacted, got %v", nested["password"])
	}
}

func TestRedactAny_SliceOfMaps(t *testing.T) {
	in := []any{
		map[string]any{"name": "a", "secret": "s1"},
		map[string]any{"name": "b", "secret": "s2"},
	}
	out := RedactAny(in).([]any)
	if len(out) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(out))
	}
	for i, elem := range out {
		m := elem.(map[string]any)
		if m["secret"] != Placeholder {
			t.Errorf("element %d: expected secret redacted, got %v", i, m["secret"])
		}
	}
}

func TestRedactAny_Struct(t *testing.T) {
	type payload struct {
		Name     string `json:"name"`
		APIKey   string `json:"api_key"`
		Password string `json:"password"`
	}
	in := payload{Name: "test", APIKey: "key123", Password: "pass"}
	out := RedactAny(in).(map[string]any)
	if out["name"] != "test" {
		t.Errorf("expected name preserved, got %v", out["name"])
	}
	if out["api_key"] != Placeholder {
		t.Errorf("expected api_key redacted, got %v", out["api_key"])
	}
	if out["password"] != Placeholder {
		t.Errorf("expected password redacted, got %v", out["password"])
	}
}

func TestRedactAny_DeeplyNested(t *testing.T) {
	in := map[string]any{
		"config": map[string]any{
			"auth": map[string]any{
				"credentials": "deep-secret",
				"provider":    "oauth",
			},
		},
	}
	out := RedactAny(in).(map[string]any)
	auth := out["config"].(map[string]any)["auth"].(map[string]any)
	if auth["credentials"] != Placeholder {
		t.Errorf("expected credentials redacted, got %v", auth["credentials"])
	}
	if auth["provider"] != "oauth" {
		t.Errorf("expected provider preserved, got %v", auth["provider"])
	}
}

func TestIsSensitive_NonSensitiveKeys(t *testing.T) {
	keys := []string{"user_id", "trace_id", "method", "path", "status_code", "duration_ms", "platform"}
	for _, k := range keys {
		if isSensitive(k) {
			t.Errorf("expected %q to NOT be sensitive", k)
		}
	}
}
