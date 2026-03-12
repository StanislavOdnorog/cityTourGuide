package service

import (
	"context"
	"math"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// --- Mock implementations ---

type mockPOIFinder struct {
	pois []repository.NearbyPOI
	err  error
}

func (m *mockPOIFinder) FindNearbyAll(_ context.Context, _, _, _ float64, _ string, _ domain.PageRequest) (*domain.PageResponse[repository.NearbyPOI], error) {
	if m.err != nil {
		return nil, m.err
	}
	items := m.pois
	if items == nil {
		items = []repository.NearbyPOI{}
	}
	return &domain.PageResponse[repository.NearbyPOI]{
		Items: items,
	}, nil
}

type mockStoryGetter struct {
	stories map[int][]domain.Story
	err     error
}

func (m *mockStoryGetter) GetByPOIID(_ context.Context, poiID int, _ string, _ *domain.StoryStatus) ([]domain.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.stories[poiID], nil
}

type mockListeningGetter struct {
	ids []int
	err error
}

func (m *mockListeningGetter) GetListenedStoryIDs(_ context.Context, _ string) ([]int, error) {
	return m.ids, m.err
}

// --- Helper ---

func makePOI(id int, name string, interestScore int16, lat, lng, distanceM float64) repository.NearbyPOI {
	return repository.NearbyPOI{
		POI: domain.POI{
			ID:            id,
			Name:          name,
			InterestScore: interestScore,
			Lat:           lat,
			Lng:           lng,
			Status:        domain.POIStatusActive,
		},
		DistanceM: distanceM,
	}
}

func makeStory(id, poiID int) domain.Story {
	return domain.Story{
		ID:       id,
		POIID:    poiID,
		Language: "en",
		Text:     "Test story text",
		Status:   domain.StoryStatusActive,
	}
}

// --- ProximityBonus tests ---

func TestProximityBonus_ZeroDistance(t *testing.T) {
	bonus := ProximityBonus(0, 150)
	if math.Abs(bonus-maxProximityBonus) > 0.001 {
		t.Errorf("expected %.1f at distance=0, got %.4f", maxProximityBonus, bonus)
	}
}

func TestProximityBonus_HalfRadius(t *testing.T) {
	bonus := ProximityBonus(75, 150)
	expected := maxProximityBonus * 0.5
	if math.Abs(bonus-expected) > 0.001 {
		t.Errorf("expected %.1f at half radius, got %.4f", expected, bonus)
	}
}

func TestProximityBonus_AtRadius(t *testing.T) {
	bonus := ProximityBonus(150, 150)
	if bonus != 0 {
		t.Errorf("expected 0 at distance=radius, got %.4f", bonus)
	}
}

func TestProximityBonus_BeyondRadius(t *testing.T) {
	bonus := ProximityBonus(200, 150)
	if bonus != 0 {
		t.Errorf("expected 0 beyond radius, got %.4f", bonus)
	}
}

func TestProximityBonus_ZeroRadius(t *testing.T) {
	bonus := ProximityBonus(50, 0)
	if bonus != 0 {
		t.Errorf("expected 0 with zero radius, got %.4f", bonus)
	}
}

func TestProximityBonus_Linear(t *testing.T) {
	// Verify linearity: bonus at 25% radius should be 75% of max.
	bonus := ProximityBonus(37.5, 150)
	expected := maxProximityBonus * 0.75
	if math.Abs(bonus-expected) > 0.001 {
		t.Errorf("expected %.4f at 25%% radius, got %.4f", expected, bonus)
	}
}

// --- Bearing tests ---

func TestBearing_East(t *testing.T) {
	// From (0,0) to (0,1) should be roughly 90°.
	b := Bearing(0, 0, 0, 1)
	if math.Abs(b-90) > 0.1 {
		t.Errorf("expected ~90° bearing east, got %.2f", b)
	}
}

func TestBearing_North(t *testing.T) {
	// From (0,0) to (1,0) should be roughly 0°.
	b := Bearing(0, 0, 1, 0)
	if math.Abs(b) > 0.1 && math.Abs(b-360) > 0.1 {
		t.Errorf("expected ~0° bearing north, got %.2f", b)
	}
}

func TestBearing_South(t *testing.T) {
	b := Bearing(0, 0, -1, 0)
	if math.Abs(b-180) > 0.1 {
		t.Errorf("expected ~180° bearing south, got %.2f", b)
	}
}

func TestBearing_West(t *testing.T) {
	b := Bearing(0, 0, 0, -1)
	if math.Abs(b-270) > 0.1 {
		t.Errorf("expected ~270° bearing west, got %.2f", b)
	}
}

// --- AngleDiff tests ---

func TestAngleDiff_Same(t *testing.T) {
	diff := AngleDiff(90, 90)
	if diff != 0 {
		t.Errorf("expected 0 for same angles, got %.2f", diff)
	}
}

func TestAngleDiff_Opposite(t *testing.T) {
	diff := AngleDiff(0, 180)
	if diff != 180 {
		t.Errorf("expected 180 for opposite angles, got %.2f", diff)
	}
}

func TestAngleDiff_Wraparound(t *testing.T) {
	// 350° and 10° should differ by 20°.
	diff := AngleDiff(350, 10)
	if math.Abs(diff-20) > 0.001 {
		t.Errorf("expected 20° wraparound diff, got %.2f", diff)
	}
}

func TestAngleDiff_Symmetric(t *testing.T) {
	d1 := AngleDiff(30, 60)
	d2 := AngleDiff(60, 30)
	if d1 != d2 {
		t.Errorf("angle diff should be symmetric: %.2f != %.2f", d1, d2)
	}
}

// --- DirectionBonus tests ---

func TestDirectionBonus_Ahead(t *testing.T) {
	// User at (0,0), heading east (90°). POI at (0,0.001) → bearing ≈ 90°.
	bonus := DirectionBonus(100, 90, 0, 0, 0, 0.001)
	expected := directionBonusFactor * 100
	if math.Abs(bonus-expected) > 0.1 {
		t.Errorf("expected direction bonus %.1f for ahead POI, got %.4f", expected, bonus)
	}
}

func TestDirectionBonus_Behind(t *testing.T) {
	// User at (0,0), heading east (90°). POI at (0,-0.001) → bearing ≈ 270°.
	bonus := DirectionBonus(100, 90, 0, 0, 0, -0.001)
	if bonus != 0 {
		t.Errorf("expected 0 direction bonus for behind POI, got %.4f", bonus)
	}
}

func TestDirectionBonus_ExactlyAtAngleLimit(t *testing.T) {
	// User heading 0° (north). POI at bearing 45° → exactly at limit → should get bonus.
	// Place POI to the northeast from (0,0).
	poiLat := 0.001 * math.Cos(45*math.Pi/180)
	poiLng := 0.001 * math.Sin(45*math.Pi/180)
	bonus := DirectionBonus(100, 0, 0, 0, poiLat, poiLng)
	expected := directionBonusFactor * 100
	if math.Abs(bonus-expected) > 0.5 {
		t.Errorf("expected direction bonus at 45° limit, got %.4f", bonus)
	}
}

func TestDirectionBonus_NoHeading(t *testing.T) {
	// Negative heading means unavailable.
	bonus := DirectionBonus(100, -1, 0, 0, 0, 0.001)
	if bonus != 0 {
		t.Errorf("expected 0 bonus with no heading, got %.4f", bonus)
	}
}

// --- CalculateScore tests ---

func TestCalculateScore_AllComponents(t *testing.T) {
	// POI ahead and close: should get base + proximity + direction bonuses.
	baseScore := 80.0
	distanceM := 30.0
	radiusM := 150.0
	heading := 90.0
	userLat, userLng := 0.0, 0.0
	poiLat, poiLng := 0.0, 0.001 // east → bearing ≈ 90°

	score := CalculateScore(baseScore, distanceM, radiusM, heading, userLat, userLng, poiLat, poiLng)

	expectedProximity := maxProximityBonus * (1.0 - distanceM/radiusM)
	expectedDirection := directionBonusFactor * baseScore
	expectedTotal := baseScore + expectedProximity + expectedDirection

	if math.Abs(score-expectedTotal) > 0.1 {
		t.Errorf("expected score ~%.2f, got %.2f", expectedTotal, score)
	}
}

func TestCalculateScore_NoBonuses(t *testing.T) {
	// POI behind and at edge of radius: minimal score.
	baseScore := 50.0
	score := CalculateScore(baseScore, 150, 150, 90, 0, 0, 0, -0.001)
	// proximity=0 (at radius), direction=0 (behind)
	if math.Abs(score-baseScore) > 0.1 {
		t.Errorf("expected score ~%.2f with no bonuses, got %.2f", baseScore, score)
	}
}

// --- GetNearbyStories integration tests (with mocks) ---

func TestGetNearbyStories_SortedByScore(t *testing.T) {
	pois := []repository.NearbyPOI{
		makePOI(1, "Low Score POI", 30, 41.7160, 44.8280, 100),
		makePOI(2, "High Score POI", 90, 41.7155, 44.8275, 50),
	}
	stories := map[int][]domain.Story{
		1: {makeStory(10, 1)},
		2: {makeStory(20, 2)},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "user1", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].POIID != 2 {
		t.Errorf("expected first candidate POI ID 2 (higher score), got %d", candidates[0].POIID)
	}
	if candidates[1].POIID != 1 {
		t.Errorf("expected second candidate POI ID 1, got %d", candidates[1].POIID)
	}
	if candidates[0].Score <= candidates[1].Score {
		t.Errorf("candidates not sorted by score DESC: %.2f <= %.2f", candidates[0].Score, candidates[1].Score)
	}
}

func TestGetNearbyStories_ListenedExcluded(t *testing.T) {
	pois := []repository.NearbyPOI{
		makePOI(1, "POI A", 80, 41.716, 44.828, 50),
		makePOI(2, "POI B", 70, 41.715, 44.827, 80),
	}
	stories := map[int][]domain.Story{
		1: {makeStory(10, 1)},
		2: {makeStory(20, 2)},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{ids: []int{10}}, // story 10 already listened
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "user1", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (listened excluded), got %d", len(candidates))
	}
	if candidates[0].StoryID != 20 {
		t.Errorf("expected story 20, got %d", candidates[0].StoryID)
	}
}

func TestGetNearbyStories_AllListened(t *testing.T) {
	pois := []repository.NearbyPOI{
		makePOI(1, "POI A", 80, 41.716, 44.828, 50),
	}
	stories := map[int][]domain.Story{
		1: {makeStory(10, 1)},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{ids: []int{10}},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "user1", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates when all listened, got %d", len(candidates))
	}
}

func TestGetNearbyStories_DirectionBonus(t *testing.T) {
	// User heading east (90°). POI-A is east (ahead), POI-B is west (behind).
	// Same base score and distance → POI-A should win due to direction bonus.
	pois := []repository.NearbyPOI{
		makePOI(1, "Behind", 60, 41.7151, 44.8261, 100), // west of user
		makePOI(2, "Ahead", 60, 41.7151, 44.8281, 100),  // east of user
	}
	stories := map[int][]domain.Story{
		1: {makeStory(10, 1)},
		2: {makeStory(20, 2)},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 200, 90, 1.0, "", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	// POI "Ahead" (east) should have higher score.
	if candidates[0].POIName != "Ahead" {
		t.Errorf("expected 'Ahead' POI first (direction bonus), got %q with score %.2f", candidates[0].POIName, candidates[0].Score)
	}
	if candidates[0].Score <= candidates[1].Score {
		t.Errorf("ahead POI should have higher score: %.2f vs %.2f", candidates[0].Score, candidates[1].Score)
	}
}

func TestGetNearbyStories_MaxFiveCandidates(t *testing.T) {
	var pois []repository.NearbyPOI
	storiesMap := make(map[int][]domain.Story)
	for i := 1; i <= 8; i++ {
		pois = append(pois, makePOI(i, "POI", int16(50+i), 41.715, 44.827, float64(i*10)))
		storiesMap[i] = []domain.Story{makeStory(100+i, i)}
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: storiesMap},
		&mockListeningGetter{},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != maxCandidates {
		t.Errorf("expected max %d candidates, got %d", maxCandidates, len(candidates))
	}
}

func TestGetNearbyStories_NoPOIs(t *testing.T) {
	svc := NewNearbyService(
		&mockPOIFinder{pois: nil},
		&mockStoryGetter{stories: nil},
		&mockListeningGetter{},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if candidates != nil {
		t.Errorf("expected nil for no POIs, got %v", candidates)
	}
}

func TestGetNearbyStories_EmptyUserID(t *testing.T) {
	// Empty userID → no listening filter, all stories returned.
	pois := []repository.NearbyPOI{
		makePOI(1, "POI", 80, 41.716, 44.828, 50),
	}
	stories := map[int][]domain.Story{
		1: {makeStory(10, 1), makeStory(11, 1)},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates for empty userID (no filter), got %d", len(candidates))
	}
}

func TestGetNearbyStories_MultipleStoriesPerPOI(t *testing.T) {
	pois := []repository.NearbyPOI{
		makePOI(1, "POI A", 80, 41.716, 44.828, 50),
	}
	stories := map[int][]domain.Story{
		1: {makeStory(10, 1), makeStory(11, 1), makeStory(12, 1)},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{ids: []int{11}}, // only story 11 listened
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "user1", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates (3 stories - 1 listened), got %d", len(candidates))
	}
	for _, c := range candidates {
		if c.StoryID == 11 {
			t.Errorf("listened story 11 should be excluded")
		}
	}
}

func TestGetNearbyStories_CandidateFields(t *testing.T) {
	audioURL := "https://example.com/audio.mp3"
	dur := int16(30)
	pois := []repository.NearbyPOI{
		makePOI(1, "Test POI", 75, 41.716, 44.828, 42.5),
	}
	stories := map[int][]domain.Story{
		1: {{
			ID:          10,
			POIID:       1,
			Language:    "en",
			Text:        "A great story about this place",
			AudioURL:    &audioURL,
			DurationSec: &dur,
			Status:      domain.StoryStatusActive,
		}},
	}

	svc := NewNearbyService(
		&mockPOIFinder{pois: pois},
		&mockStoryGetter{stories: stories},
		&mockListeningGetter{},
	)

	candidates, err := svc.GetNearbyStories(context.Background(), 41.7151, 44.8271, 150, -1, 1.0, "", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]
	if c.POIID != 1 {
		t.Errorf("expected POI ID 1, got %d", c.POIID)
	}
	if c.POIName != "Test POI" {
		t.Errorf("expected POI name 'Test POI', got %q", c.POIName)
	}
	if c.StoryID != 10 {
		t.Errorf("expected story ID 10, got %d", c.StoryID)
	}
	if c.StoryText != "A great story about this place" {
		t.Errorf("unexpected story text: %q", c.StoryText)
	}
	if c.AudioURL == nil || *c.AudioURL != audioURL {
		t.Errorf("expected audio URL %q, got %v", audioURL, c.AudioURL)
	}
	if c.DurationSec == nil || *c.DurationSec != dur {
		t.Errorf("expected duration %d, got %v", dur, c.DurationSec)
	}
	if math.Abs(c.DistanceM-42.5) > 0.001 {
		t.Errorf("expected distance 42.5, got %.4f", c.DistanceM)
	}
	if c.Score <= 0 {
		t.Errorf("expected positive score, got %.2f", c.Score)
	}
}
