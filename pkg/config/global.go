package config

import (
	"github.com/illumination-k/kodama/pkg/env"
)

// GlobalConfig represents global configuration for Kodama
type GlobalConfig struct {
	Defaults DefaultsConfig   `yaml:"defaults"`
	Sync     GlobalSyncConfig `yaml:"sync,omitempty"`
	Git      GitConfig        `yaml:"git,omitempty"`
	Claude   ClaudeConfig     `yaml:"claude,omitempty"`
}

// DefaultsConfig holds default values for session creation
type DefaultsConfig struct {
	Namespace    string         `yaml:"namespace"`
	Image        string         `yaml:"image"`
	Resources    ResourceConfig `yaml:"resources"`
	Storage      StorageConfig  `yaml:"storage"`
	Ttyd         TtydConfig     `yaml:"ttyd"`
	BranchPrefix string         `yaml:"branchPrefix"`
	Env          env.EnvConfig  `yaml:"env,omitempty"`
}

// StorageConfig holds default storage sizes
type StorageConfig struct {
	Workspace  string `yaml:"workspace"`
	ClaudeHome string `yaml:"claudeHome"`
}

// GitConfig holds Git-related configuration
type GitConfig struct {
	SecretName string `yaml:"secretName,omitempty"`
}

// ClaudeConfig holds Claude authentication configuration
type ClaudeConfig struct {
	// Authentication type: token, file
	AuthType string `yaml:"authType,omitempty"` // "token", "file"

	// Token authentication settings
	Token TokenAuthConfig `yaml:"token,omitempty"`

	// File authentication settings
	File FileAuthConfig `yaml:"file,omitempty"`
}

// TokenAuthConfig holds token authentication settings
type TokenAuthConfig struct {
	// K8s secret name for token
	SecretName string `yaml:"secretName,omitempty"`
	SecretKey  string `yaml:"secretKey,omitempty"` // Default: "token"

	// Environment variable name
	EnvVar string `yaml:"envVar,omitempty"` // Default: "CLAUDE_CODE_AUTH_TOKEN"
}

// FileAuthConfig holds file authentication settings
type FileAuthConfig struct {
	// Path to auth file (default: ~/.kodama/claude-auth.json)
	Path string `yaml:"path,omitempty"`

	// Profile to use (default: "default")
	Profile string `yaml:"profile,omitempty"`
}

// GlobalSyncConfig holds global sync-related configuration
type GlobalSyncConfig struct {
	UseGitignore *bool           `yaml:"useGitignore,omitempty"`
	Exclude      []string        `yaml:"exclude,omitempty"`
	CustomDirs   []CustomDirSync `yaml:"customDirs,omitempty"`
}

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults
func DefaultGlobalConfig() *GlobalConfig {
	useGitignore := true
	ttydEnabled := true
	ttydWritable := true
	return &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "default",
			Image:     "ghcr.io/illumination-k/kodama:latest",
			Resources: ResourceConfig{
				CPU:    "1",
				Memory: "2Gi",
			},
			Storage: StorageConfig{
				Workspace:  "10Gi",
				ClaudeHome: "1Gi",
			},
			Ttyd: TtydConfig{
				Enabled:  &ttydEnabled,
				Port:     7681,
				Writable: &ttydWritable,
			},
			BranchPrefix: "kodama/",
		},
		Sync: GlobalSyncConfig{
			Exclude:      []string{}, // No default excludes
			UseGitignore: &useGitignore,
		},
	}
}

// Merge merges this config with another, with the other taking precedence
func (g *GlobalConfig) Merge(other *GlobalConfig) {
	if other.Defaults.Namespace != "" {
		g.Defaults.Namespace = other.Defaults.Namespace
	}
	if other.Defaults.Image != "" {
		g.Defaults.Image = other.Defaults.Image
	}
	if other.Defaults.Resources.CPU != "" {
		g.Defaults.Resources.CPU = other.Defaults.Resources.CPU
	}
	if other.Defaults.Resources.Memory != "" {
		g.Defaults.Resources.Memory = other.Defaults.Resources.Memory
	}
	if other.Defaults.Storage.Workspace != "" {
		g.Defaults.Storage.Workspace = other.Defaults.Storage.Workspace
	}
	if other.Defaults.Storage.ClaudeHome != "" {
		g.Defaults.Storage.ClaudeHome = other.Defaults.Storage.ClaudeHome
	}
	if other.Defaults.BranchPrefix != "" {
		g.Defaults.BranchPrefix = other.Defaults.BranchPrefix
	}
	// Merge ttyd config
	if other.Defaults.Ttyd.Port != 0 {
		g.Defaults.Ttyd.Port = other.Defaults.Ttyd.Port
	}
	if other.Defaults.Ttyd.Options != "" {
		g.Defaults.Ttyd.Options = other.Defaults.Ttyd.Options
	}
	// Enabled is a *bool, only merge if explicitly set (non-nil)
	if other.Defaults.Ttyd.Enabled != nil {
		g.Defaults.Ttyd.Enabled = other.Defaults.Ttyd.Enabled
	}
	// Writable is a *bool, only merge if explicitly set (non-nil)
	if other.Defaults.Ttyd.Writable != nil {
		g.Defaults.Ttyd.Writable = other.Defaults.Ttyd.Writable
	}
	if other.Git.SecretName != "" {
		g.Git.SecretName = other.Git.SecretName
	}
	// Merge sync config
	if len(other.Sync.Exclude) > 0 {
		g.Sync.Exclude = other.Sync.Exclude
	}
	if other.Sync.UseGitignore != nil {
		g.Sync.UseGitignore = other.Sync.UseGitignore
	}
	if len(other.Sync.CustomDirs) > 0 {
		g.Sync.CustomDirs = other.Sync.CustomDirs
	}
	// Merge env config
	if len(other.Defaults.Env.DotenvFiles) > 0 {
		g.Defaults.Env.DotenvFiles = other.Defaults.Env.DotenvFiles
	}
	if len(other.Defaults.Env.ExcludeVars) > 0 {
		// Append to existing exclusions rather than replacing
		g.Defaults.Env.ExcludeVars = append(g.Defaults.Env.ExcludeVars, other.Defaults.Env.ExcludeVars...)
	}
}
