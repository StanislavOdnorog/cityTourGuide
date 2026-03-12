package config

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// setRequiredEnv sets the minimum env vars needed for config.Load() to succeed.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
	t.Setenv("JWT_SECRET", "test-secret")
}

func TestLoad_ServerTimeoutDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadTimeout", cfg.Server.ReadTimeout, 10 * time.Second},
		{"WriteTimeout", cfg.Server.WriteTimeout, 30 * time.Second},
		{"IdleTimeout", cfg.Server.IdleTimeout, 120 * time.Second},
		{"ShutdownTimeout", cfg.Server.ShutdownTimeout, 30 * time.Second},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("Server.%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestLoad_ProviderTimeoutDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Claude.Timeout != 60*time.Second {
		t.Errorf("Claude.Timeout = %v, want 60s", cfg.Claude.Timeout)
	}
	if cfg.ElevenLabs.Timeout != 120*time.Second {
		t.Errorf("ElevenLabs.Timeout = %v, want 120s", cfg.ElevenLabs.Timeout)
	}
	if cfg.FCM.Timeout != 30*time.Second {
		t.Errorf("FCM.Timeout = %v, want 30s", cfg.FCM.Timeout)
	}
}

func TestLoad_DatabasePoolDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.MaxConns != 25 {
		t.Errorf("Database.MaxConns = %d, want 25", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 2 {
		t.Errorf("Database.MinConns = %d, want 2", cfg.Database.MinConns)
	}
	if cfg.Database.MaxConnLifetime != 3600*time.Second {
		t.Errorf("Database.MaxConnLifetime = %v, want 1h", cfg.Database.MaxConnLifetime)
	}
	if cfg.Database.MaxConnIdleTime != 300*time.Second {
		t.Errorf("Database.MaxConnIdleTime = %v, want 5m", cfg.Database.MaxConnIdleTime)
	}
	if cfg.Database.HealthCheckPeriod != 30*time.Second {
		t.Errorf("Database.HealthCheckPeriod = %v, want 30s", cfg.Database.HealthCheckPeriod)
	}
}

func TestLoad_ServerTimeoutCustomEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SERVER_READ_TIMEOUT", "5")
	t.Setenv("SERVER_WRITE_TIMEOUT", "60")
	t.Setenv("SERVER_IDLE_TIMEOUT", "90")
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", "15")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 5s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 60*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want 60s", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 90*time.Second {
		t.Errorf("Server.IdleTimeout = %v, want 90s", cfg.Server.IdleTimeout)
	}
	if cfg.Server.ShutdownTimeout != 15*time.Second {
		t.Errorf("Server.ShutdownTimeout = %v, want 15s", cfg.Server.ShutdownTimeout)
	}
}

func TestLoad_DurationGoDurationFormat(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SERVER_READ_TIMEOUT", "5s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "1m")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "168h")
	t.Setenv("CLAUDE_HTTP_TIMEOUT", "11s")
	t.Setenv("ELEVENLABS_HTTP_TIMEOUT", "2m")
	t.Setenv("FCM_HTTP_TIMEOUT", "45s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 5s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != time.Minute {
		t.Errorf("Server.WriteTimeout = %v, want 1m", cfg.Server.WriteTimeout)
	}
	if cfg.JWT.AccessTTL != 15*time.Minute {
		t.Errorf("JWT.AccessTTL = %v, want 15m", cfg.JWT.AccessTTL)
	}
	if cfg.JWT.RefreshTTL != 168*time.Hour {
		t.Errorf("JWT.RefreshTTL = %v, want 168h", cfg.JWT.RefreshTTL)
	}
	if cfg.Claude.Timeout != 11*time.Second {
		t.Errorf("Claude.Timeout = %v, want 11s", cfg.Claude.Timeout)
	}
	if cfg.ElevenLabs.Timeout != 2*time.Minute {
		t.Errorf("ElevenLabs.Timeout = %v, want 2m", cfg.ElevenLabs.Timeout)
	}
	if cfg.FCM.Timeout != 45*time.Second {
		t.Errorf("FCM.Timeout = %v, want 45s", cfg.FCM.Timeout)
	}
}

