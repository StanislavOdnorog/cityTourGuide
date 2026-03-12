package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// AudioCleanupStoryRepo defines the story repository methods needed by the cleanup service.
type AudioCleanupStoryRepo interface {
	GetByID(ctx context.Context, id int) (*domain.Story, error)
	Update(ctx context.Context, story *domain.Story) (*domain.Story, error)
	Delete(ctx context.Context, id int) error
}

// AudioCleanupPOIRepo defines the POI repository methods needed by the cleanup service.
type AudioCleanupPOIRepo interface {
	Delete(ctx context.Context, id int) error
}

// AudioCleanupCityRepo defines the city repository methods needed by the cleanup service.
type AudioCleanupCityRepo interface {
	Delete(ctx context.Context, id int) error
}

// CleanupEnqueuer enqueues orphaned object keys for background deletion.
type CleanupEnqueuer interface {
	Enqueue(ctx context.Context, objectKeys []string) error
}

// AudioURLCollector queries audio URLs that would be orphaned by a cascading delete.
type AudioURLCollector interface {
	// AudioURLsByStoryID returns the audio_url for a single story (if set).
	AudioURLsByStoryID(ctx context.Context, storyID int) ([]string, error)
	// AudioURLsByPOIID returns audio_urls for all stories belonging to a POI.
	AudioURLsByPOIID(ctx context.Context, poiID int) ([]string, error)
	// AudioURLsByCityID returns audio_urls for all stories belonging to a city.
	AudioURLsByCityID(ctx context.Context, cityID int) ([]string, error)
}

// AudioCleanupService wraps entity deletion to schedule removal of orphaned
// audio objects from storage. The primary database mutation always succeeds
// independently of cleanup scheduling — if enqueueing fails, the error is
// logged but does not roll back the deletion.
type AudioCleanupService struct {
	pool      *pgxpool.Pool
	storyRepo AudioCleanupStoryRepo
	poiRepo   AudioCleanupPOIRepo
	cityRepo  AudioCleanupCityRepo
	collector AudioURLCollector
	enqueuer  CleanupEnqueuer
}

// NewAudioCleanupService creates a new AudioCleanupService.
func NewAudioCleanupService(
	pool *pgxpool.Pool,
	storyRepo AudioCleanupStoryRepo,
	poiRepo AudioCleanupPOIRepo,
	cityRepo AudioCleanupCityRepo,
	collector AudioURLCollector,
	enqueuer CleanupEnqueuer,
) *AudioCleanupService {
	return &AudioCleanupService{
		pool:      pool,
		storyRepo: storyRepo,
		poiRepo:   poiRepo,
		cityRepo:  cityRepo,
		collector: collector,
		enqueuer:  enqueuer,
	}
}

// DeleteStory deletes a story and schedules cleanup of its audio object.
func (s *AudioCleanupService) DeleteStory(ctx context.Context, id int) error {
	// Collect audio URLs before deletion.
	urls, err := s.collector.AudioURLsByStoryID(ctx, id)
	if err != nil {
		slog.Warn("audio cleanup: failed to collect URLs before story delete", "story_id", id, "error", err)
		// Proceed with deletion; orphan will remain until a sweep.
	}

	if err := s.storyRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.enqueueKeys(ctx, urls)
	return nil
}

// UpdateStory updates a story and schedules cleanup if the audio object changed
// or was removed. The update is committed before any cleanup job is enqueued.
func (s *AudioCleanupService) UpdateStory(ctx context.Context, story *domain.Story) (*domain.Story, error) {
	var oldAudioURL string
	existing, err := s.storyRepo.GetByID(ctx, story.ID)
	if err == nil && existing.AudioURL != nil {
		oldAudioURL = *existing.AudioURL
	}

	updated, err := s.storyRepo.Update(ctx, story)
	if err != nil {
		return nil, err
	}

	if oldAudioURL == "" {
		return updated, nil
	}

	if updated.AudioURL == nil || *updated.AudioURL != oldAudioURL {
		s.enqueueKeys(ctx, []string{oldAudioURL})
	}

	return updated, nil
}

// DeletePOI deletes a POI (cascading to stories) and schedules cleanup
// of all owned audio objects.
func (s *AudioCleanupService) DeletePOI(ctx context.Context, id int) error {
	urls, err := s.collector.AudioURLsByPOIID(ctx, id)
	if err != nil {
		slog.Warn("audio cleanup: failed to collect URLs before POI delete", "poi_id", id, "error", err)
	}

	if err := s.poiRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.enqueueKeys(ctx, urls)
	return nil
}

