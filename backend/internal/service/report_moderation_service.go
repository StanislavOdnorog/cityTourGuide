package service

import (
	"context"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// ReportModerationRepository provides atomic report moderation operations.
type ReportModerationRepository interface {
	ModerateDisableStory(ctx context.Context, reportID int) (*domain.ModeratedReportResult, error)
}

// ReportModerationService coordinates admin report moderation actions.
type ReportModerationService struct {
	repo ReportModerationRepository
}

// NewReportModerationService creates a new report moderation service.
func NewReportModerationService(repo ReportModerationRepository) *ReportModerationService {
	return &ReportModerationService{repo: repo}
}

// DisableStory resolves the report and disables the reported story atomically.
func (s *ReportModerationService) DisableStory(ctx context.Context, reportID int) (*domain.ModeratedReportResult, error) {
	return s.repo.ModerateDisableStory(ctx, reportID)
}
