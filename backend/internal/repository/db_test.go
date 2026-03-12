package repository

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyPoolConfig_SetsAllFields(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	opts := PoolConfig{
		MaxConns:          50,
		MinConns:          5,
		MaxConnLifetime:   2 * time.Hour,
		MaxConnIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 45 * time.Second,
	}

	applyPoolConfig(cfg, opts)

	if cfg.MaxConns != 50 {
		t.Errorf("MaxConns = %d, want 50", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Errorf("MinConns = %d, want 5", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != 2*time.Hour {
		t.Errorf("MaxConnLifetime = %v, want 2h", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != 10*time.Minute {
		t.Errorf("MaxConnIdleTime = %v, want 10m", cfg.MaxConnIdleTime)
	}
	if cfg.HealthCheckPeriod != 45*time.Second {
		t.Errorf("HealthCheckPeriod = %v, want 45s", cfg.HealthCheckPeriod)
	}
}

func TestApplyPoolConfig_PreservesDefaults_WhenEmpty(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	origMaxConns := cfg.MaxConns
	origLifetime := cfg.MaxConnLifetime
	origIdleTime := cfg.MaxConnIdleTime
	origHealthCheck := cfg.HealthCheckPeriod

	// Apply empty config (all zero values)
	applyPoolConfig(cfg, PoolConfig{})

	if cfg.MaxConns != origMaxConns {
		t.Errorf("MaxConns changed from %d to %d", origMaxConns, cfg.MaxConns)
	}
	if cfg.MaxConnLifetime != origLifetime {
		t.Errorf("MaxConnLifetime changed from %v to %v", origLifetime, cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != origIdleTime {
		t.Errorf("MaxConnIdleTime changed from %v to %v", origIdleTime, cfg.MaxConnIdleTime)
	}
	if cfg.HealthCheckPeriod != origHealthCheck {
		t.Errorf("HealthCheckPeriod changed from %v to %v", origHealthCheck, cfg.HealthCheckPeriod)
	}
}