// DeleteCity deletes a city (cascading to POIs and stories) and schedules
// cleanup of all owned audio objects.
func (s *AudioCleanupService) DeleteCity(ctx context.Context, id int) error {
	urls, err := s.collector.AudioURLsByCityID(ctx, id)
	if err != nil {
		slog.Warn("audio cleanup: failed to collect URLs before city delete", "city_id", id, "error", err)
	}

	if err := s.cityRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.enqueueKeys(ctx, urls)
	return nil
}

// ScheduleReplacedAudio enqueues cleanup for a previously stored audio URL.
// Prefer UpdateStory when the database write and cleanup scheduling must stay ordered.
func (s *AudioCleanupService) ScheduleReplacedAudio(ctx context.Context, oldAudioURL string) {
	if oldAudioURL == "" {
		return
	}
	s.enqueueKeys(ctx, []string{oldAudioURL})
}

// enqueueKeys converts audio URLs to object keys and enqueues them for deletion.
// Errors are logged but never returned.
func (s *AudioCleanupService) enqueueKeys(ctx context.Context, audioURLs []string) {
	keys := make([]string, 0, len(audioURLs))
	for _, u := range audioURLs {
		key := AudioURLToObjectKey(u)
		if key != "" {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return
	}

	if err := s.enqueuer.Enqueue(ctx, keys); err != nil {
		slog.Error("audio cleanup: failed to enqueue cleanup jobs",
			"keys", keys, "error", err)
	}
}

// AudioURLToObjectKey extracts the S3 object key from a stored audio URL.
// The URL format is: {endpoint}/{bucket}/{key}
// Returns empty string if the URL cannot be parsed or has no recognizable key.
func AudioURLToObjectKey(audioURL string) string {
	if audioURL == "" {
		return ""
	}

	parsed, err := url.Parse(audioURL)
	if err != nil {
		return ""
	}

	// Path is /{bucket}/{key} — trim the leading slash and split at the first /
	path := strings.TrimPrefix(parsed.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return ""
	}

	// parts[1] is the object key (e.g. "audio/1/10/100.mp3")
	return parts[1]
}

// storyAuditPayload returns a trimmed payload for story audit logs.
func storyCleanupPayload(storyID int, audioURL string) map[string]any {
	return map[string]any{
		"story_id":  storyID,
		"audio_url": audioURL,
	}
}

// Ensure AudioCleanupService satisfies interfaces used by handlers.
var _ interface {
	UpdateStory(ctx context.Context, story *domain.Story) (*domain.Story, error)
	DeleteStory(ctx context.Context, id int) error
	DeletePOI(ctx context.Context, id int) error
	DeleteCity(ctx context.Context, id int) error
	ScheduleReplacedAudio(ctx context.Context, oldAudioURL string)
} = (*AudioCleanupService)(nil)

// --- AudioURLCollectorImpl queries audio_urls from the database. ---

// AudioURLCollectorImpl implements AudioURLCollector using direct pool queries.
type AudioURLCollectorImpl struct {
	pool *pgxpool.Pool
}

// NewAudioURLCollector creates a new AudioURLCollectorImpl.
func NewAudioURLCollector(pool *pgxpool.Pool) *AudioURLCollectorImpl {
	return &AudioURLCollectorImpl{pool: pool}
}

func (c *AudioURLCollectorImpl) AudioURLsByStoryID(ctx context.Context, storyID int) ([]string, error) {
	return c.queryURLs(ctx,
		`SELECT audio_url FROM story WHERE id = $1 AND audio_url IS NOT NULL`, storyID)
}

func (c *AudioURLCollectorImpl) AudioURLsByPOIID(ctx context.Context, poiID int) ([]string, error) {
	return c.queryURLs(ctx,
		`SELECT audio_url FROM story WHERE poi_id = $1 AND audio_url IS NOT NULL`, poiID)
}

func (c *AudioURLCollectorImpl) AudioURLsByCityID(ctx context.Context, cityID int) ([]string, error) {
	return c.queryURLs(ctx,
		`SELECT s.audio_url FROM story s
		 INNER JOIN poi p ON s.poi_id = p.id
		 WHERE p.city_id = $1 AND s.audio_url IS NOT NULL`, cityID)
}

func (c *AudioURLCollectorImpl) queryURLs(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := c.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audio_url_collector: %w", err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, fmt.Errorf("audio_url_collector: scan: %w", err)
		}
		urls = append(urls, u)
	}

	return urls, rows.Err()
}
