// Package logger provides structured JSON logging using Go's log/slog.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the global slog logger with a JSON handler wrapped by
// RedactHandler. The redaction layer ensures sensitive attributes (tokens,
// passwords, etc.) are scrubbed even if application code accidentally logs them.
// The log level is controlled by the LOG_LEVEL environment variable
// (debug, info, warn, error). Default is info.
func Setup() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	})

	slog.SetDefault(slog.New(NewRedactHandler(jsonHandler)))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
