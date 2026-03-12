package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas/city-stories-guide/backend/internal/config"
	"github.com/saas/city-stories-guide/backend/internal/handler"
	"github.com/saas/city-stories-guide/backend/internal/logger"
	_ "github.com/saas/city-stories-guide/backend/internal/metrics" // register business metrics
	"github.com/saas/city-stories-guide/backend/internal/migrations"
	"github.com/saas/city-stories-guide/backend/internal/platform/fcm"
	"github.com/saas/city-stories-guide/backend/internal/platform/oauth"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

const maxRequestBodySize = 1 << 20

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize structured JSON logging before anything else.
	logger.Setup()

	cfg, err := config.LoadFor(config.RuntimeAPI)
	if err != nil {
		return err
	}

	slog.Info("config loaded", "config", cfg.LogSafe())

	ctx := context.Background()

	// Initialize database connection pool
	pool, err := repository.NewPool(ctx, cfg.Database.URL, repository.PoolConfig{
		MaxConns:          cfg.Database.MaxConns,
		MinConns:          cfg.Database.MinConns,
		MaxConnLifetime:   cfg.Database.MaxConnLifetime,
		MaxConnIdleTime:   cfg.Database.MaxConnIdleTime,
		HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
	})
	if err != nil {
		return err
	}

	if err := verifyDatabaseSchema(ctx, pool); err != nil {
		pool.Close()
		return err
	}

	slog.Info("database connection established")

	// Initialize repositories
	cityRepo := repository.NewCityRepo(pool)
	poiRepo := repository.NewPOIRepo(pool)
	storyRepo := repository.NewStoryRepo(pool)
	listeningRepo := repository.NewListeningRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	reportRepo := repository.NewReportRepo(pool)
	adminStatsRepo := repository.NewCachedAdminStatsRepo(
		repository.NewAdminStatsRepo(pool),
		30*time.Second,
	)
	inflationRepo := repository.NewInflationRepo(pool)
	purchaseRepo := repository.NewPurchaseRepo(pool)
	auditLogRepo := repository.NewAuditLogRepo(pool)

	orphanCleanupRepo := repository.NewOrphanCleanupRepo(pool)

	deviceTokenRepo := repository.NewDeviceTokenRepo(pool)
	pushNotifRepo := repository.NewPushNotificationRepo(pool)

	// Initialize FCM client (optional — nil if not configured)
	var fcmReadinessErr error
	fcmClient, err := fcm.NewClient(ctx, &fcm.Config{
		CredentialsJSON: cfg.FCM.CredentialsJSON,
		HTTPClient:      newExternalHTTPClient(cfg.FCM.Timeout),
	})
	if err != nil {
		fcmReadinessErr = err
		slog.Error("failed to initialize FCM client", "error", err)
		// Non-fatal: push notifications will be disabled
	}
	if fcmClient != nil {
		slog.Info("FCM push notifications enabled")
	}

	// Initialize services
	nearbyService := service.NewNearbyService(poiRepo, storyRepo)
	authService := service.NewAuthService(userRepo, service.AuthConfig{
		Secret:     cfg.JWT.Secret,
		AccessTTL:  cfg.JWT.AccessTTL,
		RefreshTTL: cfg.JWT.RefreshTTL,
	})
	purchaseService := service.NewPurchaseService(purchaseRepo)
	userService := service.NewUserService(userRepo)
	var fcmSender fcm.Sender
	if fcmClient != nil {
		fcmSender = fcmClient
	}
	pushNotifService := service.NewPushNotificationService(
		deviceTokenRepo, pushNotifRepo, fcmSender,
		service.PushNotificationConfig{GeoMaxPerDay: 2, ContentMaxPerWeek: 1},
	)
	reportModerationService := service.NewReportModerationService(reportRepo)
	audioURLCollector := service.NewAudioURLCollector(pool)
	audioCleanupService := service.NewAudioCleanupService(
		pool, storyRepo, poiRepo, cityRepo, audioURLCollector, orphanCleanupRepo,
	)

	// Initialize handlers
	nearbyHandler := handler.NewNearbyHandler(nearbyService)
	cityHandler := handler.NewCityHandler(cityRepo, storyRepo, auditLogRepo, audioCleanupService)
	poiHandler := handler.NewPOIHandler(poiRepo, auditLogRepo, audioCleanupService)
	storyHandler := handler.NewStoryHandler(storyRepo, auditLogRepo, audioCleanupService)
	authHandler := handler.NewAuthHandler(authService)
	purchaseHandler := handler.NewPurchaseHandler(purchaseService)
	healthHandler := handler.NewHealthHandler(pool, handler.ReadinessCheck{
		Name:     "fcm",
		Required: false,
		Check: func(context.Context) error {
			return fcmReadinessErr
		},
	})

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

	userHandler := handler.NewUserHandler(userService)
	listeningHandler := handler.NewListeningHandler(listeningRepo)
	reportHandler := handler.NewReportHandler(reportRepo, reportModerationService, auditLogRepo)
	adminStatsHandler := handler.NewAdminStatsHandler(adminStatsRepo)
	inflationHandler := handler.NewInflationHandler(inflationRepo, auditLogRepo)
	auditLogHandler := handler.NewAuditLogHandler(auditLogRepo)
	deviceHandler := handler.NewDeviceHandler(pushNotifService)

	r := buildRouter(routerOptions{
		Mode:              cfg.Server.Mode,
		AllowedOrigins:    cfg.Server.AllowedOrigins,
		HealthHandler:     healthHandler,
		NearbyHandler:     nearbyHandler,
		CityHandler:       cityHandler,
		POIHandler:        poiHandler,
		StoryHandler:      storyHandler,
		ListeningHandler:  listeningHandler,
		ReportHandler:     reportHandler,
		DeviceHandler:     deviceHandler,
		UserHandler:       userHandler,
		PurchaseHandler:   purchaseHandler,
		AuthHandler:       authHandler,
		AdminStatsHandler: adminStatsHandler,
		InflationHandler:  inflationHandler,
		AuditLogHandler:   auditLogHandler,
		JWTValidator:      authService,
		AdminValidator:    authService,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           r,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		pool.Close()
		return err
	}

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	cleanup := []func() error{
		func() error {
			pool.Close()
			return nil
		},
	}

	return serveWithGracefulShutdown(
		srv,
		listener,
		sigCh,
		healthHandler.SetShuttingDown,
		cfg.Server.ShutdownTimeout,
		os.Exit,
		cleanup...,
	)
}

func newExternalHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func serveWithGracefulShutdown(
	srv *http.Server,
	listener net.Listener,
	sigCh <-chan os.Signal,
	setShuttingDown func(bool),
	shutdownTimeout time.Duration,
	forceExit func(int),
	cleanup ...func() error,
) error {
	errCh := make(chan error, 1)
	shutdownDone := make(chan struct{})

	go func() {
		slog.Info("starting server", "addr", listener.Addr().String())
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		start := time.Now()
		if setShuttingDown != nil {
			setShuttingDown(true)
		}

		go func() {
			select {
			case sig := <-sigCh:
				slog.Error("received second shutdown signal, forcing exit", "signal", sig.String())
				forceExit(1)
			case <-shutdownDone:
			}
		}()

		slog.Info("shutdown initiated", "signal", sig.String(), "timeout", shutdownTimeout.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			close(shutdownDone)
			return err
		}

		for _, closeFn := range cleanup {
			if closeFn == nil {
				continue
			}
			if err := closeFn(); err != nil {
				close(shutdownDone)
				return err
			}
		}

		close(shutdownDone)
		slog.Info("shutdown complete", "duration", time.Since(start).String())
		return nil
	}
}

func verifyDatabaseSchema(ctx context.Context, pool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}) error {
	migrationsDir, err := migrations.ResolveDir()
	if err != nil {
		return err
	}

	return migrations.Verify(ctx, pool, migrationsDir)
}
