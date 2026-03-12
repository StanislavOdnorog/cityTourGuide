package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// reset truncates all application tables (preserving schema and migrations)
// so the database can be cleanly reseeded. This is intended for local
// development only and refuses to run against production-looking URLs.

func run() error {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	lower := strings.ToLower(dbURL)
	for _, keyword := range []string{"prod", "production", "rds.amazonaws.com", "cloud.google.com"} {
		if strings.Contains(lower, keyword) {
			return fmt.Errorf("DATABASE_URL contains %q — refusing to reset what looks like a production database", keyword)
		}
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %v", err)
	}

	// Truncate in dependency order (children first). CASCADE handles any
	// remaining FK constraints.
	tables := []string{
		"admin_audit_logs",
		"user_listening",
		"report",
		"push_notifications",
		"device_tokens",
		"story",
		"poi",
		"purchases",
		"inflation_jobs",
		"users",
		"cities",
	}

	for _, t := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", t))
		if err != nil {
			// Table might not exist if migrations haven't all been applied.
			log.Printf("  WARN: could not truncate %s: %v", t, err)
		} else {
			log.Printf("  Truncated %s", t)
		}
	}

	fmt.Println()
	fmt.Println("Database reset complete. Run 'make seed' to repopulate.")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
