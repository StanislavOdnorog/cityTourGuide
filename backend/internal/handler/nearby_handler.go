package handler

import (
	"context"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/service"
)

// NearbyStoriesGetter defines the interface for fetching nearby stories.
type NearbyStoriesGetter interface {
	GetNearbyStories(
		ctx context.Context,
		lat, lng, radiusM, heading, speed float64,
		userID, language string,
	) ([]service.StoryCandidate, error)
}

// NearbyHandler handles the nearby-stories endpoint.
type NearbyHandler struct {
	nearbyService NearbyStoriesGetter
}

// NewNearbyHandler creates a new NearbyHandler.
func NewNearbyHandler(ns NearbyStoriesGetter) *NearbyHandler {
	return &NearbyHandler{nearbyService: ns}
}

// nearbyQuery holds validated query parameters for GET /api/v1/nearby-stories.
type nearbyQuery struct {
	Lat      float64
	Lng      float64
	Radius   float64
	Heading  float64
	Speed    float64
	Language string
	UserID   string
}

// GetNearbyStories handles GET /api/v1/nearby-stories.
func (h *NearbyHandler) GetNearbyStories(c *gin.Context) {
	q, ok := parseNearbyQuery(c)
	if !ok {
		return // error already written to response
	}

	candidates, err := h.nearbyService.GetNearbyStories(
		c.Request.Context(),
		q.Lat, q.Lng, q.Radius, q.Heading, q.Speed,
		q.UserID, q.Language,
	)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to fetch nearby stories")
		return
	}

	if candidates == nil {
		candidates = []service.StoryCandidate{}
	}

	c.JSON(http.StatusOK, gin.H{"data": candidates})
}

// parseNearbyQuery extracts and validates query parameters.
// Returns false if validation failed (error already written to response).
func parseNearbyQuery(c *gin.Context) (nearbyQuery, bool) {
	var q nearbyQuery
	var ok bool

	// lat (required, -90..90)
	q.Lat, ok = parseRequiredFloat(c, "lat", -90, 90)
	if !ok {
		return q, false
	}

	// lng (required, -180..180)
	q.Lng, ok = parseRequiredFloat(c, "lng", -180, 180)
	if !ok {
		return q, false
	}

	// radius (optional, default 150, range [10, 500])
	q.Radius, ok = parseOptionalFloat(c, "radius", 150, 10, 500)
	if !ok {
		return q, false
	}

	// heading (optional, default -1 meaning unavailable, no range)
	q.Heading, ok = parseOptionalFloat(c, "heading", -1, math.Inf(-1), math.Inf(1))
	if !ok {
		return q, false
	}

	// speed (optional, default 0, no range, clamp to >= 0)
	q.Speed, ok = parseOptionalFloat(c, "speed", 0, math.Inf(-1), math.Inf(1))
	if !ok {
		return q, false
	}
	q.Speed = math.Max(0, q.Speed)

	// language (optional, default "en")
	q.Language = c.DefaultQuery("language", "en")

	// user_id (optional)
	q.UserID = c.Query("user_id")

	return q, true
}