func TestLoad_DatabasePoolCustomEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_MAX_CONNS", "50")
	t.Setenv("DB_MIN_CONNS", "5")
	t.Setenv("DB_MAX_CONN_LIFETIME", "7200")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "600")
	t.Setenv("DB_HEALTH_CHECK_PERIOD", "60")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.MaxConns != 50 {
		t.Errorf("Database.MaxConns = %d, want 50", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 5 {
		t.Errorf("Database.MinConns = %d, want 5", cfg.Database.MinConns)
	}
	if cfg.Database.MaxConnLifetime != 7200*time.Second {
		t.Errorf("Database.MaxConnLifetime = %v, want 2h", cfg.Database.MaxConnLifetime)
	}
	if cfg.Database.MaxConnIdleTime != 600*time.Second {
		t.Errorf("Database.MaxConnIdleTime = %v, want 10m", cfg.Database.MaxConnIdleTime)
	}
	if cfg.Database.HealthCheckPeriod != 60*time.Second {
		t.Errorf("Database.HealthCheckPeriod = %v, want 60s", cfg.Database.HealthCheckPeriod)
	}
}

func TestLoad_RejectsZeroServerTimeout(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SERVER_READ_TIMEOUT", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for zero SERVER_READ_TIMEOUT, got nil")
	}
}

func TestLoad_RejectsNegativeMaxConns(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_MAX_CONNS", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for negative DB_MAX_CONNS, got nil")
	}
}

func TestLoad_MalformedDurationReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		value  string
	}{
		{"garbage duration", "SERVER_READ_TIMEOUT", "notanumber"},
		{"float seconds", "SERVER_WRITE_TIMEOUT", "3.5x"},
		{"empty-ish", "SERVER_IDLE_TIMEOUT", "  "},
		{"explicit empty", "JWT_ACCESS_TTL", ""},
		{"db duration garbage", "DB_MAX_CONN_LIFETIME", "abc"},
		{"jwt duration garbage", "JWT_ACCESS_TTL", "xyz"},
		{"claude duration garbage", "CLAUDE_HTTP_TIMEOUT", "abc"},
		{"elevenlabs duration garbage", "ELEVENLABS_HTTP_TIMEOUT", "what"},
		{"fcm duration garbage", "FCM_HTTP_TIMEOUT", "forever"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.envKey, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() expected error for %s=%q, got nil", tt.envKey, tt.value)
			}
			if !strings.Contains(err.Error(), tt.envKey) {
				t.Errorf("error should mention %s, got: %v", tt.envKey, err)
			}
		})
	}
}

func TestLoad_MalformedInt32ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		value  string
	}{
		{"garbage", "DB_MAX_CONNS", "abc"},
		{"float", "DB_MIN_CONNS", "2.5"},
		{"overflow", "DB_MAX_CONNS", "99999999999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.envKey, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() expected error for %s=%q, got nil", tt.envKey, tt.value)
			}
			if !strings.Contains(err.Error(), tt.envKey) {
				t.Errorf("error should mention %s, got: %v", tt.envKey, err)
			}
		})
	}
}

func TestLoad_GinModeValidation(t *testing.T) {
	validModes := []string{"debug", "release", "test"}
	for _, mode := range validModes {
		t.Run("valid_"+mode, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("GIN_MODE", mode)

			_, err := Load()
			if err != nil {
				t.Fatalf("Load() unexpected error for GIN_MODE=%q: %v", mode, err)
			}
		})
	}

	invalidModes := []string{"production", "staging", "dev"}
	for _, mode := range invalidModes {
		t.Run("invalid_"+mode, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("GIN_MODE", mode)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() expected error for GIN_MODE=%q, got nil", mode)
			}
			if !strings.Contains(err.Error(), "GIN_MODE") {
				t.Errorf("error should mention GIN_MODE, got: %v", err)
			}
		})
	}
}

func TestLoad_RejectsZeroProviderTimeout(t *testing.T) {
	tests := []string{
		"CLAUDE_HTTP_TIMEOUT",
		"ELEVENLABS_HTTP_TIMEOUT",
		"FCM_HTTP_TIMEOUT",
	}

	for _, envKey := range tests {
		t.Run(envKey, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(envKey, "0")

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() expected error for %s=0, got nil", envKey)
			}
			if !strings.Contains(err.Error(), envKey) {
				t.Errorf("error should mention %s, got: %v", envKey, err)
			}
		})
	}
}

