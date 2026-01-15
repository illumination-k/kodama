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
//
//nolint:govet // fieldalignment: accepting minor memory overhead for logical field grouping
type SessionConfig struct {
	CreatedAt     time.Time      `yaml:"createdAt"`
	UpdatedAt     time.Time      `yaml:"updatedAt"`
	Sync          SyncConfig     `yaml:"sync,omitempty"`
	Resources     ResourceConfig `yaml:"resources,omitempty"`
	Name          string         `yaml:"name"`
	Namespace     string         `yaml:"namespace"`
	Repo          string         `yaml:"repo"`
	Branch        string         `yaml:"branch"`
	BaseBranch    string         `yaml:"baseBranch,omitempty"`
	PodName       string         `yaml:"podName"`
	WorkspacePVC  string         `yaml:"workspacePVC"`
	ClaudeHomePVC string         `yaml:"claudeHomePVC"`
	CommitHash    string         `yaml:"commitHash,omitempty"`
	Status        SessionStatus  `yaml:"status"`
	AutoBranch    bool           `yaml:"autoBranch,omitempty"`
}

// SyncConfig holds configuration for file synchronization
type SyncConfig struct {
	LocalPath      string `yaml:"localPath,omitempty"`
	MutagenSession string `yaml:"mutagenSession,omitempty"`
	Enabled        bool   `yaml:"enabled"`
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
	// Repo is optional for MVP (git support to be added later)
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
