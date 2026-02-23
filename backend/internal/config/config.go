// Package config handles application configuration loading and validation.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	S3         S3Config
	Claude     ClaudeConfig
	ElevenLabs ElevenLabsConfig
	JWT        JWTConfig
	OAuth      OAuthConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port           string
	Mode           string // "debug", "release", "test"
	AllowedOrigins []string
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL string
}

// S3Config holds S3-compatible storage settings.
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
}

// ClaudeConfig holds Anthropic Claude API settings.
type ClaudeConfig struct {
	APIKey string
}

// ElevenLabsConfig holds ElevenLabs TTS API settings.
type ElevenLabsConfig struct {
	APIKey string
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// OAuthConfig holds OAuth provider settings.
type OAuthConfig struct {
	GoogleClientID  string
	AppleClientID   string // App bundle ID
	AppleTeamID     string
	AppleKeyID      string
	ApplePrivateKey string // PEM-encoded ECDSA private key
}

// Load reads configuration from environment variables.
// It attempts to load a .env file if present (for local development).
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:           getEnv("PORT", "8080"),
			Mode:           getEnv("GIN_MODE", "debug"),
			AllowedOrigins: parseOrigins(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		},
		Database: DatabaseConfig{
			URL: os.Getenv("DATABASE_URL"),
		},
		S3: S3Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
			Bucket:    getEnv("S3_BUCKET", "city-stories"),
		},
		Claude: ClaudeConfig{
			APIKey: os.Getenv("CLAUDE_API_KEY"),
		},
		ElevenLabs: ElevenLabsConfig{
			APIKey: os.Getenv("ELEVENLABS_API_KEY"),
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			AccessTTL:  getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		OAuth: OAuthConfig{
			GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
			AppleClientID:   os.Getenv("APPLE_CLIENT_ID"),
			AppleTeamID:     os.Getenv("APPLE_TEAM_ID"),
			AppleKeyID:      os.Getenv("APPLE_KEY_ID"),
			ApplePrivateKey: os.Getenv("APPLE_PRIVATE_KEY"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required configuration fields are set.
func (c *Config) validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}

	return nil
}

// LogSafe returns a string representation of the config with sensitive values masked.
func (c *Config) LogSafe() string {
	return fmt.Sprintf(
		"Config{Server: {Port: %s, Mode: %s}, Database: {URL: ***}, S3: {Endpoint: %s, Bucket: %s}, Claude: {APIKey: %s}, ElevenLabs: {APIKey: %s}, JWT: {AccessTTL: %s, RefreshTTL: %s}}",
		c.Server.Port,
		c.Server.Mode,
		c.S3.Endpoint,
		c.S3.Bucket,
		maskKey(c.Claude.APIKey),
		maskKey(c.ElevenLabs.APIKey),
		c.JWT.AccessTTL,
		c.JWT.RefreshTTL,
	)
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getDurationEnv parses a duration from an environment variable (in seconds).
func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	seconds, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return time.Duration(seconds) * time.Second
}

// parseOrigins splits a comma-separated list of origins into a slice.
func parseOrigins(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// maskKey returns a masked version of an API key for safe logging.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}
