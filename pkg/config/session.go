package config

import (
	"errors"
	"time"
)

var (
	// ErrSessionNameRequired is returned when session name is empty
	ErrSessionNameRequired = errors.New("session name is required")

	// ErrNamespaceRequired is returned when namespace is empty
	ErrNamespaceRequired = errors.New("namespace is required")

	// ErrRepoRequired is returned when repository URL is empty
	ErrRepoRequired = errors.New("repository URL is required")
)

// SessionStatus represents the current state of a session
type SessionStatus string

const (
	StatusPending  SessionStatus = "Pending"
	StatusStarting SessionStatus = "Starting"
	StatusRunning  SessionStatus = "Running"
	StatusStopped  SessionStatus = "Stopped"
	StatusFailed   SessionStatus = "Failed"
)

// SessionConfig represents a Kodama session configuration
type SessionConfig struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`

	// Repository configuration
	Repo       string `yaml:"repo"`
	Branch     string `yaml:"branch"`
	BaseBranch string `yaml:"baseBranch,omitempty"`
	AutoBranch bool   `yaml:"autoBranch,omitempty"`

	// Kubernetes resources
	PodName       string `yaml:"podName"`
	WorkspacePVC  string `yaml:"workspacePVC"`
	ClaudeHomePVC string `yaml:"claudeHomePVC"`

	// Git information
	CommitHash string `yaml:"commitHash,omitempty"`

	// Sync configuration
	Sync SyncConfig `yaml:"sync,omitempty"`

	// Session status
	Status    SessionStatus `yaml:"status"`
	CreatedAt time.Time     `yaml:"createdAt"`
	UpdatedAt time.Time     `yaml:"updatedAt"`

	// Resource limits
	Resources ResourceConfig `yaml:"resources,omitempty"`
}

// SyncConfig holds configuration for file synchronization
type SyncConfig struct {
	Enabled        bool   `yaml:"enabled"`
	LocalPath      string `yaml:"localPath,omitempty"`
	MutagenSession string `yaml:"mutagenSession,omitempty"`
}

// ResourceConfig holds resource limit configuration
type ResourceConfig struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// Validate checks if the session configuration is valid
func (s *SessionConfig) Validate() error {
	if s.Name == "" {
		return ErrSessionNameRequired
	}
	if s.Namespace == "" {
		return ErrNamespaceRequired
	}
	if s.Repo == "" {
		return ErrRepoRequired
	}
	return nil
}

// IsRunning returns true if the session is in Running state
func (s *SessionConfig) IsRunning() bool {
	return s.Status == StatusRunning
}

// IsStopped returns true if the session is in Stopped state
func (s *SessionConfig) IsStopped() bool {
	return s.Status == StatusStopped
}

// UpdateStatus updates the session status and timestamp
func (s *SessionConfig) UpdateStatus(status SessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()
}
