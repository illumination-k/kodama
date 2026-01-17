package config

import (
	"testing"
)

func TestConfigResolver_Resolve_GlobalOnly(t *testing.T) {
	// Test with only global config (no template)
	global := DefaultGlobalConfig()
	resolver := NewConfigResolver(global, nil)

	resolved := resolver.Resolve()

	// Verify global defaults are applied
	if resolved.Namespace != "default" {
		t.Errorf("expected namespace 'default', got '%s'", resolved.Namespace)
	}
	if resolved.Image != "ghcr.io/illumination-k/kodama:latest" {
		t.Errorf("expected default image, got '%s'", resolved.Image)
	}
	if resolved.CPU != "1" {
		t.Errorf("expected CPU '1', got '%s'", resolved.CPU)
	}
	if resolved.Memory != "2Gi" {
		t.Errorf("expected memory '2Gi', got '%s'", resolved.Memory)
	}
	if !resolved.TtydEnabled {
		t.Error("expected ttyd enabled by default")
	}
	if resolved.TtydPort != 7681 {
		t.Errorf("expected ttyd port 7681, got %d", resolved.TtydPort)
	}
	if !resolved.TtydWritable {
		t.Error("expected ttyd writable by default")
	}
	if resolved.StorageWorkspace != "10Gi" {
		t.Errorf("expected workspace storage '10Gi', got '%s'", resolved.StorageWorkspace)
	}
	if resolved.StorageClaudeHome != "1Gi" {
		t.Errorf("expected claude home storage '1Gi', got '%s'", resolved.StorageClaudeHome)
	}
	if resolved.BranchPrefix != "kodama/" {
		t.Errorf("expected branch prefix 'kodama/', got '%s'", resolved.BranchPrefix)
	}
}

func TestConfigResolver_Resolve_GlobalWithCustomValues(t *testing.T) {
	// Test with customized global config
	global := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "custom-namespace",
			Image:     "custom-image:v1",
			Resources: ResourceConfig{
				CPU:    "2",
				Memory: "4Gi",
				CustomResources: map[string]string{
					"nvidia.com/gpu": "1",
				},
			},
			Storage: StorageConfig{
				Workspace:  "20Gi",
				ClaudeHome: "2Gi",
			},
			BranchPrefix: "custom/",
		},
		Git: GitConfig{
			SecretName: "my-git-secret",
		},
		Sync: GlobalSyncConfig{
			Exclude: []string{"*.log", "tmp/"},
		},
	}

	resolver := NewConfigResolver(global, nil)
	resolved := resolver.Resolve()

	if resolved.Namespace != "custom-namespace" {
		t.Errorf("expected namespace 'custom-namespace', got '%s'", resolved.Namespace)
	}
	if resolved.Image != "custom-image:v1" {
		t.Errorf("expected image 'custom-image:v1', got '%s'", resolved.Image)
	}
	if resolved.CPU != "2" {
		t.Errorf("expected CPU '2', got '%s'", resolved.CPU)
	}
	if resolved.Memory != "4Gi" {
		t.Errorf("expected memory '4Gi', got '%s'", resolved.Memory)
	}
	if resolved.CustomResources["nvidia.com/gpu"] != "1" {
		t.Errorf("expected GPU '1', got '%s'", resolved.CustomResources["nvidia.com/gpu"])
	}
	if resolved.GitSecret != "my-git-secret" {
		t.Errorf("expected git secret 'my-git-secret', got '%s'", resolved.GitSecret)
	}
	if resolved.BranchPrefix != "custom/" {
		t.Errorf("expected branch prefix 'custom/', got '%s'", resolved.BranchPrefix)
	}
	if len(resolved.SyncExclude) != 2 {
		t.Errorf("expected 2 exclude patterns, got %d", len(resolved.SyncExclude))
	}
}

