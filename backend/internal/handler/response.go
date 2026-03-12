package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// writeCursorPage writes a standard cursor-paginated JSON response.
// It normalizes nil item slices to empty arrays so they serialize as [] not null.
func writeCursorPage[T any](c *gin.Context, page *domain.PageResponse[T]) {
	items := page.Items
	if items == nil {
		items = []T{}
	}
	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": page.NextCursor,
		"has_more":    page.HasMore,
	})
}

// writeCursorPageItems writes a cursor-paginated response using pre-transformed items.
// Use this when the handler maps domain items to a different response type.
func writeCursorPageItems[T any](c *gin.Context, items []T, nextCursor string, hasMore bool) {
	if items == nil {
		items = []T{}
	}
	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func errorJSON(c *gin.Context, status int, message string) {
	c.JSON(status, errorBody(c, message))
}

func errorJSONWithFields(c *gin.Context, status int, message string, fields gin.H) {
	c.JSON(status, errorBodyWithFields(c, message, fields))
}

func errorBody(c *gin.Context, message string) gin.H {
	body := gin.H{"error": message}
	if traceID, ok := c.Get("trace_id"); ok {
		if value, ok := traceID.(string); ok && value != "" {
			body["trace_id"] = value
		}
	}
	return body
}

func errorBodyWithFields(c *gin.Context, message string, fields gin.H) gin.H {
	body := errorBody(c, message)
	for k, v := range fields {
		body[k] = v
	}
	return body
}
