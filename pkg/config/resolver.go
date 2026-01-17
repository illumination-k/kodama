package config

// ResolvedConfig represents the merged configuration from global and template sources
// This does NOT include CLI flags, which are applied at the usecase layer
type ResolvedConfig struct {
	// Basic fields
	Namespace       string
	Image           string
	CPU             string
	Memory          string
	CustomResources map[string]string
	GitSecret       string
	Branch          string
	CloneDepth      int
	SingleBranch    bool
	GitCloneArgs    string
	Repo            string
	Command         string

	// Ttyd config
	TtydEnabled  bool
	TtydPort     int
	TtydOptions  string
	TtydWritable bool

	// Sync config (from template only, but fallback to global)
	SyncExclude      []string
	SyncUseGitignore *bool
	SyncCustomDirs   []CustomDirSync
	ClaudeAuth       *ClaudeAuthOverride

	// Storage (from global only)
	StorageWorkspace  string
	StorageClaudeHome string
	BranchPrefix      string
}

// ConfigResolver merges global and template configurations
type ConfigResolver struct {
	global   *GlobalConfig
	template *SessionConfig
}

// NewConfigResolver creates a new ConfigResolver
// global must not be nil, template can be nil if no template config is provided
func NewConfigResolver(global *GlobalConfig, template *SessionConfig) *ConfigResolver {
	return &ConfigResolver{
		global:   global,
		template: template,
	}
}

// Resolve merges global and template configs with the following priority:
// Template > Global > Hardcoded defaults
// Returns a ResolvedConfig that can be further overridden by CLI flags at the usecase layer
func (r *ConfigResolver) Resolve() *ResolvedConfig {
	resolved := &ResolvedConfig{
		CustomResources: make(map[string]string),
	}

	// Layer 1: Apply global config defaults
	resolved.Namespace = r.global.Defaults.Namespace
	resolved.Image = r.global.Defaults.Image
	resolved.CPU = r.global.Defaults.Resources.CPU
	resolved.Memory = r.global.Defaults.Resources.Memory
	resolved.GitSecret = r.global.Git.SecretName

	// Merge custom resources from global config
	if r.global.Defaults.Resources.CustomResources != nil {
		for k, v := range r.global.Defaults.Resources.CustomResources {
			resolved.CustomResources[k] = v
		}
	}

	// Ttyd config from global
	if r.global.Defaults.Ttyd.Enabled != nil {
		resolved.TtydEnabled = *r.global.Defaults.Ttyd.Enabled
	} else {
		resolved.TtydEnabled = true // Default
	}
	resolved.TtydPort = r.global.Defaults.Ttyd.Port
	resolved.TtydOptions = r.global.Defaults.Ttyd.Options
	if r.global.Defaults.Ttyd.Writable != nil {
		resolved.TtydWritable = *r.global.Defaults.Ttyd.Writable
	} else {
		resolved.TtydWritable = true // Default
	}

	// Storage config (global only)
	resolved.StorageWorkspace = r.global.Defaults.Storage.Workspace
	resolved.StorageClaudeHome = r.global.Defaults.Storage.ClaudeHome
	resolved.BranchPrefix = r.global.Defaults.BranchPrefix

	// Sync config from global
	resolved.SyncExclude = r.global.Sync.Exclude
	resolved.SyncUseGitignore = r.global.Sync.UseGitignore
	resolved.SyncCustomDirs = r.global.Sync.CustomDirs

	// Layer 2: Apply template config (overrides global)
	if r.template != nil {
		if r.template.Namespace != "" {
			resolved.Namespace = r.template.Namespace
		}
		if r.template.Image != "" {
			resolved.Image = r.template.Image
		}
		if r.template.Resources.CPU != "" {
			resolved.CPU = r.template.Resources.CPU
		}
		if r.template.Resources.Memory != "" {
			resolved.Memory = r.template.Resources.Memory
		}

		// Merge custom resources from template (overrides global)
		if r.template.Resources.CustomResources != nil {
			for k, v := range r.template.Resources.CustomResources {
				resolved.CustomResources[k] = v
			}
		}

		if r.template.GitSecret != "" {
			resolved.GitSecret = r.template.GitSecret
		}
		if r.template.Branch != "" {
			resolved.Branch = r.template.Branch
		}
		if r.template.GitClone.Depth > 0 {
			resolved.CloneDepth = r.template.GitClone.Depth
		}
		if r.template.GitClone.SingleBranch {
			resolved.SingleBranch = r.template.GitClone.SingleBranch
		}
		if r.template.GitClone.ExtraArgs != "" {
			resolved.GitCloneArgs = r.template.GitClone.ExtraArgs
		}
		if r.template.Repo != "" {
			resolved.Repo = r.template.Repo
		}
		if len(r.template.Command) > 0 {
			// Convert []string to space-separated string
			resolved.Command = joinCommand(r.template.Command)
		}

		// Ttyd config from template
		if r.template.Ttyd.Enabled != nil {
			resolved.TtydEnabled = *r.template.Ttyd.Enabled
		}
		if r.template.Ttyd.Port != 0 {
			resolved.TtydPort = r.template.Ttyd.Port
		}
		if r.template.Ttyd.Options != "" {
			resolved.TtydOptions = r.template.Ttyd.Options
		}
		if r.template.Ttyd.Writable != nil {
			resolved.TtydWritable = *r.template.Ttyd.Writable
		}

		// Sync config from template (overrides global)
		if len(r.template.Sync.Exclude) > 0 {
			resolved.SyncExclude = r.template.Sync.Exclude
		}
		if r.template.Sync.UseGitignore != nil {
			resolved.SyncUseGitignore = r.template.Sync.UseGitignore
		}
		if len(r.template.Sync.CustomDirs) > 0 {
			resolved.SyncCustomDirs = r.template.Sync.CustomDirs
		}
		if r.template.ClaudeAuth != nil {
			resolved.ClaudeAuth = r.template.ClaudeAuth
		}
	}

	return resolved
}

// joinCommand joins command slice into a space-separated string
func joinCommand(cmd []string) string {
	if len(cmd) == 0 {
		return ""
	}
	result := ""
	for i, part := range cmd {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}
