package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// mockPurchaseRepo is a mock for PurchaseRepository.
type mockPurchaseRepo struct {
	purchases      map[string]*domain.Purchase // keyed by transaction_id
	createFn       func(ctx context.Context, p *domain.Purchase) (*domain.Purchase, error)
	getByTxIDFn    func(ctx context.Context, txID string) (*domain.Purchase, error)
	activeFn       func(ctx context.Context, userID string) ([]domain.Purchase, error)
	listeningsSinceFn func(ctx context.Context, userID string, since time.Time) (int, error)
	nextID         int
}

func newMockPurchaseRepo() *mockPurchaseRepo {
	return &mockPurchaseRepo{
		purchases: make(map[string]*domain.Purchase),
		nextID:    1,
	}
}

func (m *mockPurchaseRepo) Create(ctx context.Context, p *domain.Purchase) (*domain.Purchase, error) {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	if p.TransactionID != nil {
		if _, exists := m.purchases[*p.TransactionID]; exists {
			return nil, repository.ErrConflict
		}
	}
	created := *p
	created.ID = m.nextID
	m.nextID++
	created.CreatedAt = time.Now()
	if p.TransactionID != nil {
		m.purchases[*p.TransactionID] = &created
	}
	return &created, nil
}

func (m *mockPurchaseRepo) GetByID(_ context.Context, id int) (*domain.Purchase, error) {
	for _, p := range m.purchases {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockPurchaseRepo) GetByTransactionID(ctx context.Context, txID string) (*domain.Purchase, error) {
	if m.getByTxIDFn != nil {
		return m.getByTxIDFn(ctx, txID)
	}
	p, ok := m.purchases[txID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return p, nil
}

func (m *mockPurchaseRepo) GetByUserID(_ context.Context, userID string) ([]domain.Purchase, error) {
	var result []domain.Purchase
	for _, p := range m.purchases {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockPurchaseRepo) GetActivePurchases(ctx context.Context, userID string) ([]domain.Purchase, error) {
	if m.activeFn != nil {
		return m.activeFn(ctx, userID)
	}
	return m.GetByUserID(ctx, userID)
}

func (m *mockPurchaseRepo) CountListeningsSince(ctx context.Context, userID string, since time.Time) (int, error) {
	if m.listeningsSinceFn != nil {
		return m.listeningsSinceFn(ctx, userID, since)
	}
	return 0, nil
}

func validRequest() *VerifyPurchaseRequest {
	return &VerifyPurchaseRequest{
		UserID:        "user-1",
		Platform:      "ios",
		TransactionID: "tx-100",
		Receipt:       "receipt-data",
		Type:          domain.PurchaseTypeSubscription,
		Price:         9.99,
	}
}

func TestVerifyAndCreate_Success(t *testing.T) {
	repo := newMockPurchaseRepo()
	svc := NewPurchaseService(repo)

	p, err := svc.VerifyAndCreate(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected purchase ID to be set")
	}
	if p.ExpiresAt == nil {
		t.Fatal("expected subscription to have expiry")
	}
}

func TestVerifyAndCreate_DuplicateViaFastPath(t *testing.T) {
	repo := newMockPurchaseRepo()
	svc := NewPurchaseService(repo)

	req := validRequest()
	_, err := svc.VerifyAndCreate(context.Background(), req)
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}

	// Second call with same transaction_id should fail via fast-path lookup
	_, err = svc.VerifyAndCreate(context.Background(), req)
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("expected ErrDuplicateTransaction, got: %v", err)
	}
}

func TestVerifyAndCreate_DuplicateViaDBConstraint(t *testing.T) {
	// Simulate a race: GetByTransactionID returns not-found (concurrent request
	// hasn't committed yet), but Create hits the unique constraint.
	repo := newMockPurchaseRepo()
	repo.getByTxIDFn = func(context.Context, string) (*domain.Purchase, error) {
		return nil, repository.ErrNotFound
	}

	callCount := 0
	repo.createFn = func(_ context.Context, p *domain.Purchase) (*domain.Purchase, error) {
		callCount++
		if callCount > 1 {
			return nil, repository.ErrConflict
		}
		created := *p
		created.ID = callCount
		created.CreatedAt = time.Now()
		return &created, nil
	}

	svc := NewPurchaseService(repo)
	req := validRequest()

	// First call succeeds
	_, err := svc.VerifyAndCreate(context.Background(), req)
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}

	// Second call: fast-path misses (race), but DB constraint catches it
	_, err = svc.VerifyAndCreate(context.Background(), req)
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("expected ErrDuplicateTransaction from DB constraint, got: %v", err)
	}
}

func TestVerifyAndCreate_EmptyTransactionID(t *testing.T) {
	svc := NewPurchaseService(newMockPurchaseRepo())
	req := validRequest()
	req.TransactionID = ""

	_, err := svc.VerifyAndCreate(context.Background(), req)
	if !errors.Is(err, ErrInvalidReceipt) {
		t.Fatalf("expected ErrInvalidReceipt, got: %v", err)
	}
}

func TestVerifyAndCreate_InvalidPlatform(t *testing.T) {
	svc := NewPurchaseService(newMockPurchaseRepo())
	req := validRequest()
	req.Platform = "web"

	_, err := svc.VerifyAndCreate(context.Background(), req)
	if err == nil || errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("expected platform validation error, got: %v", err)
	}
}

func TestVerifyAndCreate_InvalidType(t *testing.T) {
	svc := NewPurchaseService(newMockPurchaseRepo())
	req := validRequest()
	req.Type = "gift"

	_, err := svc.VerifyAndCreate(context.Background(), req)
	if err == nil || errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("expected type validation error, got: %v", err)
	}
}

