package fcm

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SendError is a structured error returned when FCM responds with a non-200 status.
type SendError struct {
	StatusCode int
	FCMCode    string // e.g. "UNREGISTERED", "INVALID_ARGUMENT"
	Message    string
	permanent  bool
}

func (e *SendError) Error() string {
	return fmt.Sprintf("fcm: status %d, code %s: %s", e.StatusCode, e.FCMCode, e.Message)
}

// IsPermanent reports whether this error indicates a permanently invalid token.
func (e *SendError) IsPermanent() bool {
	return e.permanent
}

// IsPermanentSendError reports whether err is an FCM SendError representing a
// permanently invalid token.
func IsPermanentSendError(err error) bool {
	if err == nil {
		return false
	}
	se, ok := err.(*SendError)
	return ok && se.IsPermanent()
}

// parseSendError parses the FCM HTTP response into a structured SendError.
func parseSendError(statusCode int, body []byte) *SendError {
	se := &SendError{
		StatusCode: statusCode,
		Message:    string(body),
	}

	// Try to extract the error code from the FCM JSON response.
	// FCM v1 returns: {"error": {"code": 404, "message": "...", "status": "NOT_FOUND",
	//   "details": [{"@type": "...", "errorCode": "UNREGISTERED"}]}}
	var resp struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err == nil {
		if resp.Error.Message != "" {
			se.Message = resp.Error.Message
		}
		// Prefer the FCM-specific errorCode from details if available.
		for _, d := range resp.Error.Details {
			if d.ErrorCode != "" {
				se.FCMCode = d.ErrorCode
				break
			}
		}
		// Fall back to the HTTP-level status string.
		if se.FCMCode == "" {
			se.FCMCode = resp.Error.Status
		}
	}

	se.permanent = isPermanentCode(statusCode, se.FCMCode)
	return se
}

// NewSendError creates a SendError from the given parameters, automatically
// classifying whether the error is permanent based on the status code and FCM code.
func NewSendError(statusCode int, fcmCode, message string) *SendError {
	return &SendError{
		StatusCode: statusCode,
		FCMCode:    fcmCode,
		Message:    message,
		permanent:  isPermanentCode(statusCode, fcmCode),
	}
}

// isPermanentCode classifies whether a given FCM status+code combination
// represents a permanently invalid token that should be deactivated.
func isPermanentCode(httpStatus int, fcmCode string) bool {
	switch fcmCode {
	case "UNREGISTERED":
		// Token is no longer valid (app uninstalled, token rotated, etc.)
		return true
	case "INVALID_ARGUMENT":
		// Token itself is malformed / invalid.
		return true
	}
	// HTTP 404 with no parsed code also indicates an unregistered token.
	if httpStatus == http.StatusNotFound {
		return true
	}
	return false
}
