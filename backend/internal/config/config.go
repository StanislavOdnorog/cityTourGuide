// Package config handles application configuration loading and validation.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ProviderMode controls whether external integrations use real or mock implementations.
type ProviderMode string

const (
	ProviderModeReal ProviderMode = "real"
	ProviderModeMock ProviderMode = "mock"
)

// Config holds all application configuration.
type Config struct {
	Provider   ProviderMode
	Server     ServerConfig
	Database   DatabaseConfig
	S3         S3Config
	Claude     ClaudeConfig
	ElevenLabs ElevenLabsConfig
	JWT        JWTConfig
	OAuth      OAuthConfig
	FCM        FCMConfig
}

// Runtime identifies which backend entrypoint is loading configuration.
type Runtime string

const (
	RuntimeAPI    Runtime = "api"
	RuntimeWorker Runtime = "worker"
)

const (
	defaultClaudeHTTPTimeout     = 60 * time.Second
	defaultElevenLabsHTTPTimeout = 120 * time.Second
	defaultFCMHTTPTimeout        = 30 * time.Second
)

// FCMConfig holds Firebase Cloud Messaging settings.
type FCMConfig struct {
	CredentialsJSON string // Service account JSON (base64-encoded or raw)
	Timeout         time.Duration
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            string
	Mode            string // "debug", "release", "test"
	AllowedOrigins  []string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
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
	APIKey  string
	Timeout time.Duration
}

// ElevenLabsConfig holds ElevenLabs TTS API settings.
type ElevenLabsConfig struct {
	APIKey  string
	Timeout time.Duration
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

// allowedGinModes contains the valid values for GIN_MODE.
var allowedGinModes = map[string]bool{
	"debug":   true,
	"release": true,
	"test":    true,
}

// Load reads configuration from environment variables.
// It attempts to load a .env file if present (for local development).
// Duration env vars accept either plain integer seconds (e.g. "30") or
// Go duration strings (e.g. "30s", "5m", "1h").
func Load() (*Config, error) {
	return LoadFor(RuntimeAPI)
}

// LoadFor reads configuration from environment variables for the given runtime.
func LoadFor(runtime Runtime) (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	var errs []string
	raw := captureRawEnv()

	readTimeout, err := getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}
	writeTimeout, err := getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}
	idleTimeout, err := getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}
	shutdownTimeout, err := getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}

	maxConns, err := getInt32Env("DB_MAX_CONNS", 25)
	if err != nil {
		errs = append(errs, err.Error())
	}
	minConns, err := getInt32Env("DB_MIN_CONNS", 2)
	if err != nil {
		errs = append(errs, err.Error())
	}
	maxConnLifetime, err := getDurationEnv("DB_MAX_CONN_LIFETIME", 3600*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}
	maxConnIdleTime, err := getDurationEnv("DB_MAX_CONN_IDLE_TIME", 300*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}
	healthCheckPeriod, err := getDurationEnv("DB_HEALTH_CHECK_PERIOD", 30*time.Second)
	if err != nil {
		errs = append(errs, err.Error())
	}

	accessTTL, err := getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		errs = append(errs, err.Error())
	}
	refreshTTL, err := getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		errs = append(errs, err.Error())
	}
	claudeTimeout, err := getDurationEnv("CLAUDE_HTTP_TIMEOUT", defaultClaudeHTTPTimeout)
	if err != nil {
		errs = append(errs, err.Error())
	}
	elevenLabsTimeout, err := getDurationEnv("ELEVENLABS_HTTP_TIMEOUT", defaultElevenLabsHTTPTimeout)
	if err != nil {
		errs = append(errs, err.Error())
	}
	fcmTimeout, err := getDurationEnv("FCM_HTTP_TIMEOUT", defaultFCMHTTPTimeout)
	if err != nil {
		errs = append(errs, err.Error())
	}

	origins, err := getOriginsEnv("ALLOWED_ORIGINS", []string{"http://localhost:5173"})
	if err != nil {
		errs = append(errs, err.Error())
	}

	providerMode := ProviderMode(getEnv("PROVIDER_MODE", "real"))
	if providerMode != ProviderModeReal && providerMode != ProviderModeMock {
		errs = append(errs, fmt.Sprintf("PROVIDER_MODE must be \"real\" or \"mock\"; got %q", providerMode))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config: %s", strings.Join(errs, "; "))
	}

	cfg := &Config{
		Provider: providerMode,
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			Mode:            getEnv("GIN_MODE", "debug"),
			AllowedOrigins:  origins,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			IdleTimeout:     idleTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			URL:               os.Getenv("DATABASE_URL"),
			MaxConns:          maxConns,
			MinConns:          minConns,
			MaxConnLifetime:   maxConnLifetime,
			MaxConnIdleTime:   maxConnIdleTime,
			HealthCheckPeriod: healthCheckPeriod,
		},
		S3: S3Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
			Bucket:    getEnv("S3_BUCKET", "city-stories"),
		},
		Claude: ClaudeConfig{
			APIKey:  os.Getenv("CLAUDE_API_KEY"),
			Timeout: claudeTimeout,
		},
		ElevenLabs: ElevenLabsConfig{
			APIKey:  os.Getenv("ELEVENLABS_API_KEY"),
			Timeout: elevenLabsTimeout,
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
		},
		OAuth: OAuthConfig{
			GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
			AppleClientID:   os.Getenv("APPLE_CLIENT_ID"),
			AppleTeamID:     os.Getenv("APPLE_TEAM_ID"),
			AppleKeyID:      os.Getenv("APPLE_KEY_ID"),
			ApplePrivateKey: os.Getenv("APPLE_PRIVATE_KEY"),
		},
		FCM: FCMConfig{
			CredentialsJSON: os.Getenv("FCM_CREDENTIALS_JSON"),
			Timeout:         fcmTimeout,
		},
	}

	if err := cfg.validate(runtime, raw); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required configuration fields are set.
