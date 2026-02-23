package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// AuthUserRepository defines the user repository methods needed by AuthService.
type AuthUserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error)
	CreateAnonymous(ctx context.Context, deviceID, languagePref string) (*domain.User, error)
}

// OAuthClaims holds common claims extracted from an OAuth provider's token.
type OAuthClaims struct {
	Sub   string
	Email string
	Name  string
}

// AuthConfig holds JWT configuration for the auth service.
type AuthConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// TokenPair holds access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// AuthService handles authentication logic.
type AuthService struct {
	repo   AuthUserRepository
	config AuthConfig
}

// Sentinel errors for auth operations.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// NewAuthService creates a new AuthService.
func NewAuthService(repo AuthUserRepository, config AuthConfig) *AuthService {
	return &AuthService{
		repo:   repo,
		config: config,
	}
}

// Register creates a new user with email and password.
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*domain.User, *TokenPair, error) {
	// Check if email is already taken
	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, nil, fmt.Errorf("auth: check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: hash password: %w", err)
	}

	hashStr := string(hash)
	user := &domain.User{
		Email:        &email,
		Name:         &name,
		PasswordHash: &hashStr,
		AuthProvider: domain.AuthProviderEmail,
		LanguagePref: "en",
		IsAnonymous:  false,
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: create user: %w", err)
	}

	tokens, err := s.generateTokenPair(created.ID, created.IsAdmin)
	if err != nil {
		return nil, nil, err
	}

	return created, tokens, nil
}

// Login authenticates a user with email and password.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, *TokenPair, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("auth: login: %w", err)
	}

	if user.PasswordHash == nil {
		return nil, nil, ErrInvalidCredentials
	}

	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); bcryptErr != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokenPair(user.ID, user.IsAdmin)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// DeviceAuth creates or retrieves an anonymous user by device UUID.
func (s *AuthService) DeviceAuth(ctx context.Context, deviceID, language string) (*domain.User, *TokenPair, error) {
	if language == "" {
		language = "en"
	}

	user, err := s.repo.CreateAnonymous(ctx, deviceID, language)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: device auth: %w", err)
	}

	tokens, err := s.generateTokenPair(user.ID, user.IsAdmin)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshTokens validates a refresh token and returns a new token pair.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims["type"] != "refresh" {
		return nil, ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return nil, ErrInvalidToken
	}

	// Verify user still exists and get current admin status
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("auth: refresh: %w", err)
	}

	return s.generateTokenPair(user.ID, user.IsAdmin)
}

// OAuthLogin finds or creates a user for an OAuth provider and returns a JWT pair.
// If the user already exists (matched by provider+providerID), it returns the existing user.
// If the user is new, it creates one with the provider's claims.
func (s *AuthService) OAuthLogin(ctx context.Context, provider domain.AuthProvider, claims *OAuthClaims) (*domain.User, *TokenPair, error) {
	// Try to find existing user by provider + provider_id
	user, err := s.repo.GetByProviderID(ctx, provider, claims.Sub)
	if err == nil {
		// Existing user found — generate tokens and return
		tokens, tokenErr := s.generateTokenPair(user.ID, user.IsAdmin)
		if tokenErr != nil {
			return nil, nil, tokenErr
		}
		return user, tokens, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, nil, fmt.Errorf("auth: oauth lookup: %w", err)
	}

	// New user — create with OAuth provider info
	newUser := &domain.User{
		AuthProvider: provider,
		ProviderID:   &claims.Sub,
		LanguagePref: "en",
		IsAnonymous:  false,
	}
	if claims.Email != "" {
		newUser.Email = &claims.Email
	}
	if claims.Name != "" {
		newUser.Name = &claims.Name
	}

	created, err := s.repo.Create(ctx, newUser)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: oauth create user: %w", err)
	}

	tokens, err := s.generateTokenPair(created.ID, created.IsAdmin)
	if err != nil {
		return nil, nil, err
	}

	return created, tokens, nil
}

// ValidateAccessToken validates an access token and returns the user ID.
func (s *AuthService) ValidateAccessToken(tokenString string) (string, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return "", ErrInvalidToken
	}

	if claims["type"] != "access" {
		return "", ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", ErrInvalidToken
	}

	return userID, nil
}

// ValidateAdminToken validates an access token and returns the user ID only if the user is admin.
func (s *AuthService) ValidateAdminToken(tokenString string) (string, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return "", ErrInvalidToken
	}

	if claims["type"] != "access" {
		return "", ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", ErrInvalidToken
	}

	admin, _ := claims["admin"].(bool)
	if !admin {
		return "", fmt.Errorf("admin access required")
	}

	return userID, nil
}

func (s *AuthService) generateTokenPair(userID string, isAdmin bool) (*TokenPair, error) {
	now := time.Now()

	accessToken, err := s.createToken(userID, "access", now.Add(s.config.AccessTTL), isAdmin)
	if err != nil {
		return nil, fmt.Errorf("auth: generate access token: %w", err)
	}

	refreshToken, err := s.createToken(userID, "refresh", now.Add(s.config.RefreshTTL), isAdmin)
	if err != nil {
		return nil, fmt.Errorf("auth: generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) createToken(userID, tokenType string, expiresAt time.Time, isAdmin bool) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"type":  tokenType,
		"iat":   time.Now().Unix(),
		"exp":   expiresAt.Unix(),
		"admin": isAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Secret))
}

func (s *AuthService) parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
