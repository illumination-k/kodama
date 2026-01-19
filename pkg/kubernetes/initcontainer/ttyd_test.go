package initcontainer

import (
	"strings"
	"testing"
)

func TestTtydInstallerConfig(t *testing.T) {
	config := NewTtydInstallerConfig("1.7.7", "kodama-bin")

	// Test basic properties
	if config.Name() != "ttyd-installer" {
		t.Errorf("Expected name 'ttyd-installer', got '%s'", config.Name())
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
		"Installing ttyd...",
		"apt-get update",
		"curl -fsSL https://github.com/tsl0922/ttyd/releases/download/1.7.7/ttyd.x86_64",
		"chmod +x /tmp/ttyd",
		"cp /tmp/ttyd /kodama/bin/ttyd",
		"ttyd installation complete",
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

func TestTtydInstallerConfigDefaultValues(t *testing.T) {
	// Test with empty values (should use defaults)
	config := NewTtydInstallerConfig("", "")

	if config.Version != "1.7.7" {
		t.Errorf("Expected default version '1.7.7', got '%s'", config.Version)
	}

	if config.BinVolumeName != "kodama-bin" {
		t.Errorf("Expected default bin volume 'kodama-bin', got '%s'", config.BinVolumeName)
	}
}

func TestTtydInstallerCustomVersion(t *testing.T) {
	config := NewTtydInstallerConfig("1.8.0", "kodama-bin")

	args := config.Args()
	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}

	script := args[0]
	// Verify custom version is in the download URL
	expectedURL := "https://github.com/tsl0922/ttyd/releases/download/1.8.0/ttyd.x86_64"
	if !strings.Contains(script, expectedURL) {
		t.Errorf("Script missing custom version URL: %s", expectedURL)
	}
}

func TestTtydInstallerBuilder(t *testing.T) {
	builder := NewBuilder()
	config := NewTtydInstallerConfig("1.7.7", "kodama-bin")

	container := builder.Build(config)

	// Verify container structure
	if container.Name != "ttyd-installer" {
		t.Errorf("Expected container name 'ttyd-installer', got '%s'", container.Name)
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
