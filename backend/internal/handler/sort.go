package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/repository"
)

func parseListSort(c *gin.Context, allowed map[string]repository.SortColumn, defaultBy, defaultDir string) (repository.ListSort, bool) {
	sortBy := c.Query("sort_by")
	if sortBy != "" {
		if _, ok := allowed[sortBy]; !ok {
			errorJSON(c, http.StatusBadRequest, "sort_by must be one of: "+strings.Join(sortedSortKeys(allowed), ", "))
			return repository.ListSort{}, false
		}
	}

	sortDir := strings.ToLower(c.Query("sort_dir"))
	if sortDir != "" && sortDir != repository.SortDirAsc && sortDir != repository.SortDirDesc {
		errorJSON(c, http.StatusBadRequest, "sort_dir must be one of: asc, desc")
		return repository.ListSort{}, false
	}

	return repository.ListSort{
		By:  defaultIfEmpty(sortBy, defaultBy),
		Dir: defaultIfEmpty(sortDir, defaultDir),
	}, true
}

func sortedSortKeys(allowed map[string]repository.SortColumn) []string {
	keys := make([]string, 0, len(allowed))
	for key := range allowed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
