package port

import (
	"github.com/illumination-k/kodama/pkg/config"
)

// SessionRepository handles persistence of session configurations
type SessionRepository interface {
	// LoadSession loads a session configuration by name
	LoadSession(name string) (*config.SessionConfig, error)

	// SaveSession saves a session configuration
	SaveSession(session *config.SessionConfig) error

	// DeleteSession removes a session configuration
	DeleteSession(name string) error

	// ListSessions returns all session configurations
	ListSessions() ([]*config.SessionConfig, error)

	// SessionExists checks if a session exists
	SessionExists(name string) bool

	// GetSessionPath returns the file path for a session config
	GetSessionPath(name string) string
}

// ConfigRepository handles persistence of global configuration
type ConfigRepository interface {
	// LoadGlobalConfig loads the global configuration
	LoadGlobalConfig() (*config.GlobalConfig, error)

	// SaveGlobalConfig saves the global configuration
	SaveGlobalConfig(config *config.GlobalConfig) error

	// LoadSessionTemplate loads a session template from an arbitrary path
	LoadSessionTemplate(path string) (*config.SessionConfig, error)

	// EnsureConfigDir creates the configuration directory structure if it doesn't exist
	EnsureConfigDir() error

	// GetGlobalConfigPath returns the file path for global config
	GetGlobalConfigPath() string
}
