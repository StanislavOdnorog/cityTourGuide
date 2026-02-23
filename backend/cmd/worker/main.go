package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/saas/city-stories-guide/backend/internal/config"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
	"github.com/saas/city-stories-guide/backend/internal/platform/s3"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/worker"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log.Printf("Worker config loaded: %s", cfg.LogSafe())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize database connection pool
	pool, err := repository.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	log.Println("Database connection established")

	// Initialize repositories
	inflationRepo := repository.NewInflationRepo(pool)
	storyRepo := repository.NewStoryRepo(pool)
	poiRepo := repository.NewPOIRepo(pool)

	// Initialize platform clients
	claudeClient := claude.NewClient(&claude.Config{
		APIKey: cfg.Claude.APIKey,
	})

	ttsClient := elevenlabs.NewClient(&elevenlabs.Config{
		APIKey: cfg.ElevenLabs.APIKey,
	})

	s3Client, err := s3.NewClient(ctx, &s3.Config{
		Endpoint:  cfg.S3.Endpoint,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		Bucket:    cfg.S3.Bucket,
	})
	if err != nil {
		return err
	}

	// Create and start the inflation worker
	w := worker.NewInflationWorker(
		inflationRepo,
		storyRepo,
		poiRepo,
		claudeClient,
		ttsClient,
		s3Client,
		nil,
	)

	log.Println("Starting inflation worker...")
	return w.Start(ctx)
}
