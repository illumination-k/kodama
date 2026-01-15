package auth

import (
	"context"
	"fmt"
	"os"
)

// TokenProvider implements authentication using static tokens
type TokenProvider struct {
	config TokenConfig
}

// NewTokenProvider creates a new token authentication provider
func NewTokenProvider(config TokenConfig) *TokenProvider {
	return &TokenProvider{config: config}
}

// GetCredentials retrieves the authentication token
// Priority: Direct token > Env var
func (p *TokenProvider) GetCredentials(ctx context.Context) (*Credentials, error) {
	var token string

	switch {
	case p.config.Token != "":
		// 1. Check direct token (testing only)
		token = p.config.Token
	case p.config.EnvVar != "":
		// 2. Check environment variable
		token = os.Getenv(p.config.EnvVar)
	default:
		// 3. Default to CLAUDE_CODE_AUTH_TOKEN
		token = os.Getenv("CLAUDE_CODE_AUTH_TOKEN")
	}

	if token == "" {
		return nil, fmt.Errorf("no authentication token available")
	}

	return &Credentials{
		Token:     token,
		ExpiresAt: nil, // Tokens don't have expiration in this provider
		Metadata:  make(map[string]string),
	}, nil
}

// Type returns the authentication type
func (p *TokenProvider) Type() AuthType {
	return AuthTypeToken
}

// NeedsRefresh indicates if credentials need to be refreshed
func (p *TokenProvider) NeedsRefresh() bool {
	return false // Static tokens don't auto-refresh
}

// Refresh attempts to refresh the credentials
func (p *TokenProvider) Refresh(ctx context.Context) error {
	return nil // No refresh needed for static tokens
}
