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

// BuildCombined creates a single init container that runs multiple installers sequentially
// This is more efficient than creating separate init containers for each installer.
// All installers must use the same image (first config's image is used).
func (b *Builder) BuildCombined(name string, configs ...InstallerConfig) corev1.Container {
	if len(configs) == 0 {
		return corev1.Container{Name: name}
	}

	// Use first config's image and command
	image := configs[0].Image()
	command := configs[0].Command()

	// Combine all scripts into one
	var combinedScript string
	combinedScript = "set -e\n"

	for _, config := range configs {
		// Extract the actual commands from each installer's script
		args := config.Args()
		if len(args) > 0 {
			script := args[0]
			// Add start message
			if msg := config.StartMessage(); msg != "" {
				combinedScript += "echo \"" + msg + "\"\n"
			}
			// Extract commands (skip the "set -e" line and echo statements)
			lines := extractCommands(script)
			for _, line := range lines {
				combinedScript += line + "\n"
			}
			// Add completion message
			if msg := config.CompletionMessage(); msg != "" {
				combinedScript += "echo \"" + msg + "\"\n"
			}
		}
	}

	// Merge volume mounts (deduplicate by name)
	volumeMounts := mergeVolumeMounts(configs)

	// Merge environment variables (deduplicate by name)
	envVars := mergeEnvVars(configs)

	return corev1.Container{
		Name:         name,
		Image:        image,
		Command:      command,
		Args:         []string{combinedScript},
		VolumeMounts: volumeMounts,
		Env:          envVars,
	}
}

// extractCommands extracts actual commands from a script, skipping "set -e" and echo statements
func extractCommands(script string) []string {
	var commands []string
	lines := splitLines(script)

	for _, line := range lines {
		trimmed := trimSpace(line)
		// Skip empty lines, "set -e", and echo statements
		if trimmed == "" || trimmed == "set -e" || startsWith(trimmed, "echo ") {
			continue
		}
		commands = append(commands, line)
	}

	return commands
}

// mergeVolumeMounts merges volume mounts from multiple configs, deduplicating by name
func mergeVolumeMounts(configs []InstallerConfig) []corev1.VolumeMount {
	seen := make(map[string]corev1.VolumeMount)

	for _, config := range configs {
		for _, mount := range config.VolumeMounts() {
			if _, exists := seen[mount.Name]; !exists {
				seen[mount.Name] = mount
			}
		}
	}

	result := make([]corev1.VolumeMount, 0, len(seen))
	for _, mount := range seen {
		result = append(result, mount)
	}

	return result
}

// mergeEnvVars merges environment variables from multiple configs, deduplicating by name
func mergeEnvVars(configs []InstallerConfig) []corev1.EnvVar {
	seen := make(map[string]corev1.EnvVar)

	for _, config := range configs {
		for _, envVar := range config.EnvVars() {
			if _, exists := seen[envVar.Name]; !exists {
				seen[envVar.Name] = envVar
			}
		}
	}

	result := make([]corev1.EnvVar, 0, len(seen))
	for _, envVar := range seen {
		result = append(result, envVar)
	}

	return result
}

// Helper functions to avoid importing strings package
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
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
