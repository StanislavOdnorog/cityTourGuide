package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/config"
	"github.com/saas/city-stories-guide/backend/internal/logger"
	"github.com/saas/city-stories-guide/backend/internal/migrations"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
	"github.com/saas/city-stories-guide/backend/internal/platform/mock"
	"github.com/saas/city-stories-guide/backend/internal/platform/s3"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger.Setup()

	cfg, err := config.LoadFor(config.RuntimeWorker)
	if err != nil {
		return err
	}

	slog.Info("worker config loaded", "config", cfg.LogSafe())

	ctx := context.Background()

	// Initialize database connection pool
	pool, err := repository.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}

	migrationsDir, err := migrations.ResolveDir()
	if err != nil {
		pool.Close()
		return err
	}
	if err := migrations.Verify(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		return err
	}

	slog.Info("database connection established")

	// Initialize repositories
	inflationRepo := repository.NewInflationRepo(pool)
	storyRepo := repository.NewStoryRepo(pool)
	poiRepo := repository.NewPOIRepo(pool)

	// Initialize platform clients based on provider mode.
	var (
		storyGen worker.StoryGenerator
		audioGen worker.AudioGenerator
		objStore worker.ObjectStorage
		objDel   worker.ObjectDeleter
	)

	switch cfg.Provider {
	case config.ProviderModeMock:
		slog.Info("provider mode: MOCK — external integrations use in-process fakes")
		storyGen = mock.NewStoryGenerator()
		audioGen = mock.NewAudioGenerator()
		mockStore := mock.NewObjectStore()
		objStore = mockStore
		objDel = mockStore

	default: // ProviderModeReal
		slog.Info("provider mode: real — connecting to external services")
		storyGen = claude.NewClient(&claude.Config{
			APIKey:     cfg.Claude.APIKey,
			HTTPClient: newExternalHTTPClient(cfg.Claude.Timeout),
		})

		audioGen = elevenlabs.NewClient(&elevenlabs.Config{
			APIKey:     cfg.ElevenLabs.APIKey,
			HTTPClient: newExternalHTTPClient(cfg.ElevenLabs.Timeout),
		})

		s3Client, s3Err := s3.NewClient(ctx, &s3.Config{
			Endpoint:  cfg.S3.Endpoint,
			AccessKey: cfg.S3.AccessKey,
			SecretKey: cfg.S3.SecretKey,
			Bucket:    cfg.S3.Bucket,
		})
		if s3Err != nil {
			pool.Close()
			return s3Err
		}
		objStore = s3Client
		objDel = s3Client
	}

	// Create workers
	w := worker.NewInflationWorker(
		inflationRepo,
		storyRepo,
		poiRepo,
		storyGen,
		audioGen,
		objStore,
		nil,
	)

	orphanCleanupRepo := repository.NewOrphanCleanupRepo(pool)
	cw := worker.NewCleanupWorker(orphanCleanupRepo, objDel, nil)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	errCh := make(chan error, 2)
	go func() {
		slog.Info("starting inflation worker")
		errCh <- w.Start(workerCtx)
	}()
	go func() {
		slog.Info("starting cleanup worker")
		errCh <- cw.Start(workerCtx)
	}()

	select {
	case err := <-errCh:
		pool.Close()
		return err
	case sig := <-sigCh:
		start := time.Now()
		slog.Info("worker shutdown initiated", "signal", sig.String(), "timeout", (60 * time.Second).String())
		cancelWorker()

		go func() {
			sig := <-sigCh
			slog.Error("received second shutdown signal, forcing exit", "signal", sig.String())
			os.Exit(1)
		}()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		select {
		case err := <-errCh:
			if err != nil {
				pool.Close()
				return err
			}
		case <-shutdownCtx.Done():
			return shutdownCtx.Err()
		}

		pool.Close()
		slog.Info("worker shutdown complete", "duration", time.Since(start).String())
		return nil
	}
}

func newExternalHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}
