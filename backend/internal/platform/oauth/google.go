package oauth

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// GoogleClaims holds the verified claims from a Google ID token.
type GoogleClaims struct {
	Sub   string // Google user ID
	Email string
	Name  string
}

// GoogleVerifier verifies Google ID tokens using Google's JWKS.
type GoogleVerifier struct {
	clientID string
	jwks     *JWKSCache
}

// NewGoogleVerifier creates a verifier for Google ID tokens.
func NewGoogleVerifier(clientID string, client *http.Client) *GoogleVerifier {
	return &GoogleVerifier{
		clientID: clientID,
		jwks:     NewJWKSCache(googleJWKSURL, client),
	}
}

// Verify validates a Google ID token and returns the extracted claims.
func (v *GoogleVerifier) Verify(idToken string) (*GoogleClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer("https://accounts.google.com"),
		jwt.WithAudience(v.clientID),
	)

	token, err := parser.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("oauth: google: missing kid in token header")
		}
		return v.jwks.GetKey(kid)
	})
	if err != nil {
		return nil, fmt.Errorf("oauth: google: verify token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("oauth: google: invalid token claims")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("oauth: google: missing sub claim")
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	return &GoogleClaims{
		Sub:   sub,
		Email: email,
		Name:  name,
	}, nil
}
