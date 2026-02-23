package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger returns a Gin middleware that writes structured JSON logs for
// every request. Fields: method, path, status_code, duration_ms, client_ip, user_id.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		userID, _ := c.Get("user_id")
		uid, _ := userID.(string)

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status_code", c.Writer.Status()),
			slog.Int64("duration_ms", duration.Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		}
		if uid != "" {
			attrs = append(attrs, slog.String("user_id", uid))
		}

		args := make([]any, len(attrs))
		for i, a := range attrs {
			args[i] = a
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			slog.Error("request completed", args...)
		case status >= 400:
			slog.Warn("request completed", args...)
		default:
			slog.Info("request completed", args...)
		}
	}
}
