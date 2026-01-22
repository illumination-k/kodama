package initcontainer

import (
	"strings"
	"testing"

	"github.com/illumination-k/kodama/pkg/gitcmd"
)

func TestWorkspaceInitializerConfig(t *testing.T) {
	opts := &gitcmd.CloneOptions{
		Depth:        1,
		SingleBranch: true,
		ExtraArgs:    "--quiet",
	}
	config := NewWorkspaceInitializerConfig(
		"https://github.com/example/repo.git",
		"feature-branch",
		opts,
	)

	// Test basic properties
	if config.Name() != "workspace-initializer" {
		t.Errorf("Expected name 'workspace-initializer', got '%s'", config.Name())
	}

	if config.Image() != "ubuntu:24.04" {
		t.Errorf("Expected image 'ubuntu:24.04', got '%s'", config.Image())
	}

	// Test IsEnabled
	if !config.IsEnabled() {
		t.Error("Expected IsEnabled to be true when GitRepo is set")
	}

	// Test command
	cmd := config.Command()
	if len(cmd) != 2 || cmd[0] != "/bin/bash" || cmd[1] != "-c" {
		t.Errorf("Expected [/bin/bash -c], got %v", cmd)
	}

	// Test args contain git initialization script
	args := config.Args()
	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}

	script := args[0]
	// Verify script contains git commands (gitcmd package generates this)
	if !strings.Contains(script, "git clone") {
		t.Error("Script missing 'git clone' command")
	}

	// Test volume mounts
	mounts := config.VolumeMounts()
	if len(mounts) != 1 {
		t.Fatalf("Expected 1 volume mount, got %d", len(mounts))
	}

	if mounts[0].Name != "workspace" || mounts[0].MountPath != "/workspace" {
		t.Errorf("Unexpected volume mount: %+v", mounts[0])
	}

	// Test env vars (should be empty - auth handled via envFrom)
	envVars := config.EnvVars()
	if len(envVars) != 0 {
		t.Fatalf("Expected 0 env vars, got %d", len(envVars))
	}
}

func TestWorkspaceInitializerConfigNoSecret(t *testing.T) {
	config := NewWorkspaceInitializerConfig(
		"https://github.com/example/repo.git",
		"main",
		nil,
	)

	// Test env vars (should be empty without secret)
	envVars := config.EnvVars()
	if len(envVars) != 0 {
		t.Errorf("Expected 0 env vars without secret, got %d", len(envVars))
	}
}

func TestWorkspaceInitializerConfigIsEnabled(t *testing.T) {
	// Test enabled with repo
	enabledConfig := NewWorkspaceInitializerConfig(
		"https://github.com/example/repo.git",
		"main",
		nil,
	)
	if !enabledConfig.IsEnabled() {
		t.Error("Expected IsEnabled to be true when GitRepo is set")
	}

	// Test disabled without repo
	disabledConfig := NewWorkspaceInitializerConfig("", "", nil)
	if disabledConfig.IsEnabled() {
		t.Error("Expected IsEnabled to be false when GitRepo is empty")
	}
}

func TestWorkspaceInitializerConfigWithWorkspaceVolume(t *testing.T) {
	config := NewWorkspaceInitializerConfig(
		"https://github.com/example/repo.git",
		"main",
		nil,
	).WithWorkspaceVolume("custom-workspace")

	mounts := config.VolumeMounts()
	if len(mounts) != 1 {
		t.Fatalf("Expected 1 volume mount, got %d", len(mounts))
	}

	if mounts[0].Name != "custom-workspace" {
		t.Errorf("Expected custom workspace volume 'custom-workspace', got '%s'", mounts[0].Name)
	}
}

func TestWorkspaceInitializerBuilder(t *testing.T) {
	builder := NewBuilder()
	opts := &gitcmd.CloneOptions{
		Depth:        1,
		SingleBranch: true,
	}
	config := NewWorkspaceInitializerConfig(
		"https://github.com/example/repo.git",
		"main",
		opts,
	)

	container := builder.Build(config)

	// Verify container structure
	if container.Name != "workspace-initializer" {
		t.Errorf("Expected container name 'workspace-initializer', got '%s'", container.Name)
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