func TestConfigResolver_Resolve_TemplateOverridesGlobal(t *testing.T) {
	// Test that template config overrides global config
	global := DefaultGlobalConfig()
	global.Defaults.Namespace = "global-ns"
	global.Defaults.Image = "global-image:v1"
	global.Defaults.Resources.CPU = "1"
	global.Defaults.Resources.Memory = "2Gi"

	template := &SessionConfig{
		Namespace: "template-ns",
		Image:     "template-image:v2",
		Resources: ResourceConfig{
			CPU:    "4",
			Memory: "8Gi",
		},
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Template should override global
	if resolved.Namespace != "template-ns" {
		t.Errorf("expected namespace 'template-ns', got '%s'", resolved.Namespace)
	}
	if resolved.Image != "template-image:v2" {
		t.Errorf("expected image 'template-image:v2', got '%s'", resolved.Image)
	}
	if resolved.CPU != "4" {
		t.Errorf("expected CPU '4', got '%s'", resolved.CPU)
	}
	if resolved.Memory != "8Gi" {
		t.Errorf("expected memory '8Gi', got '%s'", resolved.Memory)
	}
}

func TestConfigResolver_Resolve_TemplatePartialOverride(t *testing.T) {
	// Test that template only overrides specified fields
	global := DefaultGlobalConfig()
	global.Defaults.Namespace = "global-ns"
	global.Defaults.Image = "global-image:v1"
	global.Defaults.Resources.CPU = "1"

	template := &SessionConfig{
		Namespace: "template-ns",
		// Image not specified - should use global
		// CPU not specified - should use global
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Template overrides namespace
	if resolved.Namespace != "template-ns" {
		t.Errorf("expected namespace 'template-ns', got '%s'", resolved.Namespace)
	}
	// Global provides image and CPU
	if resolved.Image != "global-image:v1" {
		t.Errorf("expected image 'global-image:v1', got '%s'", resolved.Image)
	}
	if resolved.CPU != "1" {
		t.Errorf("expected CPU '1', got '%s'", resolved.CPU)
	}
}

func TestConfigResolver_Resolve_CustomResourcesMerge(t *testing.T) {
	// Test that custom resources are properly merged
	global := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "default",
			Image:     "test:v1",
			Resources: ResourceConfig{
				CPU:    "1",
				Memory: "2Gi",
				CustomResources: map[string]string{
					"nvidia.com/gpu": "1",
					"amd.com/gpu":    "0",
				},
			},
		},
	}

	template := &SessionConfig{
		Resources: ResourceConfig{
			CustomResources: map[string]string{
				"nvidia.com/gpu":  "2", // Override global
				"intel.com/fpga":  "1", // Add new
				// amd.com/gpu not specified - should be overridden
			},
		},
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Template should completely replace custom resources (not merge)
	if resolved.CustomResources["nvidia.com/gpu"] != "2" {
		t.Errorf("expected GPU '2', got '%s'", resolved.CustomResources["nvidia.com/gpu"])
	}
	if resolved.CustomResources["intel.com/fpga"] != "1" {
		t.Errorf("expected FPGA '1', got '%s'", resolved.CustomResources["intel.com/fpga"])
	}
	// AMD GPU should NOT be present (template replaces, doesn't merge)
	if _, exists := resolved.CustomResources["amd.com/gpu"]; exists {
		t.Error("amd.com/gpu should not exist after template override")
	}
}

func TestConfigResolver_Resolve_TtydConfig(t *testing.T) {
	// Test ttyd configuration merging
	ttydDisabled := false
	ttydReadonly := false

	global := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "default",
			Image:     "test:v1",
			Ttyd: TtydConfig{
				Enabled:  &ttydDisabled,
				Port:     9000,
				Options:  "--global-opt",
				Writable: &ttydReadonly,
			},
		},
	}

	ttydEnabled := true
	ttydWritable := true

	template := &SessionConfig{
		Ttyd: TtydConfig{
			Enabled:  &ttydEnabled,
			Port:     8080,
			Options:  "--template-opt",
			Writable: &ttydWritable,
		},
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	if !resolved.TtydEnabled {
		t.Error("expected ttyd enabled from template")
	}
	if resolved.TtydPort != 8080 {
		t.Errorf("expected ttyd port 8080, got %d", resolved.TtydPort)
	}
	if resolved.TtydOptions != "--template-opt" {
		t.Errorf("expected ttyd options '--template-opt', got '%s'", resolved.TtydOptions)
	}
	if !resolved.TtydWritable {
		t.Error("expected ttyd writable from template")
	}
}

func TestConfigResolver_Resolve_GitCloneConfig(t *testing.T) {
	// Test git clone configuration
	template := &SessionConfig{
		Repo:   "https://github.com/example/repo.git",
		Branch: "feature/test",
		GitClone: GitCloneConfig{
			Depth:        1,
			SingleBranch: true,
			ExtraArgs:    "--recurse-submodules",
		},
	}

	resolver := NewConfigResolver(DefaultGlobalConfig(), template)
	resolved := resolver.Resolve()

	if resolved.Repo != "https://github.com/example/repo.git" {
		t.Errorf("expected repo URL, got '%s'", resolved.Repo)
	}
	if resolved.Branch != "feature/test" {
		t.Errorf("expected branch 'feature/test', got '%s'", resolved.Branch)
	}
	if resolved.CloneDepth != 1 {
		t.Errorf("expected clone depth 1, got %d", resolved.CloneDepth)
	}
	if !resolved.SingleBranch {
		t.Error("expected single branch clone")
	}
	if resolved.GitCloneArgs != "--recurse-submodules" {
		t.Errorf("expected git clone args, got '%s'", resolved.GitCloneArgs)
	}
}

