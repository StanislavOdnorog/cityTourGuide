package service

import (
	"context"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// mockUserRepo is a mock for AuthUserRepository.
type mockUserRepo struct {
	users         map[string]*domain.User
	emailIndex    map[string]*domain.User
	createErr     error
	getByIDErr    error
	getByEmailErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:      make(map[string]*domain.User),
		emailIndex: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *domain.User) (*domain.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	u := *user
	u.ID = "test-uuid-123"
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	m.users[u.ID] = &u
	if u.Email != nil {
		m.emailIndex[*u.Email] = &u
	}
	return &u, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	u, ok := m.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	u, ok := m.emailIndex[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) CreateAnonymous(_ context.Context, deviceID string, languagePref string) (*domain.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	u := &domain.User{
		ID:           deviceID,
		AuthProvider: domain.AuthProviderEmail,
		LanguagePref: languagePref,
		IsAnonymous:  true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[deviceID] = u
	return u, nil
}

func testConfig() AuthConfig {
	return AuthConfig{
		Secret:     "test-secret-key-for-testing-only",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	}
}

func TestAuthService_Register_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	user, tokens, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("user should not be nil")
	}
	if user.Email == nil || *user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", user.Email)
	}
	if user.IsAnonymous {
		t.Error("user should not be anonymous")
	}
	if tokens == nil {
		t.Fatal("tokens should not be nil")
	}
	if tokens.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("refresh token should not be empty")
	}
	if tokens.ExpiresIn != 900 { // 15 min = 900 sec
		t.Errorf("expected expires_in 900, got %d", tokens.ExpiresIn)
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, _, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("first register should succeed: %v", err)
	}

	_, _, err = svc.Register(context.Background(), "test@example.com", "otherpass", "Other")
	if err != ErrEmailAlreadyExists {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, _, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	user, tokens, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if user == nil || tokens == nil {
		t.Fatal("user and tokens should not be nil")
	}
	if tokens.AccessToken == "" {
		t.Error("access token should not be empty")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, _, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, _, err = svc.Login(context.Background(), "test@example.com", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, _, err := svc.Login(context.Background(), "unknown@example.com", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_DeviceAuth_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	user, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != "device-uuid-123" {
		t.Errorf("expected device ID as user ID, got %s", user.ID)
	}
	if !user.IsAnonymous {
		t.Error("device user should be anonymous")
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Error("tokens should not be nil/empty")
	}
}

func TestAuthService_DeviceAuth_DefaultLanguage(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	user, _, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.LanguagePref != "en" {
		t.Errorf("expected default language en, got %s", user.LanguagePref)
	}
}

func TestAuthService_RefreshTokens_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	newTokens, err := svc.RefreshTokens(context.Background(), tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	if newTokens.AccessToken == "" {
		t.Error("new access token should not be empty")
	}
	if newTokens.RefreshToken == "" {
		t.Error("new refresh token should not be empty")
	}
}

func TestAuthService_RefreshTokens_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, err := svc.RefreshTokens(context.Background(), "invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_RefreshTokens_AccessTokenRejected(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	// Try to use access token as refresh token — should be rejected
	_, err = svc.RefreshTokens(context.Background(), tokens.AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken when using access token as refresh, got %v", err)
	}
}

func TestAuthService_RefreshTokens_UserDeleted(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	// Simulate user deletion
	delete(repo.users, "device-uuid-123")

	_, err = svc.RefreshTokens(context.Background(), tokens.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for deleted user, got %v", err)
	}
}

func TestAuthService_ValidateAccessToken_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	userID, err := svc.ValidateAccessToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if userID != "device-uuid-123" {
		t.Errorf("expected user ID device-uuid-123, got %s", userID)
	}
}

func TestAuthService_ValidateAccessToken_Invalid(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, err := svc.ValidateAccessToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_ValidateAccessToken_RefreshTokenRejected(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	// Try to use refresh token as access token — should be rejected
	_, err = svc.ValidateAccessToken(tokens.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken when using refresh as access, got %v", err)
	}
}

func TestAuthService_ValidateAccessToken_WrongSecret(t *testing.T) {
	repo := newMockUserRepo()
	svc1 := NewAuthService(repo, testConfig())
	svc2 := NewAuthService(repo, AuthConfig{
		Secret:     "different-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})

	_, tokens, err := svc1.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	// Validate with different secret — should fail
	_, err = svc2.ValidateAccessToken(tokens.AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for wrong secret, got %v", err)
	}
}

func TestAuthService_TokenPair_BothTokensDifferent(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, tokens, err := svc.DeviceAuth(context.Background(), "device-uuid-123", "en")
	if err != nil {
		t.Fatalf("device auth failed: %v", err)
	}

	if tokens.AccessToken == tokens.RefreshToken {
		t.Error("access and refresh tokens should be different")
	}
}

func TestAuthService_Register_PasswordHashed(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testConfig())

	_, _, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	user := repo.emailIndex["test@example.com"]
	if user.PasswordHash == nil {
		t.Fatal("password hash should not be nil")
	}
	if *user.PasswordHash == "password123" {
		t.Error("password should be hashed, not stored in plain text")
	}
}