func TestLoad_DefaultGinMode(t *testing.T) {
	setRequiredEnv(t)
	// GIN_MODE not set — should default to "debug"
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, want %q", cfg.Server.Mode, "debug")
	}
}

func TestLoad_OriginsValidation(t *testing.T) {
	t.Run("valid comma-separated", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000, https://example.com")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if len(cfg.Server.AllowedOrigins) != 2 {
			t.Errorf("AllowedOrigins length = %d, want 2", len(cfg.Server.AllowedOrigins))
		}
	})

	t.Run("only commas and spaces", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("ALLOWED_ORIGINS", " , , ")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for empty origins, got nil")
		}
		if !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
			t.Errorf("error should mention ALLOWED_ORIGINS, got: %v", err)
		}
	})

	t.Run("rejects invalid origins", func(t *testing.T) {
		tests := []string{
			"localhost:3000",
			"ftp://example.com",
			"https://example.com/path",
			"https://user@example.com",
			"https://example.com?x=1",
		}

		for _, origin := range tests {
			t.Run(origin, func(t *testing.T) {
				setRequiredEnv(t)
				t.Setenv("ALLOWED_ORIGINS", origin)

				_, err := Load()
				if err == nil {
					t.Fatalf("Load() expected error for ALLOWED_ORIGINS=%q, got nil", origin)
				}
				if !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
					t.Errorf("error should mention ALLOWED_ORIGINS, got: %v", err)
				}
				if !strings.Contains(err.Error(), origin) {
					t.Errorf("error should mention invalid origin value %q, got: %v", origin, err)
				}
			})
		}
	})
}

func TestLoad_AppleOAuthAllOrNothing(t *testing.T) {
	t.Run("all set", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("APPLE_CLIENT_ID", "com.example.app")
		t.Setenv("APPLE_TEAM_ID", "TEAMID")
		t.Setenv("APPLE_KEY_ID", "KEYID")
		t.Setenv("APPLE_PRIVATE_KEY", "-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----")

		_, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
	})

	t.Run("none set", func(t *testing.T) {
		setRequiredEnv(t)

		_, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
	})

	t.Run("partial - only client ID", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("APPLE_CLIENT_ID", "com.example.app")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for partial Apple OAuth config, got nil")
		}
		if !strings.Contains(err.Error(), "Apple OAuth") {
			t.Errorf("error should mention Apple OAuth, got: %v", err)
		}
		if !strings.Contains(err.Error(), "APPLE_TEAM_ID") {
			t.Errorf("error should mention missing Apple env vars, got: %v", err)
		}
	})

	t.Run("partial - missing private key", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("APPLE_CLIENT_ID", "com.example.app")
		t.Setenv("APPLE_TEAM_ID", "TEAMID")
		t.Setenv("APPLE_KEY_ID", "KEYID")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for partial Apple OAuth config, got nil")
		}
	})
}

func TestLoad_GoogleOAuthValidation(t *testing.T) {
	t.Run("unset is allowed", func(t *testing.T) {
		setRequiredEnv(t)

		if _, err := Load(); err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
	})

	t.Run("configured empty value fails", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for empty GOOGLE_CLIENT_ID, got nil")
		}
		if !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID") {
			t.Errorf("error should mention GOOGLE_CLIENT_ID, got: %v", err)
		}
	})

	t.Run("configured value succeeds", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "google-client-id")

		if _, err := Load(); err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
	})
}

