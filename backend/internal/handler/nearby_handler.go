package handler

import (
	"context"
	"math"
	"net/http"
	"strconv"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch nearby stories"})
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

	// lat (required)
	latStr := c.Query("lat")
	if latStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat is required"})
		return q, false
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat must be a valid number"})
		return q, false
	}
	if lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat must be between -90 and 90"})
		return q, false
	}
	q.Lat = lat

	// lng (required)
	lngStr := c.Query("lng")
	if lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng is required"})
		return q, false
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng must be a valid number"})
		return q, false
	}
	if lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng must be between -180 and 180"})
		return q, false
	}
	q.Lng = lng

	// radius (optional, default 150, range [10, 500])
	radiusStr := c.DefaultQuery("radius", "150")
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "radius must be a valid number"})
		return q, false
	}
	if radius < 10 || radius > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "radius must be between 10 and 500"})
		return q, false
	}
	q.Radius = radius

	// heading (optional, default -1 meaning unavailable)
	headingStr := c.DefaultQuery("heading", "-1")
	heading, err := strconv.ParseFloat(headingStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "heading must be a valid number"})
		return q, false
	}
	q.Heading = heading

	// speed (optional, default 0)
	speedStr := c.DefaultQuery("speed", "0")
	speed, err := strconv.ParseFloat(speedStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "speed must be a valid number"})
		return q, false
	}
	q.Speed = math.Max(0, speed)

	// language (optional, default "en")
	q.Language = c.DefaultQuery("language", "en")

	// user_id (optional)
	q.UserID = c.Query("user_id")

	return q, true
}
