package service

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

const (
	// maxProximityBonus is the maximum bonus for being close (distance=0).
	maxProximityBonus = 30.0
	// directionBonusFactor is the fraction of base score added when POI is ahead.
	directionBonusFactor = 0.20
	// directionAngleLimit is the half-angle (degrees) for the "ahead" cone.
	directionAngleLimit = 45.0
	// nearbyPOIFetchLimit caps the POI query fan-out before scoring and truncation.
	nearbyPOIFetchLimit = 50
	// maxCandidates is the maximum number of story candidates returned.
	maxCandidates = 5
)

// StoryCandidate represents a scored story recommendation for the client.
type StoryCandidate struct {
	POIID       int     `json:"poi_id"`
	POIName     string  `json:"poi_name"`
	POILat      float64 `json:"poi_lat"`
	POILng      float64 `json:"poi_lng"`
	StoryID     int     `json:"story_id"`
	StoryText   string  `json:"story_text"`
	AudioURL    *string `json:"audio_url"`
	DurationSec *int16  `json:"duration_sec"`
	DistanceM   float64 `json:"distance_m"`
	Score       float64 `json:"score"`
}

// POIFinder retrieves nearby POIs with active stories.
type POIFinder interface {
	FindNearbyAll(ctx context.Context, lat, lng, radiusM float64, language string, page domain.PageRequest) (*domain.PageResponse[repository.NearbyPOI], error)
}

// StoryGetter fetches stories for a given POI.
type StoryGetter interface {
	GetByPOIID(ctx context.Context, poiID int, language string, status *domain.StoryStatus) ([]domain.Story, error)
	GetByPOIIDs(ctx context.Context, poiIDs []int, language string, status *domain.StoryStatus, excludeUserID string) (map[int][]domain.Story, error)
}

// NearbyService selects and scores nearby stories for a walking user.
type NearbyService struct {
	poiFinder   POIFinder
	storyGetter StoryGetter
}

// NewNearbyService creates a new NearbyService.
func NewNearbyService(pf POIFinder, sg StoryGetter) *NearbyService {
	return &NearbyService{
		poiFinder:   pf,
		storyGetter: sg,
	}
}

// GetNearbyStories returns up to 5 top-scored story candidates near the user.
// Stories already listened to by the user are excluded.
// heading is the user's compass bearing in degrees [0,360); use negative to skip direction bonus.
// speed is the user's walking speed in m/s (reserved for future pacing logic).
func (s *NearbyService) GetNearbyStories(
	ctx context.Context,
	lat, lng, radiusM, heading, speed float64,
	userID, language string,
) ([]StoryCandidate, error) {
	// 1. Find nearby POIs that have active stories in the given language.
	// Cap POI fan-out to limit DB and allocation overhead in dense areas.
	poiPage := domain.PageRequest{Limit: nearbyPOIFetchLimit}
	nearbyResult, err := s.poiFinder.FindNearbyAll(ctx, lat, lng, radiusM, language, poiPage)
	if err != nil {
		return nil, fmt.Errorf("nearby_service: find nearby: %w", err)
	}
	if len(nearbyResult.Items) == 0 {
		return nil, nil
	}
	nearbyPOIs := nearbyResult.Items

	// 2. Batch-load stories for all nearby POIs, excluding already-listened
	//    stories via a NOT EXISTS subquery when userID is provided.
	activeStatus := domain.StoryStatusActive
	poiIDs := make([]int, len(nearbyPOIs))
	for i := range nearbyPOIs {
		poiIDs[i] = nearbyPOIs[i].ID
	}

	storiesByPOI, err := s.storyGetter.GetByPOIIDs(ctx, poiIDs, language, &activeStatus, userID)
	if err != nil {
		return nil, fmt.Errorf("nearby_service: get stories batch: %w", err)
	}

	var candidates []StoryCandidate

	for i := range nearbyPOIs {
		np := &nearbyPOIs[i]

		for j := range storiesByPOI[np.ID] {
			story := &storiesByPOI[np.ID][j]

			score := CalculateScore(
				float64(np.InterestScore),
				np.DistanceM, radiusM,
				heading,
				lat, lng, np.Lat, np.Lng,
			)

			candidates = append(candidates, StoryCandidate{
				POIID:       np.ID,
				POIName:     np.POI.DisplayName(language),
				POILat:      np.Lat,
				POILng:      np.Lng,
				StoryID:     story.ID,
				StoryText:   story.Text,
				AudioURL:    story.AudioURL,
				DurationSec: story.DurationSec,
				DistanceM:   np.DistanceM,
				Score:       score,
			})
		}
	}

	// 4. Sort by score descending, with deterministic tie-breakers:
	//    1. Score descending
	//    2. DistanceM ascending (closer first)
	//    3. POIID ascending
	//    4. StoryID ascending
	sort.Slice(candidates, func(i, j int) bool {
		a, b := &candidates[i], &candidates[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.DistanceM != b.DistanceM {
			return a.DistanceM < b.DistanceM
		}
		if a.POIID != b.POIID {
			return a.POIID < b.POIID
		}
		return a.StoryID < b.StoryID
	})

	// 5. Return top N candidates.
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	return candidates, nil
}

// CalculateScore computes the composite score for a story candidate.
//
//	score = base_interest_score + proximity_bonus + direction_bonus
func CalculateScore(baseInterestScore, distanceM, radiusM, heading, userLat, userLng, poiLat, poiLng float64) float64 {
	score := baseInterestScore
	score += ProximityBonus(distanceM, radiusM)
	score += DirectionBonus(baseInterestScore, heading, userLat, userLng, poiLat, poiLng)
	return score
}

// ProximityBonus returns a bonus that linearly increases as distance decreases.
// At distance=0 the bonus is maxProximityBonus; at distance>=radius it is 0.
func ProximityBonus(distanceM, radiusM float64) float64 {
	if radiusM <= 0 {
		return 0
	}
	ratio := distanceM / radiusM
	if ratio >= 1.0 {
		return 0
	}
	return maxProximityBonus * (1.0 - ratio)
}

// DirectionBonus returns +20% of the base score if the POI is within ±45° of the user's heading.
// A negative heading means heading is unavailable, so no bonus is applied.
func DirectionBonus(baseScore, heading, userLat, userLng, poiLat, poiLng float64) float64 {
	if heading < 0 {
		return 0
	}
	brng := Bearing(userLat, userLng, poiLat, poiLng)
	diff := AngleDiff(heading, brng)
	if diff <= directionAngleLimit {
		return directionBonusFactor * baseScore
	}
	return 0
}

// Bearing computes the initial bearing from point A to point B in degrees [0, 360).
func Bearing(lat1, lng1, lat2, lng2 float64) float64 {
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180

	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)

	θ := math.Atan2(y, x)
	return math.Mod(θ*180/math.Pi+360, 360)
}

// AngleDiff returns the smallest angular difference between two bearings in degrees [0, 180].
func AngleDiff(a, b float64) float64 {
	diff := math.Abs(a - b)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}
