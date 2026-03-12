// Package fcm provides a client for Firebase Cloud Messaging HTTP v1 API.
package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	fcmScope   = "https://www.googleapis.com/auth/firebase.messaging"
	fcmBaseURL = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
)

// Sender is the interface for sending FCM messages.
type Sender interface {
	Send(ctx context.Context, msg *Message) error
}

// Client sends push notifications via Firebase Cloud Messaging.
type Client struct {
	tokenSource oauth2.TokenSource
	projectID   string
	sendURL     string
	httpClient  *http.Client
}

// Config holds FCM client configuration.
type Config struct {
	CredentialsJSON string // Service account JSON
	HTTPClient      *http.Client
	Timeout         time.Duration
	TokenSource     oauth2.TokenSource
	SendURL         string
}

// Message represents an FCM push notification message.
type Message struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

type notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type androidConfig struct {
	Priority     string               `json:"priority,omitempty"`
	Notification *androidNotification `json:"notification,omitempty"`
}

type androidNotification struct {
	ChannelID string `json:"channel_id,omitempty"`
	Sound     string `json:"sound,omitempty"`
}

type apnsConfig struct {
	Payload *apnsPayload `json:"payload,omitempty"`
}

type apnsPayload struct {
	Aps *aps `json:"aps,omitempty"`
}

type aps struct {
	Sound string `json:"sound,omitempty"`
}

type fcmRequest struct {
	Message *fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Notification *notification     `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *androidConfig    `json:"android,omitempty"`
	APNS         *apnsConfig       `json:"apns,omitempty"`
}

// NewClient creates a new FCM client. Returns nil if credentials are not configured.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg.CredentialsJSON == "" && cfg.TokenSource == nil {
		return nil, nil //nolint:nilnil // nil client means FCM is not configured
	}

	httpClient := configuredHTTPClient(cfg.HTTPClient, cfg.Timeout, 30*time.Second)

	if cfg.TokenSource != nil {
		sendURL := cfg.SendURL
		if sendURL == "" {
			return nil, fmt.Errorf("fcm: send URL is required when using a custom token source")
		}
		return &Client{
			tokenSource: cfg.TokenSource,
			sendURL:     sendURL,
			httpClient:  httpClient,
		}, nil
	}

	credJSON := []byte(cfg.CredentialsJSON)

	// Extract project ID from credentials JSON.
	var credsData struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(credJSON, &credsData); err != nil {
		return nil, fmt.Errorf("fcm: extract project id: %w", err)
	}
	if credsData.ProjectID == "" {
		return nil, fmt.Errorf("fcm: project_id not found in credentials")
	}

	jwtCfg, err := google.JWTConfigFromJSON(credJSON, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("fcm: parse credentials: %w", err)
	}

	return &Client{
		tokenSource: jwtCfg.TokenSource(ctx),
		projectID:   credsData.ProjectID,
		sendURL:     fmt.Sprintf(fcmBaseURL, credsData.ProjectID),
		httpClient:  httpClient,
	}, nil
}

// Send sends a push notification to a device token.
func (c *Client) Send(ctx context.Context, msg *Message) error {
	token, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("fcm: get access token: %w", err)
	}

	fcmMsg := &fcmMessage{
		Token: msg.Token,
		Notification: &notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Data: msg.Data,
		Android: &androidConfig{
			Priority: "high",
			Notification: &androidNotification{
				ChannelID: "city-stories",
				Sound:     "default",
			},
		},
		APNS: &apnsConfig{
			Payload: &apnsPayload{
				Aps: &aps{
					Sound: "default",
				},
			},
		},
	}

	body, err := json.Marshal(fcmRequest{Message: fcmMsg})
	if err != nil {
		return fmt.Errorf("fcm: marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("fcm: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		sendErr := parseSendError(resp.StatusCode, respBody)
		middleware.LoggerFromContext(ctx).Error("fcm: send failed",
			"status", resp.StatusCode,
			"fcm_code", sendErr.FCMCode,
			"permanent", sendErr.IsPermanent(),
			"body", string(respBody),
		)
		return sendErr
	}

	return nil
}

func configuredHTTPClient(base *http.Client, timeout, defaultTimeout time.Duration) *http.Client {
	overrideTimeout := timeout > 0
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if base == nil {
		return &http.Client{Timeout: timeout}
	}
	clientCopy := *base
	if clientCopy.Timeout == 0 || overrideTimeout {
		clientCopy.Timeout = timeout
	}
	return &clientCopy
}
