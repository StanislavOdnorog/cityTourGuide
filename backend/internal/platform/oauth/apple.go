package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleJWKSURL  = "https://appleid.apple.com/auth/keys"
	appleTokenURL = "https://appleid.apple.com/auth/token" //nolint:gosec // URL, not a credential
	appleIssuer   = "https://appleid.apple.com"
)

// AppleClaims holds the verified claims from an Apple ID token.
type AppleClaims struct {
	Sub   string // Apple user ID
	Email string
}

// AppleConfig holds the configuration for Apple Sign-In verification.
type AppleConfig struct {
	ClientID string // App bundle ID (e.g. com.citystories.app)
	TeamID   string // Apple Developer Team ID
	KeyID    string // Key ID from Apple Developer Console
	// PrivateKey is the PEM-encoded ECDSA private key from Apple.
	// Used to generate the client_secret JWT.
	PrivateKey string
}

// AppleVerifier verifies Apple Sign-In authorization codes and tokens.
type AppleVerifier struct {
	config AppleConfig
	jwks   *JWKSCache
	client *http.Client
}

// NewAppleVerifier creates a verifier for Apple Sign-In.
func NewAppleVerifier(config AppleConfig, client *http.Client) *AppleVerifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &AppleVerifier{
		config: config,
		jwks:   NewJWKSCache(appleJWKSURL, client),
		client: client,
	}
}

// appleTokenResponse is the response from Apple's token endpoint.
type appleTokenResponse struct {
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
}

// Verify exchanges an Apple authorization code for an ID token, verifies it,
// and returns the extracted claims.
func (v *AppleVerifier) Verify(authorizationCode string) (*AppleClaims, error) {
	clientSecret, err := v.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("oauth: apple: generate client secret: %w", err)
	}

	idToken, err := v.exchangeCode(authorizationCode, clientSecret)
	if err != nil {
		return nil, err
	}

	return v.verifyIDToken(idToken)
}

// VerifyIDToken verifies an Apple ID token directly (for cases where
// the mobile app already has the id_token from Apple's SDK).
func (v *AppleVerifier) VerifyIDToken(idToken string) (*AppleClaims, error) {
	return v.verifyIDToken(idToken)
}

func (v *AppleVerifier) exchangeCode(code, clientSecret string) (string, error) {
	data := url.Values{
		"client_id":     {v.config.ClientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}

	resp, err := v.client.PostForm(appleTokenURL, data)
	if err != nil {
		return "", fmt.Errorf("oauth: apple: exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("oauth: apple: read response: %w", err)
	}

	var tokenResp appleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("oauth: apple: parse response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("oauth: apple: token error: %s", tokenResp.Error)
	}

	if tokenResp.IDToken == "" {
		return "", fmt.Errorf("oauth: apple: empty id_token in response")
	}

	return tokenResp.IDToken, nil
}

func (v *AppleVerifier) verifyIDToken(idToken string) (*AppleClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(v.config.ClientID),
	)

	token, err := parser.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("oauth: apple: missing kid in token header")
		}
		return v.jwks.GetKey(kid)
	})
	if err != nil {
		return nil, fmt.Errorf("oauth: apple: verify token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("oauth: apple: invalid token claims")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("oauth: apple: missing sub claim")
	}

	email, _ := claims["email"].(string)

	return &AppleClaims{
		Sub:   sub,
		Email: email,
	}, nil
}

func (v *AppleVerifier) generateClientSecret() (string, error) {
	key, err := jwt.ParseECPrivateKeyFromPEM([]byte(v.config.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("parse apple private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": v.config.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": appleIssuer,
		"sub": v.config.ClientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = v.config.KeyID

	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign client secret: %w", err)
	}

	return signed, nil
}
