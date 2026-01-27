package repository

import (
	"github.com/illumination-k/kodama/pkg/application/port"
	"github.com/illumination-k/kodama/pkg/config"
)

// SessionFileRepository implements port.SessionRepository using file-based storage
type SessionFileRepository struct {
	store *config.Store
}

// NewSessionFileRepository creates a new SessionFileRepository
func NewSessionFileRepository() (port.SessionRepository, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	return &SessionFileRepository{store: store}, nil
}

// NewSessionFileRepositoryWithPath creates a repository with a custom config directory
func NewSessionFileRepositoryWithPath(configDir string) port.SessionRepository {
	return &SessionFileRepository{
		store: config.NewStoreWithPath(configDir),
	}
}

// LoadSession loads a session configuration by name
func (r *SessionFileRepository) LoadSession(name string) (*config.SessionConfig, error) {
	return r.store.LoadSession(name)
}

// SaveSession saves a session configuration
func (r *SessionFileRepository) SaveSession(session *config.SessionConfig) error {
	return r.store.SaveSession(session)
}

// DeleteSession removes a session configuration
func (r *SessionFileRepository) DeleteSession(name string) error {
	return r.store.DeleteSession(name)
}

// ListSessions returns all session configurations
func (r *SessionFileRepository) ListSessions() ([]*config.SessionConfig, error) {
	return r.store.ListSessions()
}

// SessionExists checks if a session exists
func (r *SessionFileRepository) SessionExists(name string) bool {
	return r.store.SessionExists(name)
}

// GetSessionPath returns the file path for a session config
func (r *SessionFileRepository) GetSessionPath(name string) string {
	return r.store.GetSessionPath(name)
}
