package fcm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestNewClient_NotConfigured(t *testing.T) {
	client, err := NewClient(context.Background(), &Config{})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	if client != nil {
		t.Fatal("expected nil client when FCM is not configured")
	}
}

func TestNewClient_CustomHTTPClientAndTimeout(t *testing.T) {
	baseClient := &http.Client{Timeout: 3 * time.Second}
	client, err := NewClient(context.Background(), &Config{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
		SendURL:     "http://example.com/send",
		HTTPClient:  baseClient,
		Timeout:     7 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	if client.httpClient == baseClient {
		t.Fatal("expected injected client to be copied")
	}
	if client.httpClient.Timeout != 7*time.Second {
		t.Errorf("http client timeout = %s, want 7s", client.httpClient.Timeout)
	}
}

func TestSend_Success(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"projects/test/messages/1"}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), &Config{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token-123"}),
		SendURL:     server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	err = client.Send(context.Background(), &Message{
		Token: "device-token",
		Title: "Hello",
		Body:  "World",
		Data:  map[string]string{"city_id": "1"},
	})
	if err != nil {
		t.Fatalf("Send() unexpected error: %v", err)
	}
	if authHeader != "Bearer token-123" {
		t.Errorf("Authorization = %q, want %q", authHeader, "Bearer token-123")
	}
}

func TestSend_Non200ReturnsStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), &Config{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token-123"}),
		SendURL:     server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	err = client.Send(context.Background(), &Message{Token: "device-token"})
	if err == nil {
		t.Fatal("expected send error")
	}

	var sendErr *SendError
	if !errors.As(err, &sendErr) {
		t.Fatalf("expected SendError, got %v", err)
	}
	if sendErr.FCMCode != "UNREGISTERED" {
		t.Errorf("FCMCode = %q, want UNREGISTERED", sendErr.FCMCode)
	}
	if !sendErr.IsPermanent() {
		t.Error("expected error to be classified as permanent")
	}
}

func TestSend_ClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), &Config{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token-123"}),
		SendURL:     server.URL,
		Timeout:     50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	err = client.Send(context.Background(), &Message{Token: "device-token"})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestSend_InjectedHTTPClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), &Config{
		TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token-123"}),
		SendURL:     server.URL,
		HTTPClient:  &http.Client{Timeout: 50 * time.Millisecond},
	})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	err = client.Send(context.Background(), &Message{Token: "device-token"})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

// failingTokenSource is an oauth2.TokenSource that always returns an error.
type failingTokenSource struct{ err error }

func (f *failingTokenSource) Token() (*oauth2.Token, error) { return nil, f.err }

func TestSend_PayloadShape(t *testing.T) {
	var gotBody []byte
	var gotMethod, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"}),
		sendURL:     server.URL,
		httpClient:  http.DefaultClient,
	}

	err := client.Send(context.Background(), &Message{
		Token: "device-abc",
		Title: "Story Ready",
		Body:  "Your story is ready to play.",
		Data:  map[string]string{"story_id": "42", "city_id": "7"},
	})
	if err != nil {
		t.Fatalf("Send() unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	// Parse the outgoing JSON payload.
	var payload struct {
		Message struct {
			Token        string            `json:"token"`
			Notification json.RawMessage   `json:"notification"`
			Data         map[string]string `json:"data"`
			Android      json.RawMessage   `json:"android"`
			APNS         json.RawMessage   `json:"apns"`
		} `json:"message"`
	}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Message.Token != "device-abc" {
		t.Errorf("token = %q, want device-abc", payload.Message.Token)
	}

	// Notification fields.
	var notif notification
	if err := json.Unmarshal(payload.Message.Notification, &notif); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}
	if notif.Title != "Story Ready" || notif.Body != "Your story is ready to play." {
		t.Errorf("notification = %+v, want title=Story Ready body=Your story is ready to play.", notif)
	}

	// Data fields.
	if payload.Message.Data["story_id"] != "42" || payload.Message.Data["city_id"] != "7" {
		t.Errorf("data = %v, want story_id=42 city_id=7", payload.Message.Data)
	}

	// Android config: high priority, channel_id, sound.
	var android androidConfig
	if err := json.Unmarshal(payload.Message.Android, &android); err != nil {
		t.Fatalf("unmarshal android: %v", err)
	}
	if android.Priority != "high" {
		t.Errorf("android.priority = %q, want high", android.Priority)
	}
	if android.Notification == nil || android.Notification.ChannelID != "city-stories" {
		t.Errorf("android.notification.channel_id = %v, want city-stories", android.Notification)
	}
	if android.Notification.Sound != "default" {
		t.Errorf("android.notification.sound = %q, want default", android.Notification.Sound)
	}

	// APNS config: sound.
	var apns apnsConfig
	if err := json.Unmarshal(payload.Message.APNS, &apns); err != nil {
		t.Fatalf("unmarshal apns: %v", err)
	}
	if apns.Payload == nil || apns.Payload.Aps == nil || apns.Payload.Aps.Sound != "default" {
		t.Errorf("apns.payload.aps.sound = %v, want default", apns)
	}
}

