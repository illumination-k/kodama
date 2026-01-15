package config

// GlobalConfig represents global configuration for Kodama
type GlobalConfig struct {
	Defaults DefaultsConfig   `yaml:"defaults"`
	Sync     GlobalSyncConfig `yaml:"sync,omitempty"`
	Git      GitConfig        `yaml:"git,omitempty"`
}

// DefaultsConfig holds default values for session creation
type DefaultsConfig struct {
	Namespace    string         `yaml:"namespace"`
	Image        string         `yaml:"image"`
	Resources    ResourceConfig `yaml:"resources"`
	Storage      StorageConfig  `yaml:"storage"`
	BranchPrefix string         `yaml:"branchPrefix"`
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

// GlobalSyncConfig holds global sync-related configuration
type GlobalSyncConfig struct {
	UseGitignore *bool    `yaml:"useGitignore,omitempty"`
	Exclude      []string `yaml:"exclude,omitempty"`
}

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults
func DefaultGlobalConfig() *GlobalConfig {
	useGitignore := true
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
}
