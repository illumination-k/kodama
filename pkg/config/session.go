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

// AgentExecution represents a single agent execution record
type AgentExecution struct {
	ExecutedAt time.Time `yaml:"executedAt"`
	Prompt     string    `yaml:"prompt,omitempty"`
	TaskID     string    `yaml:"taskID,omitempty"`
	Status     string    `yaml:"status"` // "pending", "running", "completed", "failed"
	Error      string    `yaml:"error,omitempty"`
}

// SessionConfig represents a Kodama session configuration
//
//nolint:govet // fieldalignment: accepting minor memory overhead for logical field grouping
type SessionConfig struct {
	CreatedAt       time.Time           `yaml:"createdAt"`
	UpdatedAt       time.Time           `yaml:"updatedAt"`
	Sync            SyncConfig          `yaml:"sync,omitempty"`
	Resources       ResourceConfig      `yaml:"resources,omitempty"`
	Ttyd            TtydConfig          `yaml:"ttyd,omitempty"`
	Name            string              `yaml:"name"`
	Namespace       string              `yaml:"namespace"`
	Repo            string              `yaml:"repo"`
	Branch          string              `yaml:"branch"`
	BaseBranch      string              `yaml:"baseBranch,omitempty"`
	PodName         string              `yaml:"podName"`
	WorkspacePVC    string              `yaml:"workspacePVC"`
	ClaudeHomePVC   string              `yaml:"claudeHomePVC"`
	CommitHash      string              `yaml:"commitHash,omitempty"`
	Image           string              `yaml:"image,omitempty"`
	Command         []string            `yaml:"command,omitempty"`
	GitSecret       string              `yaml:"gitSecret,omitempty"`
	GitClone        GitCloneConfig      `yaml:"gitClone,omitempty"`
	Status          SessionStatus       `yaml:"status"`
	AutoBranch      bool                `yaml:"autoBranch,omitempty"`
	ClaudeAuth      *ClaudeAuthOverride `yaml:"claudeAuth,omitempty"`
	AgentExecutions []AgentExecution    `yaml:"agentExecutions,omitempty"`
	LastAgentRun    *time.Time          `yaml:"lastAgentRun,omitempty"`
}

// ClaudeAuthOverride allows per-session authentication overrides
type ClaudeAuthOverride struct {
	AuthType   string `yaml:"authType,omitempty"`   // Override global auth type
	SecretName string `yaml:"secretName,omitempty"` // Override secret name
	Profile    string `yaml:"profile,omitempty"`    // Override file auth profile
}

// GitCloneConfig holds git clone options
type GitCloneConfig struct {
	Depth        int    `yaml:"depth,omitempty"`        // Shallow clone depth (0 = full)
	SingleBranch bool   `yaml:"singleBranch,omitempty"` // Clone only single branch
	ExtraArgs    string `yaml:"extraArgs,omitempty"`    // Additional git clone arguments
}

// SyncConfig holds configuration for file synchronization
type SyncConfig struct {
	UseGitignore   *bool           `yaml:"useGitignore,omitempty"`
	LocalPath      string          `yaml:"localPath,omitempty"`
	MutagenSession string          `yaml:"mutagenSession,omitempty"`
	Exclude        []string        `yaml:"exclude,omitempty"`
	CustomDirs     []CustomDirSync `yaml:"customDirs,omitempty"`
	Enabled        bool            `yaml:"enabled"`
}

// ResourceConfig holds resource limit configuration
type ResourceConfig struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// TtydConfig holds ttyd (Web-based terminal) configuration
type TtydConfig struct {
	Enabled  *bool  `yaml:"enabled,omitempty"`  // nil = use default (true), explicitly set to override
	Port     int    `yaml:"port,omitempty"`     // Default: 7681
	Options  string `yaml:"options,omitempty"`  // Additional ttyd options
	Writable *bool  `yaml:"writable,omitempty"` // nil = use default (true), false = read-only mode
}

// Validate checks if the session configuration is valid
func (s *SessionConfig) Validate() error {
	if s.Name == "" {
		return ErrSessionNameRequired
	}
	if s.Namespace == "" {
		return ErrNamespaceRequired
	}
	// Repo is now optional (not required when using sync)
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

// RecordAgentExecution adds a new agent execution record
func (s *SessionConfig) RecordAgentExecution(execution AgentExecution) {
	s.AgentExecutions = append(s.AgentExecutions, execution)
	now := execution.ExecutedAt
	s.LastAgentRun = &now
	s.UpdatedAt = time.Now()
}

// GetLastAgentExecution returns the most recent agent execution
func (s *SessionConfig) GetLastAgentExecution() *AgentExecution {
	if len(s.AgentExecutions) == 0 {
		return nil
	}
	return &s.AgentExecutions[len(s.AgentExecutions)-1]
}

// HasPendingAgentTask checks if there's a pending agent task
func (s *SessionConfig) HasPendingAgentTask() bool {
	for _, exec := range s.AgentExecutions {
		if exec.Status == "pending" || exec.Status == "running" {
			return true
		}
	}
	return false
}
