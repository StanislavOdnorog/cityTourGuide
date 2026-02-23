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
	"github.com/saas/city-stories-guide/backend/internal/middleware"
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
	cityRepo := repository.NewCityRepo(pool)
	poiRepo := repository.NewPOIRepo(pool)
	storyRepo := repository.NewStoryRepo(pool)
	listeningRepo := repository.NewListeningRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	reportRepo := repository.NewReportRepo(pool)
	inflationRepo := repository.NewInflationRepo(pool)

	// Initialize services
	nearbyService := service.NewNearbyService(poiRepo, storyRepo, listeningRepo)
	authService := service.NewAuthService(userRepo, service.AuthConfig{
		Secret:     cfg.JWT.Secret,
		AccessTTL:  cfg.JWT.AccessTTL,
		RefreshTTL: cfg.JWT.RefreshTTL,
	})

	// Initialize handlers
	nearbyHandler := handler.NewNearbyHandler(nearbyService)
	cityHandler := handler.NewCityHandler(cityRepo, storyRepo)
	poiHandler := handler.NewPOIHandler(poiRepo)
	storyHandler := handler.NewStoryHandler(storyRepo)
	authHandler := handler.NewAuthHandler(authService)
	listeningHandler := handler.NewListeningHandler(listeningRepo)
	reportHandler := handler.NewReportHandler(reportRepo)
	inflationHandler := handler.NewInflationHandler(inflationRepo)

	// Rate limiters
	authRateLimiter := middleware.NewRateLimiter(5, time.Minute)    // 5 req/min for auth
	apiRateLimiter := middleware.NewRateLimiter(60, time.Minute)    // 60 req/min general
	nearbyRateLimiter := middleware.NewRateLimiter(10, time.Minute) // 10 req/min for AI-dependent

	gin.SetMode(cfg.Server.Mode)
	r := gin.New() // use gin.New() instead of gin.Default() to control middleware

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: cfg.Server.AllowedOrigins,
	}))
	r.Use(middleware.ValidateGPSParams())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes — public
	v1 := r.Group("/api/v1")
	v1.Use(apiRateLimiter.Middleware())
	v1.GET("/nearby-stories", nearbyRateLimiter.Middleware(), nearbyHandler.GetNearbyStories)
	v1.GET("/cities", cityHandler.ListCities)
	v1.GET("/cities/:id", cityHandler.GetCity)
	v1.GET("/cities/:id/download-manifest", cityHandler.GetDownloadManifest)
	v1.GET("/pois", poiHandler.ListPOIs)
	v1.GET("/pois/:id", poiHandler.GetPOI)
	v1.GET("/stories", storyHandler.ListStories)
	v1.GET("/stories/:id", storyHandler.GetStory)
	v1.POST("/listenings", listeningHandler.TrackListening)
	v1.POST("/reports", reportHandler.CreateReport)

	// Auth routes with stricter rate limiting
	auth := v1.Group("/auth")
	auth.Use(authRateLimiter.Middleware())
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/device", authHandler.DeviceAuth)
	auth.POST("/refresh", authHandler.Refresh)

	// API v1 routes — admin (protected with JWT + admin claim)
	admin := v1.Group("/admin")
	admin.Use(middleware.AdminAuth(authService))
	admin.POST("/cities", cityHandler.CreateCity)
	admin.PUT("/cities/:id", cityHandler.UpdateCity)
	admin.DELETE("/cities/:id", cityHandler.DeleteCity)
	admin.POST("/pois", poiHandler.CreatePOI)
	admin.PUT("/pois/:id", poiHandler.UpdatePOI)
	admin.DELETE("/pois/:id", poiHandler.DeletePOI)
	admin.POST("/stories", storyHandler.CreateStory)
	admin.PUT("/stories/:id", storyHandler.UpdateStory)
	admin.DELETE("/stories/:id", storyHandler.DeleteStory)
	admin.GET("/reports", reportHandler.ListReports)
	admin.PUT("/reports/:id", reportHandler.UpdateReportStatus)
	admin.GET("/pois/:id/reports", reportHandler.ListByPOI)
	admin.POST("/pois/:id/inflate", inflationHandler.TriggerInflation)
	admin.GET("/pois/:id/inflation-jobs", inflationHandler.ListByPOI)

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
