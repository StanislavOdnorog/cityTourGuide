package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/repository"
)

// handleDBError checks whether err is a classified repository error and writes
// the appropriate HTTP response.  It returns true if the error was handled
// (response written), false otherwise — letting the caller fall through to its
// own generic 500 response.
//
// resourceName is used in human-readable messages, e.g. "city", "POI".
func handleDBError(c *gin.Context, err error, resourceName string) bool {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		errorJSON(c, http.StatusNotFound, resourceName+" not found")
		return true
	case errors.Is(err, repository.ErrConflict):
		errorJSON(c, http.StatusConflict, resourceName+" already exists")
		return true
	case errors.Is(err, repository.ErrInvalidReference):
		errorJSON(c, http.StatusBadRequest, "referenced record does not exist")
		return true
	case errors.Is(err, repository.ErrCheckViolation):
		errorJSON(c, http.StatusBadRequest, "value violates constraint")
		return true
	case errors.Is(err, repository.ErrInvalidInput):
		errorJSON(c, http.StatusBadRequest, "invalid input value")
		return true
	default:
		return false
	}
}
