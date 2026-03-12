package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/handler"
	"github.com/saas/city-stories-guide/backend/internal/repository"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

func TestBuildRouter_PublicRouteSmoke(t *testing.T) {
	router, _ := newSmokeRouter()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "healthz", method: http.MethodGet, path: "/healthz"},
		{name: "readyz", method: http.MethodGet, path: "/readyz"},
		{name: "cities", method: http.MethodGet, path: "/api/v1/cities"},
		{name: "pois", method: http.MethodGet, path: "/api/v1/pois?city_id=1"},
		{name: "stories", method: http.MethodGet, path: "/api/v1/stories?poi_id=1"},
		{name: "auth login", method: http.MethodPost, path: "/api/v1/auth/login", body: `{"email":"user@example.com","password":"password123"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performRequest(router, tt.method, tt.path, tt.body, "")
			if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
				t.Fatalf("expected registered route for %s %s, got %d", tt.method, tt.path, rec.Code)
			}
		})
	}
}

func TestBuildRouter_ProtectedUserRouteUsesJWTAuth(t *testing.T) {
	router, auth := newSmokeRouter()

	unauthorized := performRequest(router, http.MethodGet, "/api/v1/users/me", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected JWT-protected route to reject missing credentials with 401, got %d", unauthorized.Code)
	}
	if unauthorized.Code == http.StatusNotFound || unauthorized.Code == http.StatusMethodNotAllowed {
		t.Fatalf("expected registered protected route, got %d", unauthorized.Code)
	}

	rec := performRequest(router, http.MethodGet, "/api/v1/users/me", "", "Bearer user-token")
	if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
		t.Fatalf("expected registered route, got %d", rec.Code)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected JWT-protected route to succeed with test token, got %d", rec.Code)
	}
	if got := auth.accessValidateCalls(); got != 1 {
		t.Fatalf("expected JWT validator to be called once, got %d", got)
	}
}

func TestBuildRouter_AdminRoutesRequireAdminAuth(t *testing.T) {
	router, auth := newSmokeRouter()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		authz  string
		want   int
	}{
		{
			name:   "admin cities without credentials",
			method: http.MethodGet,
			path:   "/api/v1/admin/cities",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "admin reports without credentials",
			method: http.MethodGet,
			path:   "/api/v1/admin/reports",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "admin pois without credentials",
			method: http.MethodGet,
			path:   "/api/v1/admin/pois?city_id=1",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "admin stories without credentials",
			method: http.MethodGet,
			path:   "/api/v1/admin/stories?poi_id=1",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "admin audit logs without credentials",
			method: http.MethodGet,
			path:   "/api/v1/admin/audit-logs",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "admin inflate without credentials",
			method: http.MethodPost,
			path:   "/api/v1/admin/pois/1/inflate",
			want:   http.StatusUnauthorized,
		},
		{
			name:   "admin reports rejects non admin token",
			method: http.MethodGet,
			path:   "/api/v1/admin/reports",
			authz:  "Bearer user-token",
			want:   http.StatusForbidden,
		},
		{
			name:   "admin audit logs rejects non admin token",
			method: http.MethodGet,
			path:   "/api/v1/admin/audit-logs",
			authz:  "Bearer user-token",
			want:   http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performRequest(router, tt.method, tt.path, tt.body, tt.authz)
			if rec.Code != tt.want {
				t.Fatalf("expected %d for %s %s, got %d", tt.want, tt.method, tt.path, rec.Code)
			}
			if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
				t.Fatalf("expected protected registered route for %s %s, got %d", tt.method, tt.path, rec.Code)
			}
		})
	}

	if got := auth.adminValidateCalls(); got != 2 {
		t.Fatalf("expected admin validator to be called twice for non-admin tokens, got %d", got)
	}
}

func newSmokeRouter() (*gin.Engine, *fakeAuthService) {
	gin.SetMode(gin.TestMode)

	authService := &fakeAuthService{}
	auditRepo := &fakeAuditLogRepo{}
	cityRepo := &fakeCityRepo{}
	poiRepo := &fakePOIRepo{}
	storyRepo := &fakeStoryRepo{}
	reportRepo := &fakeReportRepo{}
	userService := &fakeUserService{}
	purchaseService := &fakePurchaseService{}
	listeningRepo := &fakeListeningRepo{}
	deviceService := &fakePushNotificationService{}
	inflationRepo := &fakeInflationRepo{}
	adminStatsRepo := &fakeAdminStatsRepo{}

	router := buildRouter(routerOptions{
		Mode:              gin.TestMode,
		AllowedOrigins:    []string{"http://localhost:3000"},
		HealthHandler:     handler.NewHealthHandler(fakeDBPinger{}),
		NearbyHandler:     handler.NewNearbyHandler(fakeNearbyStoriesService{}),
		CityHandler:       handler.NewCityHandler(cityRepo, cityRepo, auditRepo),
		POIHandler:        handler.NewPOIHandler(poiRepo, auditRepo),
		StoryHandler:      handler.NewStoryHandler(storyRepo, auditRepo),
		ListeningHandler:  handler.NewListeningHandler(listeningRepo),
		ReportHandler:     handler.NewReportHandler(reportRepo, fakeReportModerationService{}, auditRepo),
		DeviceHandler:     handler.NewDeviceHandler(deviceService),
		UserHandler:       handler.NewUserHandler(userService),
		PurchaseHandler:   handler.NewPurchaseHandler(purchaseService),
		AuthHandler:       handler.NewAuthHandler(authService),
		AdminStatsHandler: handler.NewAdminStatsHandler(adminStatsRepo),
		InflationHandler:  handler.NewInflationHandler(inflationRepo, auditRepo),
		AuditLogHandler:   handler.NewAuditLogHandler(auditRepo),
		JWTValidator:      authService,
		AdminValidator:    authService,
	})

	return router, authService
}

func performRequest(router http.Handler, method, path, body, authz string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body == "" {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

type fakeDBPinger struct{}

func (fakeDBPinger) Ping(context.Context) error { return nil }

type fakeNearbyStoriesService struct{}

func (fakeNearbyStoriesService) GetNearbyStories(context.Context, float64, float64, float64, float64, float64, string, string) ([]service.StoryCandidate, error) {
	return []service.StoryCandidate{}, nil
}

type fakeAuthService struct {
	mu                sync.Mutex
	validateAccessN   int
	validateAdminAuth int
}

func (f *fakeAuthService) Register(context.Context, string, string, string) (*domain.User, *service.TokenPair, error) {
	return &domain.User{ID: "user-1"}, &service.TokenPair{AccessToken: "user-token", RefreshToken: "refresh-token", ExpiresIn: 3600}, nil
}

func (f *fakeAuthService) Login(context.Context, string, string) (*domain.User, *service.TokenPair, error) {
	return &domain.User{ID: "user-1"}, &service.TokenPair{AccessToken: "user-token", RefreshToken: "refresh-token", ExpiresIn: 3600}, nil
}

func (f *fakeAuthService) DeviceAuth(context.Context, string, string) (*domain.User, *service.TokenPair, error) {
	return &domain.User{ID: "user-1"}, &service.TokenPair{AccessToken: "user-token", RefreshToken: "refresh-token", ExpiresIn: 3600}, nil
}

func (f *fakeAuthService) RefreshTokens(context.Context, string) (*service.TokenPair, error) {
	return &service.TokenPair{AccessToken: "user-token", RefreshToken: "refresh-token", ExpiresIn: 3600}, nil
}

func (f *fakeAuthService) OAuthLogin(context.Context, domain.AuthProvider, *service.OAuthClaims) (*domain.User, *service.TokenPair, error) {
	return &domain.User{ID: "user-1"}, &service.TokenPair{AccessToken: "user-token", RefreshToken: "refresh-token", ExpiresIn: 3600}, nil
}

func (f *fakeAuthService) ValidateAccessToken(token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validateAccessN++
	if token == "user-token" || token == "admin-token" {
		return "user-1", nil
	}
	return "", errors.New("invalid token")
}

func (f *fakeAuthService) ValidateAdminToken(token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validateAdminAuth++
	switch token {
	case "admin-token":
		return "admin-1", nil
	case "user-token":
		return "", errors.New("admin access required")
	default:
		return "", errors.New("invalid token")
	}
}

func (f *fakeAuthService) accessValidateCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.validateAccessN
}

func (f *fakeAuthService) adminValidateCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.validateAdminAuth
}

type fakeCityRepo struct{}

func (fakeCityRepo) Create(context.Context, *domain.City) (*domain.City, error) {
	return &domain.City{}, nil
}
func (fakeCityRepo) GetByID(context.Context, int, bool) (*domain.City, error) {
	return &domain.City{}, nil
}
func (fakeCityRepo) GetActiveByID(context.Context, int) (*domain.City, error) {
	return &domain.City{}, nil
}
func (fakeCityRepo) GetAll(context.Context) ([]domain.City, error) { return []domain.City{}, nil }
func (fakeCityRepo) List(context.Context, domain.PageRequest, bool, repository.ListSort) (*domain.PageResponse[domain.City], error) {
	return &domain.PageResponse[domain.City]{Items: []domain.City{}}, nil
}
func (fakeCityRepo) ListActive(context.Context, domain.PageRequest) (*domain.PageResponse[domain.City], error) {
	return &domain.PageResponse[domain.City]{Items: []domain.City{}}, nil
}
func (fakeCityRepo) Update(context.Context, *domain.City) (*domain.City, error) {
	return &domain.City{}, nil
}
func (fakeCityRepo) Delete(context.Context, int) error { return nil }
func (fakeCityRepo) Restore(context.Context, int) (*domain.City, error) {
	return &domain.City{}, nil
}
func (fakeCityRepo) GetDownloadManifest(context.Context, int, string) ([]domain.DownloadManifestItem, error) {
	return []domain.DownloadManifestItem{}, nil
}

type fakePOIRepo struct{}

func (fakePOIRepo) Create(context.Context, *domain.POI) (*domain.POI, error) {
	return &domain.POI{}, nil
}
func (fakePOIRepo) GetByID(context.Context, int) (*domain.POI, error) { return &domain.POI{}, nil }
func (fakePOIRepo) GetByCityID(context.Context, int, *domain.POIStatus, *domain.POIType) ([]domain.POI, error) {
	return []domain.POI{}, nil
}
func (fakePOIRepo) ListByCityID(context.Context, int, *domain.POIStatus, *domain.POIType, domain.PageRequest, repository.ListSort) (*domain.PageResponse[domain.POI], error) {
	return &domain.PageResponse[domain.POI]{Items: []domain.POI{}}, nil
}
func (fakePOIRepo) Update(context.Context, *domain.POI) (*domain.POI, error) {
	return &domain.POI{}, nil
}
func (fakePOIRepo) Delete(context.Context, int) error { return nil }

type fakeStoryRepo struct{}

func (fakeStoryRepo) Create(context.Context, *domain.Story) (*domain.Story, error) {
	return &domain.Story{}, nil
}
func (fakeStoryRepo) GetByID(context.Context, int) (*domain.Story, error) {
	return &domain.Story{}, nil
}
func (fakeStoryRepo) GetByPOIID(context.Context, int, string, *domain.StoryStatus) ([]domain.Story, error) {
	return []domain.Story{}, nil
}
func (fakeStoryRepo) ListByPOIID(context.Context, int, string, *domain.StoryStatus, domain.PageRequest, repository.ListSort) (*domain.PageResponse[domain.Story], error) {
	return &domain.PageResponse[domain.Story]{Items: []domain.Story{}}, nil
}
func (fakeStoryRepo) Update(context.Context, *domain.Story) (*domain.Story, error) {
	return &domain.Story{}, nil
}
func (fakeStoryRepo) Delete(context.Context, int) error { return nil }

type fakeListeningRepo struct{}

func (fakeListeningRepo) CreateOrUpdate(context.Context, string, int, bool, *float64, *float64) (*domain.UserListening, error) {
	return &domain.UserListening{}, nil
}
func (fakeListeningRepo) ListByUserID(context.Context, string, domain.PageRequest) (*domain.PageResponse[domain.UserListening], error) {
	return &domain.PageResponse[domain.UserListening]{Items: []domain.UserListening{}}, nil
}

type fakeReportRepo struct{}

func (fakeReportRepo) Create(context.Context, int, string, domain.ReportType, *string, *float64, *float64) (*domain.Report, error) {
	return &domain.Report{}, nil
}
func (fakeReportRepo) GetByID(context.Context, int) (*domain.Report, error) {
	return &domain.Report{}, nil
}
func (fakeReportRepo) List(context.Context, string, domain.PageRequest) (*domain.PageResponse[domain.Report], error) {
	return &domain.PageResponse[domain.Report]{Items: []domain.Report{}}, nil
}
func (fakeReportRepo) ListAdmin(context.Context, string, domain.PageRequest, repository.ListSort) (*domain.PageResponse[domain.AdminReportListItem], error) {
	return &domain.PageResponse[domain.AdminReportListItem]{Items: []domain.AdminReportListItem{}}, nil
}
func (fakeReportRepo) UpdateStatus(context.Context, int, domain.ReportStatus) (*domain.Report, error) {
	return &domain.Report{}, nil
}
func (fakeReportRepo) GetByPOIID(context.Context, int) ([]domain.Report, error) {
	return []domain.Report{}, nil
}

type fakeReportModerationService struct{}

func (fakeReportModerationService) DisableStory(context.Context, int) (*domain.ModeratedReportResult, error) {
	return &domain.ModeratedReportResult{}, nil
}

type fakeInflationRepo struct{}

func (fakeInflationRepo) Create(context.Context, *domain.InflationJob) (*domain.InflationJob, error) {
	return &domain.InflationJob{ID: 1, CreatedAt: time.Now()}, nil
}
func (fakeInflationRepo) GetByPOIID(context.Context, int) ([]domain.InflationJob, error) {
	return []domain.InflationJob{}, nil
}
func (fakeInflationRepo) CountActiveByPOIID(context.Context, int) (int, error) { return 0, nil }

type fakeUserService struct{}

func (fakeUserService) ScheduleDeletion(context.Context, string) error { return nil }
func (fakeUserService) RestoreAccount(context.Context, string) error   { return nil }
func (fakeUserService) GetByID(context.Context, string) (*domain.User, error) {
	return &domain.User{ID: "user-1"}, nil
}

type fakePurchaseService struct{}

func (fakePurchaseService) VerifyAndCreate(context.Context, *service.VerifyPurchaseRequest) (*domain.Purchase, error) {
	return &domain.Purchase{}, nil
}
func (fakePurchaseService) GetStatus(context.Context, string) (*service.PurchaseStatus, error) {
	return &service.PurchaseStatus{}, nil
}

type fakePushNotificationService struct{}

func (fakePushNotificationService) RegisterDeviceToken(context.Context, string, string, domain.DevicePlatform) (*domain.DeviceToken, error) {
	return &domain.DeviceToken{}, nil
}
func (fakePushNotificationService) UnregisterDeviceToken(context.Context, string) error { return nil }
func (fakePushNotificationService) GetUserDeviceTokens(context.Context, string) ([]domain.DeviceToken, error) {
	return []domain.DeviceToken{}, nil
}

type fakeAdminStatsRepo struct{}

func (fakeAdminStatsRepo) Get(context.Context) (*repository.AdminStats, error) {
	return &repository.AdminStats{}, nil
}

type fakeAuditLogRepo struct{}

func (fakeAuditLogRepo) Insert(context.Context, *domain.AuditLog) error { return nil }
func (fakeAuditLogRepo) List(context.Context, repository.AuditLogFilter, domain.PageRequest, repository.ListSort) (*domain.PageResponse[domain.AuditLog], error) {
	return &domain.PageResponse[domain.AuditLog]{Items: []domain.AuditLog{}}, nil
}
