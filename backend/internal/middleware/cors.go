package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	AllowedOrigins []string
}

// CORS returns a Gin middleware that handles Cross-Origin Resource Sharing.
// It allows requests from the specified origins and blocks all others.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowed := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[strings.TrimRight(o, "/")] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		normalised := strings.TrimRight(origin, "/")

		if !allowed[normalised] {
			abortErrorJSON(c, http.StatusForbidden, "origin not allowed")
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
