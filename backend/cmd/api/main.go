package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/config"
	"github.com/saas/city-stories-guide/backend/internal/handler"
	"github.com/saas/city-stories-guide/backend/internal/logger"
	"github.com/saas/city-stories-guide/backend/internal/middleware"
	"github.com/saas/city-stories-guide/backend/internal/platform/fcm"
	"github.com/saas/city-stories-guide/backend/internal/platform/oauth"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize structured JSON logging before anything else.
	logger.Setup()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.Info("config loaded", "config", cfg.LogSafe())

	// Create context that listens for OS shutdown signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize database connection pool
	pool, err := repository.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	slog.Info("database connection established")

	// Initialize repositories
	cityRepo := repository.NewCityRepo(pool)
	poiRepo := repository.NewPOIRepo(pool)
	storyRepo := repository.NewStoryRepo(pool)
	listeningRepo := repository.NewListeningRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	reportRepo := repository.NewReportRepo(pool)
	inflationRepo := repository.NewInflationRepo(pool)
	purchaseRepo := repository.NewPurchaseRepo(pool)

	deviceTokenRepo := repository.NewDeviceTokenRepo(pool)
	pushNotifRepo := repository.NewPushNotificationRepo(pool)

	// Initialize FCM client (optional — nil if not configured)
	fcmClient, err := fcm.NewClient(ctx, &fcm.Config{
		CredentialsJSON: cfg.FCM.CredentialsJSON,
	})
	if err != nil {
		slog.Error("failed to initialize FCM client", "error", err)
		// Non-fatal: push notifications will be disabled
	}
	if fcmClient != nil {
		slog.Info("FCM push notifications enabled")
	}

	// Initialize services
	nearbyService := service.NewNearbyService(poiRepo, storyRepo, listeningRepo)
	authService := service.NewAuthService(userRepo, service.AuthConfig{
		Secret:     cfg.JWT.Secret,
		AccessTTL:  cfg.JWT.AccessTTL,
		RefreshTTL: cfg.JWT.RefreshTTL,
	})
	purchaseService := service.NewPurchaseService(purchaseRepo)
	pushNotifService := service.NewPushNotificationService(
		deviceTokenRepo, pushNotifRepo, fcmClient,
		service.PushNotificationConfig{GeoMaxPerDay: 2, ContentMaxPerWeek: 1},
	)

	// Initialize handlers
	nearbyHandler := handler.NewNearbyHandler(nearbyService)
	cityHandler := handler.NewCityHandler(cityRepo, storyRepo)
	poiHandler := handler.NewPOIHandler(poiRepo)
	storyHandler := handler.NewStoryHandler(storyRepo)
	authHandler := handler.NewAuthHandler(authService)
	purchaseHandler := handler.NewPurchaseHandler(purchaseService)
	healthHandler := handler.NewHealthHandler(pool)

	// Set up OAuth verifiers (only if configured)
	if cfg.OAuth.GoogleClientID != "" {
		googleVerifier := oauth.NewGoogleVerifier(cfg.OAuth.GoogleClientID, nil)
		authHandler.SetGoogleVerifier(oauth.NewGoogleHandlerAdapter(googleVerifier))
		slog.Info("Google Sign-In enabled")
	}
	if cfg.OAuth.AppleClientID != "" {
		appleVerifier := oauth.NewAppleVerifier(oauth.AppleConfig{
			ClientID:   cfg.OAuth.AppleClientID,
			TeamID:     cfg.OAuth.AppleTeamID,
			KeyID:      cfg.OAuth.AppleKeyID,
			PrivateKey: cfg.OAuth.ApplePrivateKey,
		}, nil)
		authHandler.SetAppleVerifier(oauth.NewAppleHandlerAdapter(appleVerifier))
		slog.Info("Apple Sign-In enabled")
	}

	listeningHandler := handler.NewListeningHandler(listeningRepo)
	reportHandler := handler.NewReportHandler(reportRepo)
	inflationHandler := handler.NewInflationHandler(inflationRepo)
	deviceHandler := handler.NewDeviceHandler(pushNotifService)

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

	// Health & readiness
	r.GET("/healthz", healthHandler.Healthz)
	r.GET("/readyz", healthHandler.Readyz)

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
	v1.POST("/device-tokens", deviceHandler.RegisterDeviceToken)
	v1.DELETE("/device-tokens", deviceHandler.UnregisterDeviceToken)

	// Purchase routes (protected with JWT)
	purchases := v1.Group("/purchases")
	purchases.Use(middleware.JWTAuth(authService))
	purchases.POST("/verify", purchaseHandler.VerifyPurchase)
	purchases.GET("/status", purchaseHandler.GetStatus)

	// Auth routes with stricter rate limiting
	auth := v1.Group("/auth")
	auth.Use(authRateLimiter.Middleware())
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/device", authHandler.DeviceAuth)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/google", authHandler.GoogleAuth)
	auth.POST("/apple", authHandler.AppleAuth)

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
		slog.Info("starting server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		stop()
		slog.Info("shutting down server")
	case err := <-errCh:
		return err
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("server stopped")
	return nil
}