func TestVerifyAndCreate_LifetimeSetsIsLTD(t *testing.T) {
	svc := NewPurchaseService(newMockPurchaseRepo())
	req := validRequest()
	req.Type = domain.PurchaseTypeLifetime

	p, err := svc.VerifyAndCreate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.IsLTD {
		t.Fatal("expected IsLTD to be true for lifetime purchase")
	}
	if p.ExpiresAt != nil {
		t.Fatal("expected no expiry for lifetime purchase")
	}
}

func TestVerifyAndCreate_CreateReturnsNonConflictError(t *testing.T) {
	repo := newMockPurchaseRepo()
	repo.createFn = func(context.Context, *domain.Purchase) (*domain.Purchase, error) {
		return nil, errors.New("connection timeout")
	}
	svc := NewPurchaseService(repo)

	_, err := svc.VerifyAndCreate(context.Background(), validRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrDuplicateTransaction) {
		t.Fatal("should not be duplicate error for connection timeout")
	}
}

// TestGetStatus_UTCDayBoundary verifies that freemium counting uses an
// explicit UTC day window. A listening one second before midnight belongs to
// the previous day; one at or after midnight belongs to the new day.
func TestGetStatus_UTCDayBoundary(t *testing.T) {
	// Fix "now" to 2025-06-15 00:00:05 UTC — five seconds after midnight.
	fixedNow := time.Date(2025, 6, 15, 0, 0, 5, 0, time.UTC)
	expectedStart := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	repo := newMockPurchaseRepo()
	var capturedSince time.Time
	repo.listeningsSinceFn = func(_ context.Context, _ string, since time.Time) (int, error) {
		capturedSince = since
		return 3, nil
	}

	svc := NewPurchaseService(repo).WithClock(func() time.Time { return fixedNow })

	status, err := svc.GetStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The service should pass the start of the UTC day to the repo.
	if !capturedSince.Equal(expectedStart) {
		t.Fatalf("expected since=%v, got %v", expectedStart, capturedSince)
	}

	if status.FreeStoriesUsed != 3 {
		t.Fatalf("expected FreeStoriesUsed=3, got %d", status.FreeStoriesUsed)
	}
	if status.FreeStoriesLeft != 2 {
		t.Fatalf("expected FreeStoriesLeft=2, got %d", status.FreeStoriesLeft)
	}
}

// TestGetStatus_ListeningBeforeMidnightNotCounted ensures that when the clock
// is set to just after midnight UTC, a listening recorded one second before
// midnight does not appear in "today's" count (since the repo receives the
// new day's start as the boundary).
func TestGetStatus_ListeningBeforeMidnightNotCounted(t *testing.T) {
	// "Now" is 2025-06-15 00:00:01 UTC.
	fixedNow := time.Date(2025, 6, 15, 0, 0, 1, 0, time.UTC)
	expectedStart := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	repo := newMockPurchaseRepo()
	repo.listeningsSinceFn = func(_ context.Context, _ string, since time.Time) (int, error) {
		if !since.Equal(expectedStart) {
			t.Errorf("expected since=%v, got %v", expectedStart, since)
		}
		// Simulate: no listenings since midnight (yesterday's listen is excluded).
		return 0, nil
	}

	svc := NewPurchaseService(repo).WithClock(func() time.Time { return fixedNow })

	status, err := svc.GetStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.FreeStoriesUsed != 0 {
		t.Fatalf("expected FreeStoriesUsed=0 (yesterday's listen excluded), got %d", status.FreeStoriesUsed)
	}
	if status.FreeStoriesLeft != DefaultFreeStoriesPerDay {
		t.Fatalf("expected FreeStoriesLeft=%d, got %d", DefaultFreeStoriesPerDay, status.FreeStoriesLeft)
	}
}

// TestGetStatus_ListeningExactlyAtMidnight verifies a listening at exactly
// 00:00:00 UTC counts toward the new day (the boundary is inclusive).
func TestGetStatus_ListeningExactlyAtMidnight(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	expectedStart := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	repo := newMockPurchaseRepo()
	repo.listeningsSinceFn = func(_ context.Context, _ string, since time.Time) (int, error) {
		if !since.Equal(expectedStart) {
			t.Errorf("expected since=%v, got %v", expectedStart, since)
		}
		// A listening at exactly midnight is >= startOfDay, so it counts.
		return 1, nil
	}

	svc := NewPurchaseService(repo).WithClock(func() time.Time { return fixedNow })

	status, err := svc.GetStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.FreeStoriesUsed != 1 {
		t.Fatalf("expected FreeStoriesUsed=1, got %d", status.FreeStoriesUsed)
	}
}

// TestGetStatus_NonUTCClockNormalized verifies that even if the injected clock
// returns a non-UTC time, the service normalizes to UTC before computing the
// day boundary.
func TestGetStatus_NonUTCClockNormalized(t *testing.T) {
	// Clock returns 2025-06-15 02:00:00 +05:30 (IST), which is 2025-06-14 20:30:00 UTC.
	ist := time.FixedZone("IST", 5*3600+30*60)
	fixedNow := time.Date(2025, 6, 15, 2, 0, 0, 0, ist)
	// UTC day boundary should be June 14, not June 15.
	expectedStart := time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC)

	repo := newMockPurchaseRepo()
	repo.listeningsSinceFn = func(_ context.Context, _ string, since time.Time) (int, error) {
		if !since.Equal(expectedStart) {
			t.Errorf("expected since=%v, got %v", expectedStart, since)
		}
		return 2, nil
	}

	svc := NewPurchaseService(repo).WithClock(func() time.Time { return fixedNow })

	status, err := svc.GetStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.FreeStoriesUsed != 2 {
		t.Fatalf("expected FreeStoriesUsed=2, got %d", status.FreeStoriesUsed)
	}
}
