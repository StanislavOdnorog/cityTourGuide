package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminTokenValidator validates an access token and checks for admin privileges.
type AdminTokenValidator interface {
	ValidateAdminToken(token string) (string, error)
}

// AdminAuth returns a Gin middleware that validates JWT tokens and checks
// that the user has admin privileges. Returns 403 if the user is not an admin.
func AdminAuth(validator AdminTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abortErrorJSON(c, http.StatusUnauthorized, "authorization header required")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
			abortErrorJSON(c, http.StatusUnauthorized, "invalid authorization format, expected: Bearer <token>")
			return
		}

		userID, err := validator.ValidateAdminToken(strings.TrimSpace(parts[1]))
		if err != nil {
			if strings.Contains(err.Error(), "admin access required") {
				abortErrorJSON(c, http.StatusForbidden, "admin access required")
				return
			}
			abortErrorJSON(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
