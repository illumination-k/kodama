package initcontainer

import (
	corev1 "k8s.io/api/core/v1"
)

// ClaudeInstallerConfig configures Claude Code CLI installation
type ClaudeInstallerConfig struct {
	// Version specifies the Claude Code version to install (e.g., "latest")
	Version string

	// BinVolumeName is the name of the volume to mount at /kodama/bin
	BinVolumeName string
}

// NewClaudeInstallerConfig creates a new Claude installer configuration
func NewClaudeInstallerConfig(version, binVolumeName string) *ClaudeInstallerConfig {
	if version == "" {
		version = "latest"
	}
	if binVolumeName == "" {
		binVolumeName = "kodama-bin"
	}

	return &ClaudeInstallerConfig{
		Version:       version,
		BinVolumeName: binVolumeName,
	}
}

// Name returns the init container name
func (c *ClaudeInstallerConfig) Name() string {
	return "claude-installer"
}

// Image returns the container image
func (c *ClaudeInstallerConfig) Image() string {
	return "ubuntu:24.04"
}

// Command returns the shell command
func (c *ClaudeInstallerConfig) Command() []string {
	return []string{"/bin/bash", "-c"}
}

// Args returns the installation script
func (c *ClaudeInstallerConfig) Args() []string {
	script := BuildScript(
		c.StartMessage(),
		c.CompletionMessage(),
		"apt-get update -qq && apt-get install -y -qq curl ca-certificates",
		"curl -fsSL https://claude.ai/install.sh | bash -s "+c.Version,
		"mkdir -p /kodama/bin",
		"cp -rL /root/.local/bin/* /kodama/bin/",
	)
	return []string{script}
}

// VolumeMounts returns required volume mounts
func (c *ClaudeInstallerConfig) VolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      c.BinVolumeName,
			MountPath: "/kodama/bin",
		},
	}
}

// EnvVars returns environment variables (none needed for Claude installer)
func (c *ClaudeInstallerConfig) EnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{}
}

// StartMessage returns the installation start message
func (c *ClaudeInstallerConfig) StartMessage() string {
	return "Installing Claude Code CLI..."
}

// CompletionMessage returns the installation completion message
func (c *ClaudeInstallerConfig) CompletionMessage() string {
	return "Claude Code installation complete"
}
