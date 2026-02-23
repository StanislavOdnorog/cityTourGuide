package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenValidator validates an access token and returns the user ID.
type TokenValidator interface {
	ValidateAccessToken(token string) (string, error)
}

// JWTAuth returns a Gin middleware that validates JWT tokens from the
// Authorization header and sets the user ID in the context.
func JWTAuth(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected: Bearer <token>"})
			return
		}

		userID, err := validator.ValidateAccessToken(strings.TrimSpace(parts[1]))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// OptionalJWTAuth is like JWTAuth but does not reject requests without a token.
// If a valid token is present, user_id is set in the context.
func OptionalJWTAuth(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.Next()
			return
		}

		userID, err := validator.ValidateAccessToken(parts[1])
		if err == nil {
			c.Set("user_id", userID)
		}

		c.Next()
	}
}