func TestLoad_S3GroupedValidation(t *testing.T) {
	t.Run("api allows S3 to be fully absent", func(t *testing.T) {
		setRequiredEnv(t)

		if _, err := Load(); err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
	})

	t.Run("api rejects partial S3 config", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("S3_ENDPOINT", "http://localhost:9000")
		t.Setenv("S3_ACCESS_KEY", "minioadmin")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for partial S3 config, got nil")
		}
		for _, key := range []string{"S3_SECRET_KEY", "S3_BUCKET"} {
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error should mention missing %s, got: %v", key, err)
			}
		}
	})

	t.Run("worker requires full S3 config", func(t *testing.T) {
		setRequiredEnv(t)

		_, err := LoadFor(RuntimeWorker)
		if err == nil {
			t.Fatal("LoadFor(RuntimeWorker) expected error when S3 is absent, got nil")
		}
		for _, key := range []string{"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error should mention missing %s, got: %v", key, err)
			}
		}
	})

	t.Run("worker accepts complete S3 config", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("S3_ENDPOINT", "http://localhost:9000")
		t.Setenv("S3_ACCESS_KEY", "minioadmin")
		t.Setenv("S3_SECRET_KEY", "minioadmin-secret")
		t.Setenv("S3_BUCKET", "city-stories")

		cfg, err := LoadFor(RuntimeWorker)
		if err != nil {
			t.Fatalf("LoadFor(RuntimeWorker) unexpected error: %v", err)
		}
		if cfg.S3.Bucket != "city-stories" {
			t.Errorf("S3.Bucket = %q, want city-stories", cfg.S3.Bucket)
		}
	})
}

func TestLoad_MultilineApplePrivateKey(t *testing.T) {
	setRequiredEnv(t)
	key := "-----BEGIN PRIVATE KEY-----\nMIGHAgEA\nmultiline\ncontent\n-----END PRIVATE KEY-----"
	t.Setenv("APPLE_CLIENT_ID", "com.example.app")
	t.Setenv("APPLE_TEAM_ID", "TEAMID")
	t.Setenv("APPLE_KEY_ID", "KEYID")
	t.Setenv("APPLE_PRIVATE_KEY", key)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OAuth.ApplePrivateKey != key {
		t.Error("Apple private key was modified during config loading")
	}
}

func TestLogSafe_DoesNotLeakDatabaseURL(t *testing.T) {
	setRequiredEnv(t)
	// Override with a recognizable URL
	os.Setenv("DATABASE_URL", "postgres://user:supersecret@host:5432/db")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	output := cfg.LogSafe()
	if strings.Contains(output, "supersecret") {
		t.Errorf("LogSafe() leaked database URL: %s", output)
	}
	if !strings.Contains(output, "ReadTimeout") {
		t.Errorf("LogSafe() missing ReadTimeout field")
	}
	if !strings.Contains(output, "MaxConns") {
		t.Errorf("LogSafe() missing MaxConns field")
	}
}

func TestLoad_ErrorDoesNotLeakSecrets(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SERVER_READ_TIMEOUT", "badvalue")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "test-secret") {
		t.Errorf("error message leaked JWT_SECRET: %s", errMsg)
	}
}

func TestGetDurationEnv_Unit(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"unset returns default", "", 42 * time.Second, false},
		{"plain seconds", "30", 30 * time.Second, false},
		{"go duration seconds", "30s", 30 * time.Second, false},
		{"go duration minutes", "5m", 5 * time.Minute, false},
		{"go duration hours", "2h", 2 * time.Hour, false},
		{"go duration composite", "1h30m", 90 * time.Minute, false},
		{"garbage", "notanumber", 0, true},
		{"empty-ish spaces", "  ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_DURATION_" + tt.name
			if tt.value != "" {
				t.Setenv(key, tt.value)
			}
			got, err := getDurationEnv(key, 42*time.Second)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getDurationEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("getDurationEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt32Env_Unit(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int32
		wantErr bool
	}{
		{"unset returns default", "", 99, false},
		{"valid", "42", 42, false},
		{"negative", "-5", -5, false},
		{"garbage", "abc", 0, true},
		{"float", "3.14", 0, true},
		{"overflow", "99999999999999", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT32_" + tt.name
			if tt.value != "" {
				t.Setenv(key, tt.value)
			}
			got, err := getInt32Env(key, 99)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getInt32Env() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("getInt32Env() = %v, want %v", got, tt.want)
			}
		})
	}
}

