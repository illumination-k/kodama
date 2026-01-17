package initcontainer

import (
	corev1 "k8s.io/api/core/v1"
)

// InstallerConfig represents configuration for an init container installer
type InstallerConfig interface {
	// Name returns the init container name
	Name() string

	// Image returns the container image to use
	Image() string

	// Command returns the command to execute
	Command() []string

	// Args returns the command arguments (script content)
	Args() []string

	// VolumeMounts returns required volume mounts
	VolumeMounts() []corev1.VolumeMount

	// EnvVars returns environment variables needed for the installer
	EnvVars() []corev1.EnvVar

	// StartMessage returns the message logged when installation starts
	StartMessage() string

	// CompletionMessage returns the message logged when installation completes
	CompletionMessage() string
}

// Builder builds init containers from installer configurations
type Builder struct{}

// NewBuilder creates a new init container builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Build creates a Kubernetes init container from an installer config
func (b *Builder) Build(config InstallerConfig) corev1.Container {
	return corev1.Container{
		Name:         config.Name(),
		Image:        config.Image(),
		Command:      config.Command(),
		Args:         config.Args(),
		VolumeMounts: config.VolumeMounts(),
		Env:          config.EnvVars(),
	}
}

// BuildScript constructs a bash script with logging messages
func BuildScript(startMsg, completionMsg string, commands ...string) string {
	script := "set -e\n"
	if startMsg != "" {
		script += "echo \"" + startMsg + "\"\n"
	}

	for _, cmd := range commands {
		script += cmd + "\n"
	}

	if completionMsg != "" {
		script += "echo \"" + completionMsg + "\"\n"
	}

	return script
}
