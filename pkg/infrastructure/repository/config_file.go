package repository

import (
	"github.com/illumination-k/kodama/pkg/application/port"
	"github.com/illumination-k/kodama/pkg/config"
)

// ConfigFileRepository implements port.ConfigRepository using file-based storage
type ConfigFileRepository struct {
	store *config.Store
}

// NewConfigFileRepository creates a new ConfigFileRepository
func NewConfigFileRepository() (port.ConfigRepository, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	return &ConfigFileRepository{store: store}, nil
}

// NewConfigFileRepositoryWithPath creates a repository with a custom config directory
func NewConfigFileRepositoryWithPath(configDir string) port.ConfigRepository {
	return &ConfigFileRepository{
		store: config.NewStoreWithPath(configDir),
	}
}

// LoadGlobalConfig loads the global configuration
func (r *ConfigFileRepository) LoadGlobalConfig() (*config.GlobalConfig, error) {
	return r.store.LoadGlobalConfig()
}

// SaveGlobalConfig saves the global configuration
func (r *ConfigFileRepository) SaveGlobalConfig(cfg *config.GlobalConfig) error {
	return r.store.SaveGlobalConfig(cfg)
}

// LoadSessionTemplate loads a session template from an arbitrary path
func (r *ConfigFileRepository) LoadSessionTemplate(path string) (*config.SessionConfig, error) {
	return r.store.LoadSessionTemplate(path)
}

// EnsureConfigDir creates the configuration directory structure if it doesn't exist
func (r *ConfigFileRepository) EnsureConfigDir() error {
	return r.store.EnsureConfigDir()
}

// GetGlobalConfigPath returns the file path for global config
func (r *ConfigFileRepository) GetGlobalConfigPath() string {
	return r.store.GetGlobalConfigPath()
}
