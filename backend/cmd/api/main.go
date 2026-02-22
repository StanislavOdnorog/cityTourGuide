package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/config"
	"github.com/saas/city-stories-guide/backend/internal/handler"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
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

	log.Printf("Config loaded: %s", cfg.LogSafe())

	// Create context that listens for OS shutdown signals
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
	poiRepo := repository.NewPOIRepo(pool)
	storyRepo := repository.NewStoryRepo(pool)
	listeningRepo := repository.NewListeningRepo(pool)

	// Initialize services
	nearbyService := service.NewNearbyService(poiRepo, storyRepo, listeningRepo)

	// Initialize handlers
	nearbyHandler := handler.NewNearbyHandler(nearbyService)

	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	v1.GET("/nearby-stories", nearbyHandler.GetNearbyStories)

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting server on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		stop()
		log.Println("Shutting down server...")
	case err := <-errCh:
		return err
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Println("Server stopped")
	return nil
}