func (c *Config) validate(runtime Runtime, raw rawEnv) error {
	var errs []string

	if c.Database.URL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}

	if c.JWT.Secret == "" {
		errs = append(errs, "JWT_SECRET is required")
	}

	// GIN_MODE validation
	if !allowedGinModes[c.Server.Mode] {
		errs = append(errs, fmt.Sprintf("GIN_MODE must be one of debug, release, test; got %q", c.Server.Mode))
	}

	// Server timeout validation
	if c.Server.ReadTimeout <= 0 {
		errs = append(errs, "SERVER_READ_TIMEOUT must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		errs = append(errs, "SERVER_WRITE_TIMEOUT must be positive")
	}
	if c.Server.IdleTimeout <= 0 {
		errs = append(errs, "SERVER_IDLE_TIMEOUT must be positive")
	}
	if c.Server.ShutdownTimeout <= 0 {
		errs = append(errs, "SERVER_SHUTDOWN_TIMEOUT must be positive")
	}
	if c.Claude.Timeout <= 0 {
		errs = append(errs, "CLAUDE_HTTP_TIMEOUT must be positive")
	}
	if c.ElevenLabs.Timeout <= 0 {
		errs = append(errs, "ELEVENLABS_HTTP_TIMEOUT must be positive")
	}
	if c.FCM.Timeout <= 0 {
		errs = append(errs, "FCM_HTTP_TIMEOUT must be positive")
	}

	// Database pool validation
	if c.Database.MaxConns <= 0 {
		errs = append(errs, "DB_MAX_CONNS must be positive")
	}
	if c.Database.MinConns < 0 {
		errs = append(errs, "DB_MIN_CONNS must not be negative")
	}
	if c.Database.MaxConnLifetime <= 0 {
		errs = append(errs, "DB_MAX_CONN_LIFETIME must be positive")
	}
	if c.Database.MaxConnIdleTime <= 0 {
		errs = append(errs, "DB_MAX_CONN_IDLE_TIME must be positive")
	}
	if c.Database.HealthCheckPeriod <= 0 {
		errs = append(errs, "DB_HEALTH_CHECK_PERIOD must be positive")
	}

	errs = append(errs, validateOptionalGroup("Google OAuth", []envField{
		raw.requiredField("GOOGLE_CLIENT_ID"),
	})...)

	errs = append(errs, validateOptionalGroup("Apple OAuth", []envField{
		raw.requiredField("APPLE_CLIENT_ID"),
		raw.requiredField("APPLE_TEAM_ID"),
		raw.requiredField("APPLE_KEY_ID"),
		raw.requiredField("APPLE_PRIVATE_KEY"),
	})...)

	if runtime == RuntimeWorker && c.Provider == ProviderModeReal {
		errs = append(errs, validateRequiredGroup("worker S3", []envField{
			raw.requiredField("S3_ENDPOINT"),
			raw.requiredField("S3_ACCESS_KEY"),
			raw.requiredField("S3_SECRET_KEY"),
			raw.requiredField("S3_BUCKET"),
		})...)
	} else {
		errs = append(errs, validateOptionalGroup("S3", []envField{
			raw.requiredField("S3_ENDPOINT"),
			raw.requiredField("S3_ACCESS_KEY"),
			raw.requiredField("S3_SECRET_KEY"),
			raw.requiredField("S3_BUCKET"),
		})...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("config: %s", strings.Join(errs, "; "))
	}
	return nil
}

// LogSafe returns a string representation of the config with sensitive values masked.
func (c *Config) LogSafe() string {
	return fmt.Sprintf(
		"Config{Provider: %s, Server: {Port: %s, Mode: %s, ReadTimeout: %s, WriteTimeout: %s, IdleTimeout: %s, ShutdownTimeout: %s}, Database: {URL: ***, MaxConns: %d, MinConns: %d, MaxConnLifetime: %s, MaxConnIdleTime: %s, HealthCheckPeriod: %s}, S3: {Endpoint: %s, Bucket: %s}, Claude: {APIKey: %s, Timeout: %s}, ElevenLabs: {APIKey: %s, Timeout: %s}, FCM: {Timeout: %s}, JWT: {AccessTTL: %s, RefreshTTL: %s}}",
		c.Provider,
		c.Server.Port,
		c.Server.Mode,
		c.Server.ReadTimeout,
		c.Server.WriteTimeout,
		c.Server.IdleTimeout,
		c.Server.ShutdownTimeout,
		c.Database.MaxConns,
		c.Database.MinConns,
		c.Database.MaxConnLifetime,
		c.Database.MaxConnIdleTime,
		c.Database.HealthCheckPeriod,
		c.S3.Endpoint,
		c.S3.Bucket,
		maskKey(c.Claude.APIKey),
		c.Claude.Timeout,
		maskKey(c.ElevenLabs.APIKey),
		c.ElevenLabs.Timeout,
		c.FCM.Timeout,
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

// getDurationEnv parses a duration from an environment variable.
// It accepts Go duration strings (e.g. "30s", "5m", "1h") or plain integer
// seconds (e.g. "30") for backward compatibility. Returns the default when
// the env var is unset. Returns an error when the env var is set but cannot
// be parsed.
func getDurationEnv(key string, defaultVal time.Duration) (time.Duration, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, fmt.Errorf("%s: invalid duration %q (value is set but empty)", key, val)
	}
	// Try plain integer seconds first (backward compatible).
	if seconds, err := strconv.Atoi(val); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	// Try Go duration string (e.g. "30s", "5m").
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q (use integer seconds or Go duration like \"30s\", \"5m\")", key, val)
	}
	return d, nil
}

// getInt32Env parses an int32 from an environment variable.
// Returns the default when the env var is unset. Returns an error when the
// env var is set but cannot be parsed.
func getInt32Env(key string, defaultVal int32) (int32, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, fmt.Errorf("%s: invalid integer %q (value is set but empty)", key, val)
	}
	n, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer %q", key, val)
	}
	return int32(n), nil
}

