package middleware

import (
	"context"
	"encoding/hex"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// TraceIDKey is the Gin context key used to store the request trace ID.
	TraceIDKey = "trace_id"

	requestIDHeader = "X-Request-ID"
	maxRequestIDLen = 128
)

type loggerContextKey struct{}

var traceRand = struct {
	mu  sync.Mutex
	rng *rand.Rand
}{
	rng: rand.New(rand.NewSource(time.Now().UnixNano())),
}

// TraceIDMiddleware ensures every request has a correlation ID and request-scoped logger.
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := requestIDFromHeader(c.GetHeader(requestIDHeader))
		if traceID == "" {
			traceID = newTraceID()
		}

		c.Set(TraceIDKey, traceID)
		c.Header(requestIDHeader, traceID)

		logger := LoggerFromContext(c.Request.Context()).With("trace_id", traceID)
		c.Request = c.Request.WithContext(ContextWithLogger(c.Request.Context(), logger))

		c.Next()
	}
}

// ContextWithLogger stores a slog logger in a context. Nil inputs fall back safely.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext returns the logger stored in context or slog.Default().
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// TraceID returns the trace ID stored on the Gin context.
func TraceID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	traceID, _ := c.Get(TraceIDKey)
	value, _ := traceID.(string)
	return value
}

func requestIDFromHeader(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > maxRequestIDLen {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return ""
	}
	return value
}

func newTraceID() string {
	var raw [16]byte

	traceRand.mu.Lock()
	for i := range raw {
		raw[i] = byte(traceRand.rng.Intn(256))
	}
	traceRand.mu.Unlock()

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	var dst [36]byte
	hex.Encode(dst[0:8], raw[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], raw[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], raw[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], raw[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], raw[10:16])

	return string(dst[:])
}

func errorBody(c *gin.Context, message string) gin.H {
	body := gin.H{"error": message}
	if traceID := TraceID(c); traceID != "" {
		body["request_id"] = traceID
	}
	return body
}

func errorBodyWithFields(c *gin.Context, message string, fields gin.H) gin.H {
	body := errorBody(c, message)
	for k, v := range fields {
		body[k] = v
	}
	return body
}

func abortErrorJSON(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, errorBody(c, message))
}

func abortErrorJSONWithFields(c *gin.Context, status int, message string, fields gin.H) {
	c.AbortWithStatusJSON(status, errorBodyWithFields(c, message, fields))
}
