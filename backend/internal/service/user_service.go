package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// GracePeriod is the time a user has to restore their account after requesting deletion.
const GracePeriod = 30 * 24 * time.Hour // 30 days

// Sentinel errors for user operations.
var (
	ErrAccountScheduledForDeletion = errors.New("account scheduled for deletion")
	ErrAccountNotScheduled         = errors.New("account is not scheduled for deletion")
)

// UserRepository defines the repository methods needed by UserService.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	SoftDelete(ctx context.Context, id string) error
	RestoreAccount(ctx context.Context, id string) error
	HardDeleteExpired(ctx context.Context, gracePeriod time.Duration) (int64, error)
}

// UserService handles user account operations.
type UserService struct {
	repo UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

// ScheduleDeletion marks a user account for deletion.
func (s *UserService) ScheduleDeletion(ctx context.Context, userID string) error {
	// Verify user exists
	_, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return repository.ErrNotFound
		}
		return fmt.Errorf("user_service: schedule deletion: %w", err)
	}

	if err := s.repo.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("user_service: schedule deletion: %w", err)
	}

	return nil
}

// RestoreAccount cancels a pending deletion.
func (s *UserService) RestoreAccount(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return repository.ErrNotFound
		}
		return fmt.Errorf("user_service: restore: %w", err)
	}

	if !user.IsScheduledForDeletion() {
		return ErrAccountNotScheduled
	}

	if err := s.repo.RestoreAccount(ctx, userID); err != nil {
		return fmt.Errorf("user_service: restore: %w", err)
	}

	return nil
}

// HardDeleteExpired permanently removes accounts past the grace period.
func (s *UserService) HardDeleteExpired(ctx context.Context) (int64, error) {
	count, err := s.repo.HardDeleteExpired(ctx, GracePeriod)
	if err != nil {
		return 0, fmt.Errorf("user_service: hard delete expired: %w", err)
	}
	return count, nil
}

// GetByID returns a user by ID.
func (s *UserService) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	return s.repo.GetByID(ctx, userID)
}
