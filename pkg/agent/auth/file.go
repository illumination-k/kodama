package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileProvider implements authentication using credentials stored in a file
type FileProvider struct {
	config        FileConfig
	lastRead      time.Time
	cachedProfile *Profile
}

// NewFileProvider creates a new file-based authentication provider
func NewFileProvider(config FileConfig) *FileProvider {
	return &FileProvider{config: config}
}

// GetCredentials retrieves authentication credentials from the auth file
func (p *FileProvider) GetCredentials(ctx context.Context) (*Credentials, error) {
	// Read and parse auth file
	authFile, err := p.readAuthFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	// Select profile
	profileName := p.config.Profile
	if profileName == "" {
		profileName = authFile.DefaultProfile
	}
	if profileName == "" {
		profileName = "default"
	}

	profile, ok := authFile.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in auth file", profileName)
	}

	// Cache the profile for refresh checks
	p.cachedProfile = &profile
	p.lastRead = time.Now()

	// Parse expiration time if provided
	var expiresAt *time.Time
	if profile.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, profile.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expiresAt format: %w", err)
		}
		expiresAt = &t

		// Check if token is expired
		if time.Now().After(t) {
			return nil, fmt.Errorf("token expired at %s", profile.ExpiresAt)
		}
	}

	return &Credentials{
		Token:     profile.Token,
		ExpiresAt: expiresAt,
		Metadata: map[string]string{
			"profile":    profileName,
			"refreshUrl": profile.RefreshURL,
		},
	}, nil
}

// Type returns the authentication type
func (p *FileProvider) Type() AuthType {
	return AuthTypeFile
}

// NeedsRefresh checks if the token is expired or near expiration
func (p *FileProvider) NeedsRefresh() bool {
	if p.cachedProfile == nil {
		return false
	}

	if p.cachedProfile.ExpiresAt == "" {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, p.cachedProfile.ExpiresAt)
	if err != nil {
		return false
	}

	// Refresh if token expires within 5 minutes
	return time.Until(expiresAt) < 5*time.Minute
}

// Refresh attempts to refresh the token
func (p *FileProvider) Refresh(ctx context.Context) error {
	if p.cachedProfile == nil || p.cachedProfile.RefreshURL == "" {
		return fmt.Errorf("token refresh not supported: no refresh URL configured")
	}

	// TODO: Implement HTTP call to refresh URL
	// For now, return error indicating manual refresh is needed
	return fmt.Errorf("automatic token refresh not yet implemented, please manually update the auth file")
}

// readAuthFile reads and parses the auth file
func (p *FileProvider) readAuthFile() (*AuthFile, error) {
	path := p.config.Path
	if path == "" {
		// Default to ~/.kodama/claude-auth.json
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".kodama", "claude-auth.json")
	}

	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Read file
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Parse JSON
	var authFile AuthFile
	if err := json.Unmarshal(data, &authFile); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	return &authFile, nil
}
