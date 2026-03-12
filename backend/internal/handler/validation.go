package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

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
			body["request_id"] = value
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
