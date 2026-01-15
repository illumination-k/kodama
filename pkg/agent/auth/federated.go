package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FederatedProvider implements OAuth/OIDC-style authentication
type FederatedProvider struct {
	config FederatedConfig
	cache  *tokenCache
}

// tokenCache holds the current token in memory
type tokenCache struct {
	Token     string
	ExpiresAt time.Time
	mu        sync.RWMutex
}

// NewFederatedProvider creates a new federated authentication provider
func NewFederatedProvider(config FederatedConfig) *FederatedProvider {
	p := &FederatedProvider{
		config: config,
		cache:  &tokenCache{},
	}

	// Try to load from cache file if configured
	if config.CacheFile != "" {
		_ = p.loadCacheFile() // Ignore errors, will refresh if needed
	}

	return p
}

// GetCredentials retrieves the current authentication token
func (p *FederatedProvider) GetCredentials(ctx context.Context) (*Credentials, error) {
	p.cache.mu.RLock()
	token := p.cache.Token
	expiresAt := p.cache.ExpiresAt
	p.cache.mu.RUnlock()

	// Check if token is valid
	if token != "" && time.Now().Before(expiresAt) {
		return &Credentials{
			Token:     token,
			ExpiresAt: &expiresAt,
			Metadata:  make(map[string]string),
		}, nil
	}

	// Token is missing or expired, need to refresh
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Return refreshed credentials
	p.cache.mu.RLock()
	defer p.cache.mu.RUnlock()

	return &Credentials{
		Token:     p.cache.Token,
		ExpiresAt: &p.cache.ExpiresAt,
		Metadata:  make(map[string]string),
	}, nil
}

// Type returns the authentication type
func (p *FederatedProvider) Type() AuthType {
	return AuthTypeFederated
}

// NeedsRefresh checks if the token is expired or near expiration
func (p *FederatedProvider) NeedsRefresh() bool {
	p.cache.mu.RLock()
	defer p.cache.mu.RUnlock()

	// Refresh if token expires within 5 minutes
	return time.Until(p.cache.ExpiresAt) < 5*time.Minute
}

// Refresh obtains a new token from the OAuth endpoint
func (p *FederatedProvider) Refresh(ctx context.Context) error {
	if p.config.TokenEndpoint == "" {
		return fmt.Errorf("token endpoint not configured")
	}

	if p.config.RefreshToken == "" {
		return fmt.Errorf("refresh token not configured")
	}

	// Prepare request body
	reqBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": p.config.RefreshToken,
		"client_id":     p.config.ClientID,
	}

	if p.config.ClientSecret != "" {
		reqBody["client_secret"] = p.config.ClientSecret
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.TokenEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status: %s", resp.Status)
	}

	// Parse response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("no access token in response")
	}

	// Update cache
	p.cache.mu.Lock()
	p.cache.Token = tokenResp.AccessToken
	p.cache.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	p.cache.mu.Unlock()

	// Write to cache file if configured
	if p.config.CacheFile != "" {
		if err := p.writeCacheFile(); err != nil {
			// Log error but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to write cache file: %v\n", err)
		}
	}

	return nil
}

// loadCacheFile loads cached token from disk
func (p *FederatedProvider) loadCacheFile() error {
	path := p.expandPath(p.config.CacheFile)

	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return err
	}

	var cache struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		return err
	}

	p.cache.mu.Lock()
	p.cache.Token = cache.Token
	p.cache.ExpiresAt = cache.ExpiresAt
	p.cache.mu.Unlock()

	return nil
}

// writeCacheFile writes cached token to disk
func (p *FederatedProvider) writeCacheFile() error {
	path := p.expandPath(p.config.CacheFile)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	p.cache.mu.RLock()
	cache := struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}{
		Token:     p.cache.Token,
		ExpiresAt: p.cache.ExpiresAt,
	}
	p.cache.mu.RUnlock()

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// expandPath expands ~ to home directory in path
func (p *FederatedProvider) expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[1:])
		}
	}
	return path
}
