package initcontainer

import (
	"strings"
	"testing"
)

func TestClaudeInstallerConfig(t *testing.T) {
	config := NewClaudeInstallerConfig("latest", "kodama-bin")

	// Test basic properties
	if config.Name() != "claude-installer" {
		t.Errorf("Expected name 'claude-installer', got '%s'", config.Name())
	}

	if config.Image() != "ubuntu:24.04" {
		t.Errorf("Expected image 'ubuntu:24.04', got '%s'", config.Image())
	}

	// Test command
	cmd := config.Command()
	if len(cmd) != 2 || cmd[0] != "/bin/bash" || cmd[1] != "-c" {
		t.Errorf("Expected [/bin/bash -c], got %v", cmd)
	}

	// Test args contain installation script
	args := config.Args()
	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}

	script := args[0]
	// Verify script contains expected commands
	expectedParts := []string{
		"Installing Claude Code CLI...",
		"apt-get update",
		"curl -fsSL https://claude.ai/install.sh",
		"cp -rL /root/.local/bin/* /kodama/bin/",
		"Claude Code installation complete",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("Script missing expected part: %s", part)
		}
	}

	// Test volume mounts
	mounts := config.VolumeMounts()
	if len(mounts) != 1 {
		t.Fatalf("Expected 1 volume mount, got %d", len(mounts))
	}

	if mounts[0].Name != "kodama-bin" || mounts[0].MountPath != "/kodama/bin" {
		t.Errorf("Unexpected volume mount: %+v", mounts[0])
	}

	// Test env vars (should be empty)
	envVars := config.EnvVars()
	if len(envVars) != 0 {
		t.Errorf("Expected 0 env vars, got %d", len(envVars))
	}
}

func TestClaudeInstallerConfigDefaultValues(t *testing.T) {
	// Test with empty values (should use defaults)
	config := NewClaudeInstallerConfig("", "")

	if config.Version != "latest" {
		t.Errorf("Expected default version 'latest', got '%s'", config.Version)
	}

	if config.BinVolumeName != "kodama-bin" {
		t.Errorf("Expected default bin volume 'kodama-bin', got '%s'", config.BinVolumeName)
	}
}

func TestClaudeInstallerBuilder(t *testing.T) {
	builder := NewBuilder()
	config := NewClaudeInstallerConfig("latest", "kodama-bin")

	container := builder.Build(config)

	// Verify container structure
	if container.Name != "claude-installer" {
		t.Errorf("Expected container name 'claude-installer', got '%s'", container.Name)
	}

	if container.Image != "ubuntu:24.04" {
		t.Errorf("Expected container image 'ubuntu:24.04', got '%s'", container.Image)
	}

	if len(container.Command) != 2 {
		t.Errorf("Expected 2 command parts, got %d", len(container.Command))
	}

	if len(container.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(container.Args))
	}

	if len(container.VolumeMounts) != 1 {
		t.Errorf("Expected 1 volume mount, got %d", len(container.VolumeMounts))
	}
}
