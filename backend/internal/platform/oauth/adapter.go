package oauth

import "github.com/saas/city-stories-guide/backend/internal/handler"

// GoogleHandlerAdapter adapts GoogleVerifier to the handler.GoogleVerifier interface.
type GoogleHandlerAdapter struct {
	v *GoogleVerifier
}

// NewGoogleHandlerAdapter wraps a GoogleVerifier for use by the auth handler.
func NewGoogleHandlerAdapter(v *GoogleVerifier) *GoogleHandlerAdapter {
	return &GoogleHandlerAdapter{v: v}
}

// Verify implements handler.GoogleVerifier.
func (a *GoogleHandlerAdapter) Verify(idToken string) (*handler.OAuthResult, error) {
	claims, err := a.v.Verify(idToken)
	if err != nil {
		return nil, err
	}
	return &handler.OAuthResult{
		Sub:   claims.Sub,
		Email: claims.Email,
		Name:  claims.Name,
	}, nil
}

// AppleHandlerAdapter adapts AppleVerifier to the handler.AppleVerifier interface.
type AppleHandlerAdapter struct {
	v *AppleVerifier
}

// NewAppleHandlerAdapter wraps an AppleVerifier for use by the auth handler.
func NewAppleHandlerAdapter(v *AppleVerifier) *AppleHandlerAdapter {
	return &AppleHandlerAdapter{v: v}
}

// Verify implements handler.AppleVerifier.
func (a *AppleHandlerAdapter) Verify(authorizationCode string) (*handler.OAuthResult, error) {
	claims, err := a.v.Verify(authorizationCode)
	if err != nil {
		return nil, err
	}
	return &handler.OAuthResult{
		Sub:   claims.Sub,
		Email: claims.Email,
	}, nil
}

// VerifyIDToken implements handler.AppleVerifier.
func (a *AppleHandlerAdapter) VerifyIDToken(idToken string) (*handler.OAuthResult, error) {
	claims, err := a.v.VerifyIDToken(idToken)
	if err != nil {
		return nil, err
	}
	return &handler.OAuthResult{
		Sub:   claims.Sub,
		Email: claims.Email,
	}, nil
}