func TestConfigResolver_Resolve_CommandConfig(t *testing.T) {
	// Test command configuration
	template := &SessionConfig{
		Command: []string{"bash", "-c", "echo hello"},
	}

	resolver := NewConfigResolver(DefaultGlobalConfig(), template)
	resolved := resolver.Resolve()

	expectedCommand := "bash -c echo hello"
	if resolved.Command != expectedCommand {
		t.Errorf("expected command '%s', got '%s'", expectedCommand, resolved.Command)
	}
}

func TestConfigResolver_Resolve_SyncConfig(t *testing.T) {
	// Test sync configuration merging
	useGitignoreTrue := true
	useGitignoreFalse := false

	global := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "default",
			Image:     "test:v1",
		},
		Sync: GlobalSyncConfig{
			UseGitignore: &useGitignoreTrue,
			Exclude:      []string{"*.log"},
			CustomDirs: []CustomDirSync{
				{LocalPath: "~/.config", RemotePath: "/home/user/.config"},
			},
		},
	}

	template := &SessionConfig{
		Sync: SyncConfig{
			UseGitignore: &useGitignoreFalse,
			Exclude:      []string{"*.tmp", "build/"},
			CustomDirs: []CustomDirSync{
				{LocalPath: "~/.ssh", RemotePath: "/home/user/.ssh"},
			},
		},
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Template should override global sync config
	if resolved.SyncUseGitignore == nil || *resolved.SyncUseGitignore {
		t.Error("expected useGitignore false from template")
	}
	if len(resolved.SyncExclude) != 2 {
		t.Errorf("expected 2 exclude patterns, got %d", len(resolved.SyncExclude))
	}
	if resolved.SyncExclude[0] != "*.tmp" || resolved.SyncExclude[1] != "build/" {
		t.Errorf("unexpected exclude patterns: %v", resolved.SyncExclude)
	}
	if len(resolved.SyncCustomDirs) != 1 {
		t.Errorf("expected 1 custom dir, got %d", len(resolved.SyncCustomDirs))
	}
	if resolved.SyncCustomDirs[0].LocalPath != "~/.ssh" {
		t.Errorf("expected custom dir '~/.ssh', got '%s'", resolved.SyncCustomDirs[0].LocalPath)
	}
}

func TestConfigResolver_Resolve_ClaudeAuthConfig(t *testing.T) {
	// Test Claude auth configuration
	template := &SessionConfig{
		ClaudeAuth: &ClaudeAuthOverride{
			AuthType:   "token",
			SecretName: "my-claude-secret",
			Profile:    "production",
		},
	}

	resolver := NewConfigResolver(DefaultGlobalConfig(), template)
	resolved := resolver.Resolve()

	if resolved.ClaudeAuth == nil {
		t.Fatal("expected ClaudeAuth to be set")
	}
	if resolved.ClaudeAuth.AuthType != "token" {
		t.Errorf("expected auth type 'token', got '%s'", resolved.ClaudeAuth.AuthType)
	}
	if resolved.ClaudeAuth.SecretName != "my-claude-secret" {
		t.Errorf("expected secret name 'my-claude-secret', got '%s'", resolved.ClaudeAuth.SecretName)
	}
	if resolved.ClaudeAuth.Profile != "production" {
		t.Errorf("expected profile 'production', got '%s'", resolved.ClaudeAuth.Profile)
	}
}

func TestConfigResolver_Resolve_EmptyTemplateFields(t *testing.T) {
	// Test that empty template fields don't override global config
	global := DefaultGlobalConfig()
	global.Defaults.Namespace = "global-ns"
	global.Defaults.Image = "global-image:v1"

	template := &SessionConfig{
		Namespace: "", // Empty - should use global
		Image:     "", // Empty - should use global
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Empty strings should not override
	if resolved.Namespace != "global-ns" {
		t.Errorf("expected namespace 'global-ns', got '%s'", resolved.Namespace)
	}
	if resolved.Image != "global-image:v1" {
		t.Errorf("expected image 'global-image:v1', got '%s'", resolved.Image)
	}
}

func TestJoinCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single element",
			input:    []string{"bash"},
			expected: "bash",
		},
		{
			name:     "multiple elements",
			input:    []string{"bash", "-c", "echo hello"},
			expected: "bash -c echo hello",
		},
		{
			name:     "with quotes",
			input:    []string{"bash", "-c", "echo 'hello world'"},
			expected: "bash -c echo 'hello world'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinCommand(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
