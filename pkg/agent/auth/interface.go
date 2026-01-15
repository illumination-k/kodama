package auth

import (
	"context"
	"time"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeToken AuthType = "token"
	AuthTypeFile  AuthType = "file"
)

// AuthProvider is the main interface for authentication providers
type AuthProvider interface {
	// GetCredentials returns the authentication credentials
	GetCredentials(ctx context.Context) (*Credentials, error)

	// Type returns the authentication type (token, file)
	Type() AuthType

	// NeedsRefresh indicates if credentials need to be refreshed
	NeedsRefresh() bool

	// Refresh attempts to refresh the credentials
	Refresh(ctx context.Context) error
}

// Credentials holds authentication information
type Credentials struct {
	Token     string            // Bearer token or API key
	ExpiresAt *time.Time        // Optional expiration time
	Metadata  map[string]string // Additional auth metadata
}

// AuthConfig represents the configuration for authentication
type AuthConfig struct {
	Type        AuthType    // Authentication type
	TokenSource TokenConfig // For token auth
	FileSource  FileConfig  // For file auth
}

// TokenConfig represents configuration for token-based authentication
type TokenConfig struct {
	// Token from environment variable
	EnvVar string

	// Token from Kubernetes secret (for pod injection)
	K8sSecretName string
	K8sSecretKey  string

	// Direct token value (for testing only)
	Token string
}

// FileConfig represents configuration for file-based authentication
type FileConfig struct {
	// Path to auth file (e.g., ~/.kodama/claude-auth.json)
	Path string

	// Profile name (for multi-profile auth files)
	Profile string
}

// AuthFile represents the structure of the auth file
type AuthFile struct {
	DefaultProfile string             `json:"defaultProfile"`
	Profiles       map[string]Profile `json:"profiles"`
}

// Profile represents a single authentication profile in the auth file
type Profile struct {
	Token      string `json:"token"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
	RefreshURL string `json:"refreshUrl,omitempty"` // For token refresh
}
