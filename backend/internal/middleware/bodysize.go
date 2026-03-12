package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LimitRequestBodySize returns middleware that limits the size of incoming
// request bodies to the given number of bytes using http.MaxBytesReader.
func LimitRequestBodySize(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
