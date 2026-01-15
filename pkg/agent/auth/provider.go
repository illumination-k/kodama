package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

// NewAuthProvider creates an auth provider based on configuration
func NewAuthProvider(config AuthConfig) (AuthProvider, error) {
	switch config.Type {
	case AuthTypeToken:
		return NewTokenProvider(config.TokenSource), nil
	case AuthTypeFile:
		return NewFileProvider(config.FileSource), nil
	case AuthTypeFederated:
		return NewFederatedProvider(config.FederatedSource), nil
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", config.Type)
	}
}

// GetDefaultAuthProvider returns a default auth provider from environment
// Falls back to: token (env var) -> file (default location) -> error
func GetDefaultAuthProvider() (AuthProvider, error) {
	// 1. Check for CLAUDE_CODE_AUTH_TOKEN environment variable
	if token := os.Getenv("CLAUDE_CODE_AUTH_TOKEN"); token != "" {
		return NewTokenProvider(TokenConfig{
			EnvVar: "CLAUDE_CODE_AUTH_TOKEN",
		}), nil
	}

	// 2. Check for default auth file location
	homeDir, err := os.UserHomeDir()
	if err == nil {
		authFilePath := filepath.Join(homeDir, ".kodama", "claude-auth.json")
		if _, err := os.Stat(authFilePath); err == nil {
			return NewFileProvider(FileConfig{
				Path: authFilePath,
			}), nil
		}
	}

	// 3. No authentication configured
	return nil, fmt.Errorf("no authentication configured: set CLAUDE_CODE_AUTH_TOKEN or create %s/.kodama/claude-auth.json", homeDir)
}