func getOriginsEnv(key string, defaultVal []string) ([]string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	return parseOrigins(key, val)
}

// parseOrigins splits a comma-separated list of origins into a slice and rejects
// malformed values that are unusable for CORS configuration.
func parseOrigins(key, s string) ([]string, error) {
	parts := strings.Split(s, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		origin := strings.TrimSpace(p)
		if origin == "" {
			continue
		}
		if err := validateOrigin(origin); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		origins = append(origins, origin)
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("%s: no valid origins after parsing %q", key, s)
	}
	return origins, nil
}

func validateOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin %q: %v", origin, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid origin %q: scheme must be http or https", origin)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid origin %q: host is required", origin)
	}
	if u.User != nil {
		return fmt.Errorf("invalid origin %q: user info is not allowed", origin)
	}
	if u.Path != "" {
		return fmt.Errorf("invalid origin %q: path is not allowed", origin)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid origin %q: query string and fragment are not allowed", origin)
	}
	return nil
}

type envField struct {
	name  string
	set   bool
	value string
}

type rawEnv map[string]string

func captureRawEnv() rawEnv {
	keys := []string{
		"GOOGLE_CLIENT_ID",
		"APPLE_CLIENT_ID",
		"APPLE_TEAM_ID",
		"APPLE_KEY_ID",
		"APPLE_PRIVATE_KEY",
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_BUCKET",
	}
	out := make(rawEnv, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			out[key] = value
		}
	}
	return out
}

func (r rawEnv) requiredField(name string) envField {
	value, ok := r[name]
	return envField{name: name, set: ok, value: value}
}

func validateOptionalGroup(group string, fields []envField) []string {
	enabled := false
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.set {
			enabled = true
		}
		if !field.set || strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
			continue
		}
	}
	if !enabled {
		return nil
	}
	if len(missing) == 0 {
		return nil
	}
	if len(fields) == 1 {
		return []string{fmt.Sprintf("%s requires %s to be set", group, fields[0].name)}
	}
	return []string{fmt.Sprintf("%s requires %s together; missing %s", group, joinEnvNames(fields), strings.Join(missing, ", "))}
}

func validateRequiredGroup(group string, fields []envField) []string {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if !field.set || strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s requires %s; missing %s", group, joinEnvNames(fields), strings.Join(missing, ", "))}
}

func joinEnvNames(fields []envField) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.name)
	}
	return strings.Join(names, ", ")
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
