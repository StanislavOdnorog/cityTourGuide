package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// requestLog is the structured log entry written for every HTTP request.
type requestLog struct {
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	DurationMs int64  `json:"duration_ms"`
	ClientIP   string `json:"client_ip"`
	UserID     string `json:"user_id,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// RequestLogger returns a Gin middleware that writes structured JSON logs for
// every request. Fields: method, path, status_code, duration_ms, client_ip, user_id.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		userID, _ := c.Get("user_id")
		uid, _ := userID.(string)

		level := "info"
		if c.Writer.Status() >= 500 {
			level = "error"
		} else if c.Writer.Status() >= 400 {
			level = "warn"
		}

		entry := requestLog{
			Timestamp:  start.UTC().Format(time.RFC3339),
			Level:      level,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			DurationMs: duration.Milliseconds(),
			ClientIP:   c.ClientIP(),
			UserID:     uid,
			UserAgent:  c.Request.UserAgent(),
		}

		data, err := json.Marshal(entry)
		if err != nil {
			log.Printf("failed to marshal request log: %v", err)
			return
		}

		log.Println(string(data))
	}
}
