package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrSessionNotFound is returned when a session config file doesn't exist
var ErrSessionNotFound = errors.New("session not found")

const (
	// DefaultConfigDir is the default directory for Kodama configuration
	DefaultConfigDir = ".kodama"

	// SessionsSubdir is the subdirectory for session configs
	SessionsSubdir = "sessions"

	// GlobalConfigFile is the filename for global configuration
	GlobalConfigFile = "config.yaml"
)

// Store handles reading and writing configuration files
type Store struct {
	configDir string
}

// NewStore creates a new configuration store
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(home, DefaultConfigDir)
	return &Store{configDir: configDir}, nil
}

// NewStoreWithPath creates a store with a custom config directory
func NewStoreWithPath(configDir string) *Store {
	return &Store{configDir: configDir}
}

// EnsureConfigDir creates the configuration directory structure if it doesn't exist
func (s *Store) EnsureConfigDir() error {
	// Create main config directory
	if err := os.MkdirAll(s.configDir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create sessions subdirectory
	sessionsDir := filepath.Join(s.configDir, SessionsSubdir)
	if err := os.MkdirAll(sessionsDir, 0o750); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return nil
}

// GetSessionPath returns the file path for a session config
func (s *Store) GetSessionPath(name string) string {
	return filepath.Join(s.configDir, SessionsSubdir, name+".yaml")
}

// GetGlobalConfigPath returns the file path for global config
func (s *Store) GetGlobalConfigPath() string {
	return filepath.Join(s.configDir, GlobalConfigFile)
}

// LoadSession loads a session configuration from disk
func (s *Store) LoadSession(name string) (*SessionConfig, error) {
	path := s.GetSessionPath(name)

	// #nosec G304 -- path is constructed from validated session name
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to read session config: %w", err)
	}

	var config SessionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse session config: %w", err)
	}

	return &config, nil
}

// SaveSession saves a session configuration to disk
func (s *Store) SaveSession(config *SessionConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	if err := s.EnsureConfigDir(); err != nil {
		return err
	}

	path := s.GetSessionPath(config.Name)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal session config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write session config: %w", err)
	}

	return nil
}

// LoadSessionTemplate loads a session template configuration from an arbitrary path
// This is used for --config flag to load session templates.
// Unlike LoadSession, this does not validate the config as templates can be partial.
func (s *Store) LoadSessionTemplate(path string) (*SessionConfig, error) {
	// Validate path exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session template file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to access template file: %w", err)
	}

	// Read and parse YAML
	data, err := os.ReadFile(path) // #nosec G304 -- user-provided path
	if err != nil {
		return nil, fmt.Errorf("failed to read session template: %w", err)
	}

	var config SessionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse session template: %w", err)
	}

	// Note: Do NOT validate here - template can have partial config
	return &config, nil
}

// DeleteSession removes a session configuration from disk
func (s *Store) DeleteSession(name string) error {
	path := s.GetSessionPath(name)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to delete session config: %w", err)
	}

	return nil
}

// ListSessions returns all session configurations
func (s *Store) ListSessions() ([]*SessionConfig, error) {
	sessionsDir := filepath.Join(s.configDir, SessionsSubdir)

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SessionConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	sessions := make([]*SessionConfig, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		session, err := s.LoadSession(name)
		if err != nil {
			// Log error but continue with other sessions
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// LoadGlobalConfig loads the global configuration
func (s *Store) LoadGlobalConfig() (*GlobalConfig, error) {
	path := s.GetGlobalConfigPath()

	// #nosec G304 -- path is constructed from config directory
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultGlobalConfig(), nil
		}
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	// Merge with defaults
	defaultConfig := DefaultGlobalConfig()
	defaultConfig.Merge(&config)

	return defaultConfig, nil
}

// SaveGlobalConfig saves the global configuration to disk
func (s *Store) SaveGlobalConfig(config *GlobalConfig) error {
	if err := s.EnsureConfigDir(); err != nil {
		return err
	}

	path := s.GetGlobalConfigPath()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}

	return nil
}

// SessionExists checks if a session configuration exists
func (s *Store) SessionExists(name string) bool {
	path := s.GetSessionPath(name)
	_, err := os.Stat(path)
	return err == nil
}
