package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ValidateGPSParams returns a Gin middleware that rejects requests containing
// GPS query parameters (lat, lng) with values outside the valid range.
// This acts as an early guard so invalid coordinates never reach the database.
func ValidateGPSParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		if latStr := c.Query("lat"); latStr != "" {
			lat, err := strconv.ParseFloat(latStr, 64)
			if err != nil {
				abortErrorJSON(c, http.StatusBadRequest, "lat must be a valid number")
				return
			}
			if lat < -90 || lat > 90 {
				abortErrorJSON(c, http.StatusBadRequest, "lat must be between -90 and 90")
				return
			}
		}

		if lngStr := c.Query("lng"); lngStr != "" {
			lng, err := strconv.ParseFloat(lngStr, 64)
			if err != nil {
				abortErrorJSON(c, http.StatusBadRequest, "lng must be a valid number")
				return
			}
			if lng < -180 || lng > 180 {
				abortErrorJSON(c, http.StatusBadRequest, "lng must be between -180 and 180")
				return
			}
		}

		c.Next()
	}
}
