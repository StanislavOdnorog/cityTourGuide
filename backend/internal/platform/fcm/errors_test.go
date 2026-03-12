package fcm

import (
	"fmt"
	"net/http"
	"testing"
)

func TestParseSendError_UnregisteredToken(t *testing.T) {
	body := []byte(`{
		"error": {
			"code": 404,
			"message": "Requested entity was not found.",
			"status": "NOT_FOUND",
			"details": [
				{
					"@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
					"errorCode": "UNREGISTERED"
				}
			]
		}
	}`)

	se := parseSendError(http.StatusNotFound, body)

	if se.FCMCode != "UNREGISTERED" {
		t.Errorf("FCMCode = %q, want UNREGISTERED", se.FCMCode)
	}
	if !se.IsPermanent() {
		t.Error("expected permanent error for UNREGISTERED token")
	}
	if se.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", se.StatusCode, http.StatusNotFound)
	}
}

func TestParseSendError_InvalidArgument(t *testing.T) {
	body := []byte(`{
		"error": {
			"code": 400,
			"message": "The registration token is not a valid FCM registration token",
			"status": "INVALID_ARGUMENT",
			"details": [
				{
					"@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
					"errorCode": "INVALID_ARGUMENT"
				}
			]
		}
	}`)

	se := parseSendError(http.StatusBadRequest, body)

	if se.FCMCode != "INVALID_ARGUMENT" {
		t.Errorf("FCMCode = %q, want INVALID_ARGUMENT", se.FCMCode)
	}
	if !se.IsPermanent() {
		t.Error("expected permanent error for INVALID_ARGUMENT")
	}
}

func TestParseSendError_TransientError(t *testing.T) {
	body := []byte(`{
		"error": {
			"code": 503,
			"message": "Service temporarily unavailable.",
			"status": "UNAVAILABLE"
		}
	}`)

	se := parseSendError(http.StatusServiceUnavailable, body)

	if se.FCMCode != "UNAVAILABLE" {
		t.Errorf("FCMCode = %q, want UNAVAILABLE", se.FCMCode)
	}
	if se.IsPermanent() {
		t.Error("UNAVAILABLE should not be permanent")
	}
}

func TestParseSendError_InternalError(t *testing.T) {
	body := []byte(`{
		"error": {
			"code": 500,
			"message": "Internal error.",
			"status": "INTERNAL"
		}
	}`)

	se := parseSendError(http.StatusInternalServerError, body)

	if se.IsPermanent() {
		t.Error("INTERNAL should not be permanent")
	}
}

func TestParseSendError_QuotaExceeded(t *testing.T) {
	body := []byte(`{
		"error": {
			"code": 429,
			"message": "Quota exceeded.",
			"status": "RESOURCE_EXHAUSTED",
			"details": [
				{
					"@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
					"errorCode": "QUOTA_EXCEEDED"
				}
			]
		}
	}`)

	se := parseSendError(http.StatusTooManyRequests, body)

	if se.IsPermanent() {
		t.Error("QUOTA_EXCEEDED should not be permanent")
	}
}

func TestParseSendError_UnparsableBody(t *testing.T) {
	body := []byte(`not json at all`)

	se := parseSendError(http.StatusBadGateway, body)

	if se.IsPermanent() {
		t.Error("unparsable body should not be permanent")
	}
	if se.Message != "not json at all" {
		t.Errorf("Message = %q, want raw body", se.Message)
	}
}

func TestParseSendError_404WithoutDetails(t *testing.T) {
	// HTTP 404 with no FCM details should still be treated as permanent.
	body := []byte(`{"error": {"code": 404, "message": "Not found.", "status": "NOT_FOUND"}}`)

	se := parseSendError(http.StatusNotFound, body)

	if !se.IsPermanent() {
		t.Error("HTTP 404 should be permanent even without UNREGISTERED detail")
	}
}

func TestIsPermanentSendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"plain error", fmt.Errorf("some error"), false},
		{"transient SendError", &SendError{StatusCode: 500, FCMCode: "INTERNAL"}, false},
		{"permanent SendError", &SendError{StatusCode: 404, FCMCode: "UNREGISTERED", permanent: true}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPermanentSendError(tt.err)
			if got != tt.want {
				t.Errorf("IsPermanentSendError() = %v, want %v", got, tt.want)
			}
		})
	}
}