// configEnvKeys lists every environment variable consumed by config.Load().
// UPDATE THIS LIST when you add or remove an env var in config.go.
var configEnvKeys = []string{
	"PROVIDER_MODE",
	"PORT",
	"GIN_MODE",
	"ALLOWED_ORIGINS",
	"SERVER_READ_TIMEOUT",
	"SERVER_WRITE_TIMEOUT",
	"SERVER_IDLE_TIMEOUT",
	"SERVER_SHUTDOWN_TIMEOUT",
	"DATABASE_URL",
	"DB_MAX_CONNS",
	"DB_MIN_CONNS",
	"DB_MAX_CONN_LIFETIME",
	"DB_MAX_CONN_IDLE_TIME",
	"DB_HEALTH_CHECK_PERIOD",
	"S3_ENDPOINT",
	"S3_ACCESS_KEY",
	"S3_SECRET_KEY",
	"S3_BUCKET",
	"CLAUDE_API_KEY",
	"CLAUDE_HTTP_TIMEOUT",
	"ELEVENLABS_API_KEY",
	"ELEVENLABS_HTTP_TIMEOUT",
	"JWT_SECRET",
	"JWT_ACCESS_TTL",
	"JWT_REFRESH_TTL",
	"GOOGLE_CLIENT_ID",
	"APPLE_CLIENT_ID",
	"APPLE_TEAM_ID",
	"APPLE_KEY_ID",
	"APPLE_PRIVATE_KEY",
	"FCM_CREDENTIALS_JSON",
	"FCM_HTTP_TIMEOUT",
}

// TestEnvExampleParity ensures backend/.env.example contains every environment
// variable consumed by the config loader. If this test fails, add the missing
// key to .env.example with a sensible default or empty placeholder.
func TestEnvExampleParity(t *testing.T) {
	// Locate .env.example relative to this test file.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// config_test.go lives at backend/internal/config/ — go up to backend/.
	envExamplePath := filepath.Join(filepath.Dir(thisFile), "..", "..", ".env.example")

	f, err := os.Open(envExamplePath)
	if err != nil {
		t.Fatalf("failed to open .env.example: %v", err)
	}
	defer f.Close()

	// Parse keys present in .env.example (lines matching KEY= or KEY=value,
	// ignoring comments and blank lines).
	exampleKeys := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			exampleKeys[key] = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading .env.example: %v", err)
	}

	var missing []string
	for _, key := range configEnvKeys {
		if !exampleKeys[key] {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf(".env.example is missing config keys: %s\n"+
			"Add them to backend/.env.example and update configEnvKeys in config_test.go if needed.",
			strings.Join(missing, ", "))
	}
}

func TestLoad_ProviderModeDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Provider != ProviderModeReal {
		t.Errorf("Provider = %q, want %q", cfg.Provider, ProviderModeReal)
	}
}

func TestLoad_ProviderModeValidValues(t *testing.T) {
	for _, mode := range []string{"real", "mock"} {
		t.Run(mode, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("PROVIDER_MODE", mode)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if string(cfg.Provider) != mode {
				t.Errorf("Provider = %q, want %q", cfg.Provider, mode)
			}
		})
	}
}

func TestLoad_ProviderModeRejectsInvalid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PROVIDER_MODE", "staging")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for PROVIDER_MODE=staging, got nil")
	}
	if !strings.Contains(err.Error(), "PROVIDER_MODE") {
		t.Errorf("error should mention PROVIDER_MODE, got: %v", err)
	}
}

func TestLoad_MockModeWorkerSkipsS3Validation(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PROVIDER_MODE", "mock")
	// No S3 env vars set — should succeed for worker in mock mode.

	cfg, err := LoadFor(RuntimeWorker)
	if err != nil {
		t.Fatalf("LoadFor(RuntimeWorker) unexpected error in mock mode: %v", err)
	}
	if cfg.Provider != ProviderModeMock {
		t.Errorf("Provider = %q, want %q", cfg.Provider, ProviderModeMock)
	}
}

func TestLoad_RealModeWorkerRequiresS3(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PROVIDER_MODE", "real")
	// No S3 env vars set — should fail for worker in real mode.

	_, err := LoadFor(RuntimeWorker)
	if err == nil {
		t.Fatal("LoadFor(RuntimeWorker) expected error when S3 is absent in real mode, got nil")
	}
	if !strings.Contains(err.Error(), "S3_ENDPOINT") {
		t.Errorf("error should mention S3_ENDPOINT, got: %v", err)
	}
}
