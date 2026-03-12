package handler

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidUUID checks if a string is a valid UUID v4 format.
func isValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// parseUserIDQuery extracts and validates the user_id query parameter.
// Returns the user_id and true if valid, or writes an error response and returns false.
func parseUserIDQuery(c *gin.Context) (string, bool) {
	return parseRequiredUUIDQuery(c, "user_id")
}

// parseRequiredUUIDQuery extracts and validates a required UUID query parameter.
func parseRequiredUUIDQuery(c *gin.Context, name string) (string, bool) {
	v := c.Query(name)
	if v == "" {
		errorJSON(c, http.StatusBadRequest, name+" is required")
		return "", false
	}
	if !isValidUUID(v) {
		errorJSON(c, http.StatusBadRequest, name+" must be a valid UUID")
		return "", false
	}
	return v, true
}

// parseRequiredFloat extracts a required float query parameter with range validation.
// Returns the parsed value and true, or writes an error response and returns false.
func parseRequiredFloat(c *gin.Context, name string, min, max float64) (float64, bool) {
	s := c.Query(name)
	if s == "" {
		errorJSON(c, http.StatusBadRequest, name+" is required")
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		errorJSON(c, http.StatusBadRequest, name+" must be a valid number")
		return 0, false
	}
	if v < min || v > max {
		errorJSON(c, http.StatusBadRequest, fmt.Sprintf("%s must be between %g and %g", name, min, max))
		return 0, false
	}
	return v, true
}

// parseOptionalFloat extracts an optional float query parameter with a default and range validation.
// Pass math.Inf(-1) / math.Inf(1) for min/max to skip range checks.
func parseOptionalFloat(c *gin.Context, name string, defaultVal, min, max float64) (float64, bool) {
	s := c.Query(name)
	if s == "" {
		return defaultVal, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		errorJSON(c, http.StatusBadRequest, name+" must be a valid number")
		return 0, false
	}
	if !math.IsInf(min, -1) && !math.IsInf(max, 1) && (v < min || v > max) {
		errorJSON(c, http.StatusBadRequest, fmt.Sprintf("%s must be between %g and %g", name, min, max))
		return 0, false
	}
	return v, true
}

// parseRequiredQueryInt extracts a required positive integer query parameter.
func parseRequiredQueryInt(c *gin.Context, name string) (int, bool) {
	s := c.Query(name)
	if s == "" {
		errorJSON(c, http.StatusBadRequest, name+" is required")
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		errorJSON(c, http.StatusBadRequest, name+" must be a positive integer")
		return 0, false
	}
	return v, true
}

// validateCoordPair validates optional lat/lng pointer pairs from request bodies.
// Returns true if valid. Writes an error response and returns false otherwise.
func validateCoordPair(c *gin.Context, lat, lng *float64) bool {
	if (lat == nil) != (lng == nil) {
		errorJSON(c, http.StatusBadRequest, "lat and lng must both be provided or both omitted")
		return false
	}
	if lat != nil && lng != nil {
		if *lat < -90 || *lat > 90 {
			errorJSON(c, http.StatusBadRequest, "lat must be between -90 and 90")
			return false
		}
		if *lng < -180 || *lng > 180 {
			errorJSON(c, http.StatusBadRequest, "lng must be between -180 and 180")
			return false
		}
	}
	return true
}

// validationDetail represents a single field validation failure.
type validationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// validationErrorResponse sends a structured validation error response.
// If the error is a validator.ValidationErrors, it extracts per-field details.
// Otherwise it falls back to a generic bad request error.
func validationErrorResponse(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := make([]validationDetail, 0, len(ve))
		for _, fe := range ve {
			details = append(details, validationDetail{
				Field:   fieldName(fe),
				Message: fieldMessage(fe),
			})
		}

		writeValidationDetails(c, details)
		return
	}

	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		writeValidationDetails(c, []validationDetail{{
			Field:   validationErr.Field,
			Message: validationErr.Message,
		}})
		return
	}

	errorJSON(c, http.StatusBadRequest, err.Error())
}

func writeValidationDetails(c *gin.Context, details []validationDetail) {
	body := gin.H{
		"error":   "validation_error",
		"details": details,
	}
	if traceID, ok := c.Get("trace_id"); ok {
		if value, ok := traceID.(string); ok && value != "" {
			body["trace_id"] = value
		}
	}
	c.JSON(http.StatusBadRequest, body)
}

// fieldName returns the JSON field name from a validator.FieldError.
func fieldName(fe validator.FieldError) string {
	return strings.ToLower(fe.Field())
}

// fieldMessage returns a human-readable message for a validator.FieldError.
func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "invalid email format"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must not exceed " + fe.Param() + " characters"
	case "url":
		return "invalid URL format"
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "gte":
		return "must be at least " + fe.Param()
	case "lte":
		return "must be at most " + fe.Param()
	case "gt":
		return "must be greater than " + fe.Param()
	case "lt":
		return "must be less than " + fe.Param()
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "numeric":
		return "must be numeric"
	default:
		return "failed " + fe.Tag() + " validation"
	}
}
