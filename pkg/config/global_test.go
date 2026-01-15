package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultGlobalConfig(t *testing.T) {
	config := DefaultGlobalConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "default", config.Defaults.Namespace)
	assert.Equal(t, "ghcr.io/illumination-k/kodama:latest", config.Defaults.Image)
	assert.Equal(t, "1", config.Defaults.Resources.CPU)
	assert.Equal(t, "2Gi", config.Defaults.Resources.Memory)
	assert.Equal(t, "10Gi", config.Defaults.Storage.Workspace)
	assert.Equal(t, "1Gi", config.Defaults.Storage.ClaudeHome)
	assert.Equal(t, "kodama/", config.Defaults.BranchPrefix)
}

func TestGlobalConfig_Merge(t *testing.T) {
	base := DefaultGlobalConfig()

	override := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "custom-namespace",
			Image:     "custom-image:latest",
			Resources: ResourceConfig{
				CPU:    "4",
				Memory: "8Gi",
			},
			Storage: StorageConfig{
				Workspace:  "50Gi",
				ClaudeHome: "5Gi",
			},
			BranchPrefix: "feature/",
		},
		Git: GitConfig{
			SecretName: "my-git-secret",
		},
	}

	base.Merge(override)

	assert.Equal(t, "custom-namespace", base.Defaults.Namespace)
	assert.Equal(t, "custom-image:latest", base.Defaults.Image)
	assert.Equal(t, "4", base.Defaults.Resources.CPU)
	assert.Equal(t, "8Gi", base.Defaults.Resources.Memory)
	assert.Equal(t, "50Gi", base.Defaults.Storage.Workspace)
	assert.Equal(t, "5Gi", base.Defaults.Storage.ClaudeHome)
	assert.Equal(t, "feature/", base.Defaults.BranchPrefix)
	assert.Equal(t, "my-git-secret", base.Git.SecretName)
}

func TestGlobalConfig_MergePartial(t *testing.T) {
	base := DefaultGlobalConfig()

	// Override only some fields
	override := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "new-namespace",
			Resources: ResourceConfig{
				CPU: "2",
			},
		},
	}

	base.Merge(override)

	// Overridden fields
	assert.Equal(t, "new-namespace", base.Defaults.Namespace)
	assert.Equal(t, "2", base.Defaults.Resources.CPU)

	// Non-overridden fields keep defaults
	assert.Equal(t, "ghcr.io/illumination-k/kodama:latest", base.Defaults.Image)
	assert.Equal(t, "2Gi", base.Defaults.Resources.Memory)
	assert.Equal(t, "10Gi", base.Defaults.Storage.Workspace)
}

func TestGlobalConfig_MergeEmpty(t *testing.T) {
	base := DefaultGlobalConfig()
	original := *base // Make a copy

	// Merge with empty config should not change anything
	empty := &GlobalConfig{}
	base.Merge(empty)

	assert.Equal(t, original.Defaults.Namespace, base.Defaults.Namespace)
	assert.Equal(t, original.Defaults.Image, base.Defaults.Image)
	assert.Equal(t, original.Defaults.Resources.CPU, base.Defaults.Resources.CPU)
}
