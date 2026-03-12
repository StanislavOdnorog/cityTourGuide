package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/saas/city-stories-guide/backend/internal/handler"
	"github.com/saas/city-stories-guide/backend/internal/middleware"
)

type routerOptions struct {
	Mode           string
	AllowedOrigins []string

	HealthHandler     *handler.HealthHandler
	NearbyHandler     *handler.NearbyHandler
	CityHandler       *handler.CityHandler
	POIHandler        *handler.POIHandler
	StoryHandler      *handler.StoryHandler
	ListeningHandler  *handler.ListeningHandler
	ReportHandler     *handler.ReportHandler
	DeviceHandler     *handler.DeviceHandler
	UserHandler       *handler.UserHandler
	PurchaseHandler   *handler.PurchaseHandler
	AuthHandler       *handler.AuthHandler
	AdminStatsHandler *handler.AdminStatsHandler
	InflationHandler  *handler.InflationHandler
	AuditLogHandler   *handler.AuditLogHandler

	JWTValidator   middleware.TokenValidator
	AdminValidator middleware.AdminTokenValidator

	AuthRateLimiter   *middleware.RateLimiter
	APIRateLimiter    *middleware.RateLimiter
	NearbyRateLimiter *middleware.RateLimiter
}

func buildRouter(opts routerOptions) *gin.Engine {
	gin.SetMode(opts.Mode)
	r := gin.New()

	authRateLimiter, apiRateLimiter, nearbyRateLimiter := routerRateLimiters(opts)

	r.Use(middleware.Metrics())
	r.Use(gin.Recovery())
	r.Use(middleware.LimitRequestBodySize(maxRequestBodySize))
	r.Use(middleware.TraceIDMiddleware())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: opts.AllowedOrigins,
	}))
	r.Use(middleware.ValidateGPSParams())

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/healthz", opts.HealthHandler.Healthz)
	r.GET("/readyz", opts.HealthHandler.Readyz)

	handler.RegisterSwagger(r)

	v1 := r.Group("/api/v1")
	v1.Use(apiRateLimiter.Middleware())
	v1.GET("/nearby-stories", nearbyRateLimiter.Middleware(), opts.NearbyHandler.GetNearbyStories)
	v1.GET("/cities", opts.CityHandler.ListCities)
	v1.GET("/cities/:id", opts.CityHandler.GetCity)
	v1.GET("/cities/:id/download-manifest", opts.CityHandler.GetDownloadManifest)
	v1.GET("/pois", opts.POIHandler.ListPOIs)
	v1.GET("/pois/:id", opts.POIHandler.GetPOI)
	v1.GET("/stories", opts.StoryHandler.ListStories)
	v1.GET("/stories/:id", opts.StoryHandler.GetStory)
	v1.GET("/listenings", opts.ListeningHandler.ListListenings)
	v1.POST("/listenings", opts.ListeningHandler.TrackListening)
	v1.POST("/reports", opts.ReportHandler.CreateReport)
	v1.POST("/device-tokens", opts.DeviceHandler.RegisterDeviceToken)
	v1.DELETE("/device-tokens", opts.DeviceHandler.UnregisterDeviceToken)

	users := v1.Group("/users")
	users.Use(middleware.JWTAuth(opts.JWTValidator))
	users.GET("/me", opts.UserHandler.GetMe)
	users.DELETE("/me", opts.UserHandler.DeleteAccount)
	users.POST("/me/restore", opts.UserHandler.RestoreAccount)

	purchases := v1.Group("/purchases")
	purchases.Use(middleware.JWTAuth(opts.JWTValidator))
	purchases.POST("/verify", opts.PurchaseHandler.VerifyPurchase)
	purchases.GET("/status", opts.PurchaseHandler.GetStatus)

	auth := v1.Group("/auth")
	auth.Use(authRateLimiter.Middleware())
	auth.POST("/register", opts.AuthHandler.Register)
	auth.POST("/login", opts.AuthHandler.Login)
	auth.POST("/device", opts.AuthHandler.DeviceAuth)
	auth.POST("/refresh", opts.AuthHandler.Refresh)
	auth.POST("/google", opts.AuthHandler.GoogleAuth)
	auth.POST("/apple", opts.AuthHandler.AppleAuth)

	admin := v1.Group("/admin")
	admin.Use(middleware.AdminAuth(opts.AdminValidator))
	admin.GET("/cities", opts.CityHandler.ListAdminCities)
	admin.POST("/cities", opts.CityHandler.CreateCity)
	admin.PUT("/cities/:id", opts.CityHandler.UpdateCity)
	admin.DELETE("/cities/:id", opts.CityHandler.DeleteCity)
	admin.POST("/cities/:id/restore", opts.CityHandler.RestoreCity)
	admin.GET("/pois", opts.POIHandler.ListAdminPOIs)
	admin.POST("/pois", opts.POIHandler.CreatePOI)
	admin.PUT("/pois/:id", opts.POIHandler.UpdatePOI)
	admin.DELETE("/pois/:id", opts.POIHandler.DeletePOI)
	admin.GET("/stories", opts.StoryHandler.ListAdminStories)
	admin.POST("/stories", opts.StoryHandler.CreateStory)
	admin.PUT("/stories/:id", opts.StoryHandler.UpdateStory)
	admin.DELETE("/stories/:id", opts.StoryHandler.DeleteStory)
	admin.GET("/stats", opts.AdminStatsHandler.Get)
	admin.GET("/reports", opts.ReportHandler.ListReports)
	admin.PUT("/reports/:id", opts.ReportHandler.UpdateReportStatus)
	admin.POST("/reports/:id/disable-story", opts.ReportHandler.DisableStory)
	admin.GET("/pois/:id/reports", opts.ReportHandler.ListByPOI)
	admin.POST("/pois/:id/inflate", opts.InflationHandler.TriggerInflation)
	admin.GET("/pois/:id/inflation-jobs", opts.InflationHandler.ListByPOI)
	admin.GET("/audit-logs", opts.AuditLogHandler.List)

	return r
}

func routerRateLimiters(opts routerOptions) (auth, api, nearby *middleware.RateLimiter) {
	auth = opts.AuthRateLimiter
	if auth == nil {
		auth = middleware.NewRateLimiter(5, time.Minute)
	}

	api = opts.APIRateLimiter
	if api == nil {
		api = middleware.NewRateLimiter(60, time.Minute)
	}

	nearby = opts.NearbyRateLimiter
	if nearby == nil {
		nearby = middleware.NewRateLimiter(10, time.Minute)
	}

	return auth, api, nearby
}

func limitRequestBodySize(limit int64) gin.HandlerFunc {
	return middleware.LimitRequestBodySize(limit)
}
