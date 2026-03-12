package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// DefaultFreeStoriesPerDay is the number of free stories a user can listen to per day.
const DefaultFreeStoriesPerDay = 5

// PurchaseRepository defines the purchase repository methods needed by PurchaseService.
type PurchaseRepository interface {
	Create(ctx context.Context, p *domain.Purchase) (*domain.Purchase, error)
	GetByID(ctx context.Context, id int) (*domain.Purchase, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*domain.Purchase, error)
	GetByUserID(ctx context.Context, userID string) ([]domain.Purchase, error)
	GetActivePurchases(ctx context.Context, userID string) ([]domain.Purchase, error)
	CountListeningsSince(ctx context.Context, userID string, since time.Time) (int, error)
}

// PurchaseStatus represents a user's current purchase/access state.
type PurchaseStatus struct {
	HasFullAccess      bool              `json:"has_full_access"`
	IsLifetime         bool              `json:"is_lifetime"`
	ActiveSubscription *domain.Purchase  `json:"active_subscription,omitempty"`
	CityPacks          []domain.Purchase `json:"city_packs"`
	FreeStoriesUsed    int               `json:"free_stories_used"`
	FreeStoriesLimit   int               `json:"free_stories_limit"`
	FreeStoriesLeft    int               `json:"free_stories_left"`
}

// Sentinel errors for purchase operations.
var (
	ErrDuplicateTransaction = errors.New("transaction already processed")
	ErrInvalidReceipt       = errors.New("invalid receipt data")
)

// PurchaseService handles purchase verification and status logic.
type PurchaseService struct {
	repo             PurchaseRepository
	freeStoriesLimit int
	// now returns the current time. Defaults to time.Now. Injectable for testing
	// UTC day-boundary behavior deterministically.
	now func() time.Time
}

// NewPurchaseService creates a new PurchaseService.
func NewPurchaseService(repo PurchaseRepository) *PurchaseService {
	return &PurchaseService{
		repo:             repo,
		freeStoriesLimit: DefaultFreeStoriesPerDay,
		now:              time.Now,
	}
}

// WithClock returns a copy of the service using the given clock function.
// This is intended for tests that need deterministic time control.
func (s *PurchaseService) WithClock(now func() time.Time) *PurchaseService {
	s.now = now
	return s
}

// VerifyAndCreate validates a purchase receipt and creates a purchase record.
// In production, this would call Apple/Google APIs for server-side receipt verification.
// Currently, it stores the purchase after basic validation and deduplication.
func (s *PurchaseService) VerifyAndCreate(ctx context.Context, req *VerifyPurchaseRequest) (*domain.Purchase, error) {
	if req.TransactionID == "" {
		return nil, ErrInvalidReceipt
	}

	if req.Platform != "ios" && req.Platform != "android" {
		return nil, fmt.Errorf("purchase: invalid platform: %s", req.Platform)
	}

	if req.Type != domain.PurchaseTypeCityPack &&
		req.Type != domain.PurchaseTypeSubscription &&
		req.Type != domain.PurchaseTypeLifetime {
		return nil, fmt.Errorf("purchase: invalid type: %s", req.Type)
	}

	// Fast-path deduplication: check if transaction already processed.
	// The database unique constraint on transaction_id is the final authority;
	// this lookup is an optimistic fast path to avoid building the record.
	existing, err := s.repo.GetByTransactionID(ctx, req.TransactionID)
	if err == nil && existing != nil {
		return nil, ErrDuplicateTransaction
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("purchase: check transaction: %w", err)
	}

	// Build purchase record
	purchase := &domain.Purchase{
		UserID:        req.UserID,
		Type:          req.Type,
		CityID:        req.CityID,
		Platform:      req.Platform,
		TransactionID: &req.TransactionID,
		Price:         req.Price,
		IsLTD:         req.Type == domain.PurchaseTypeLifetime,
	}

	// Set expiration for subscriptions
	if req.Type == domain.PurchaseTypeSubscription {
		expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days
		purchase.ExpiresAt = &expiresAt
	}

	created, err := s.repo.Create(ctx, purchase)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrDuplicateTransaction
		}
		return nil, fmt.Errorf("purchase: create: %w", err)
	}

	return created, nil
}

// GetStatus returns the current purchase/access status for a user.
func (s *PurchaseService) GetStatus(ctx context.Context, userID string) (*PurchaseStatus, error) {
	activePurchases, err := s.repo.GetActivePurchases(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("purchase: get active purchases: %w", err)
	}

	// Compute the start of the current UTC day so freemium limits are
	// deterministic regardless of database timezone configuration.
	utcNow := s.now().UTC()
	startOfDay := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)

	todayCount, err := s.repo.CountListeningsSince(ctx, userID, startOfDay)
	if err != nil {
		return nil, fmt.Errorf("purchase: count today listenings: %w", err)
	}

	status := &PurchaseStatus{
		FreeStoriesUsed:  todayCount,
		FreeStoriesLimit: s.freeStoriesLimit,
		FreeStoriesLeft:  max(0, s.freeStoriesLimit-todayCount),
	}

	for _, p := range activePurchases {
		switch {
		case p.IsLTD || p.Type == domain.PurchaseTypeLifetime:
			status.HasFullAccess = true
			status.IsLifetime = true
		case p.Type == domain.PurchaseTypeSubscription:
			status.HasFullAccess = true
			pCopy := p
			status.ActiveSubscription = &pCopy
		case p.Type == domain.PurchaseTypeCityPack:
			status.CityPacks = append(status.CityPacks, p)
		}
	}

	return status, nil
}

// HasCityAccess checks if a user has access to a specific city (via city pack, subscription, or lifetime).
func (s *PurchaseService) HasCityAccess(ctx context.Context, userID string, cityID int) (bool, error) {
	status, err := s.GetStatus(ctx, userID)
	if err != nil {
		return false, err
	}

	if status.HasFullAccess {
		return true, nil
	}

	for _, pack := range status.CityPacks {
		if pack.CityID != nil && *pack.CityID == cityID {
			return true, nil
		}
	}

	// Check freemium allowance
	return status.FreeStoriesLeft > 0, nil
}

// CanListenFree checks if a user still has free story listens available today.
func (s *PurchaseService) CanListenFree(ctx context.Context, userID string) (canListen bool, remaining int, err error) {
	status, err := s.GetStatus(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	if status.HasFullAccess {
		return true, -1, nil // unlimited
	}

	return status.FreeStoriesLeft > 0, status.FreeStoriesLeft, nil
}

// VerifyPurchaseRequest holds the request data for purchase verification.
type VerifyPurchaseRequest struct {
	UserID        string              `json:"user_id"`
	Platform      string              `json:"platform"`
	TransactionID string              `json:"transaction_id"`
	Receipt       string              `json:"receipt"`
	Type          domain.PurchaseType `json:"type"`
	CityID        *int                `json:"city_id"`
	Price         float64             `json:"price"`
}