func TestSend_NilData_OmitsDataField(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"}),
		sendURL:     server.URL,
		httpClient:  http.DefaultClient,
	}

	err := client.Send(context.Background(), &Message{
		Token: "device-abc",
		Title: "Hello",
		Body:  "World",
		Data:  nil,
	})
	if err != nil {
		t.Fatalf("Send() unexpected error: %v", err)
	}

	// When Data is nil, the "data" key should not appear in the JSON (omitempty).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var msg map[string]json.RawMessage
	if err := json.Unmarshal(raw["message"], &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if _, ok := msg["data"]; ok {
		t.Error("expected data field to be omitted when nil")
	}
}

func TestSend_TokenSourceError(t *testing.T) {
	client := &Client{
		tokenSource: &failingTokenSource{err: fmt.Errorf("oauth2: token expired and refresh failed")},
		sendURL:     "http://localhost:0/unused",
		httpClient:  http.DefaultClient,
	}

	err := client.Send(context.Background(), &Message{Token: "device-token", Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("expected error from failing token source")
	}
	if !strings.Contains(err.Error(), "get access token") {
		t.Errorf("error = %q, want it to contain 'get access token'", err.Error())
	}
	if !strings.Contains(err.Error(), "token expired") {
		t.Errorf("error = %q, want it to wrap original cause", err.Error())
	}
}

func TestSend_TransportError(t *testing.T) {
	// Point at a listener that immediately closes connections.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close() // close so connections are refused

	client := &Client{
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"}),
		sendURL:     "http://" + addr + "/send",
		httpClient:  &http.Client{Timeout: 2 * time.Second},
	}

	sendErr := client.Send(context.Background(), &Message{Token: "device-token", Title: "t", Body: "b"})
	if sendErr == nil {
		t.Fatal("expected transport error")
	}
	if !strings.Contains(sendErr.Error(), "send request") {
		t.Errorf("error = %q, want it to contain 'send request'", sendErr.Error())
	}
}

func TestSend_Non200_UnstructuredBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service temporarily unavailable")) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := &Client{
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"}),
		sendURL:     server.URL,
		httpClient:  http.DefaultClient,
	}

	err := client.Send(context.Background(), &Message{Token: "device-token", Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}

	var sendErr *SendError
	if !errors.As(err, &sendErr) {
		t.Fatalf("expected SendError, got %T: %v", err, err)
	}
	if sendErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want 503", sendErr.StatusCode)
	}
	// With unparseable body, the raw message should be preserved.
	if !strings.Contains(sendErr.Message, "service temporarily unavailable") {
		t.Errorf("Message = %q, want it to contain upstream body", sendErr.Message)
	}
	// Non-permanent (503 is retriable).
	if sendErr.IsPermanent() {
		t.Error("503 should not be classified as permanent")
	}
}

func TestSend_Non200_IncludesStatusAndBody(t *testing.T) {
	body := `{"error":{"code":400,"message":"Invalid registration","status":"INVALID_ARGUMENT","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"INVALID_ARGUMENT"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(body)) //nolint:errcheck // test helper
	}))
	defer server.Close()

	client := &Client{
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"}),
		sendURL:     server.URL,
		httpClient:  http.DefaultClient,
	}

	err := client.Send(context.Background(), &Message{Token: "bad-token", Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	var sendErr *SendError
	if !errors.As(err, &sendErr) {
		t.Fatalf("expected SendError, got %T: %v", err, err)
	}
	if sendErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", sendErr.StatusCode)
	}
	if sendErr.FCMCode != "INVALID_ARGUMENT" {
		t.Errorf("FCMCode = %q, want INVALID_ARGUMENT", sendErr.FCMCode)
	}
	if !sendErr.IsPermanent() {
		t.Error("INVALID_ARGUMENT should be classified as permanent")
	}
	// Error() string should include status, code, and message.
	errStr := sendErr.Error()
	if !strings.Contains(errStr, "400") || !strings.Contains(errStr, "INVALID_ARGUMENT") {
		t.Errorf("Error() = %q, want it to include status and FCM code", errStr)
	}
}
