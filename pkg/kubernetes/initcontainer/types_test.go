package initcontainer

import (
	"strings"
	"testing"
)

func TestBuildScript(t *testing.T) {
	tests := []struct {
		name           string
		startMsg       string
		completionMsg  string
		commands       []string
		expectedParts  []string
		unexpectedPart string
	}{
		{
			name:          "Full script with messages",
			startMsg:      "Starting installation...",
			completionMsg: "Installation complete",
			commands: []string{
				"apt-get update",
				"apt-get install -y curl",
			},
			expectedParts: []string{
				"set -e",
				"echo \"Starting installation...\"",
				"apt-get update",
				"apt-get install -y curl",
				"echo \"Installation complete\"",
			},
		},
		{
			name:          "Script without start message",
			startMsg:      "",
			completionMsg: "Done",
			commands: []string{
				"command1",
				"command2",
			},
			expectedParts: []string{
				"set -e",
				"command1",
				"command2",
				"echo \"Done\"",
			},
			unexpectedPart: "echo \"\"",
		},
		{
			name:          "Script without completion message",
			startMsg:      "Starting",
			completionMsg: "",
			commands: []string{
				"command1",
			},
			expectedParts: []string{
				"set -e",
				"echo \"Starting\"",
				"command1",
			},
		},
		{
			name:          "Script without any messages",
			startMsg:      "",
			completionMsg: "",
			commands: []string{
				"command1",
				"command2",
			},
			expectedParts: []string{
				"set -e",
				"command1",
				"command2",
			},
		},
		{
			name:          "Empty script",
			startMsg:      "",
			completionMsg: "",
			commands:      []string{},
			expectedParts: []string{
				"set -e",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := BuildScript(tt.startMsg, tt.completionMsg, tt.commands...)

			// Verify expected parts are present
			for _, part := range tt.expectedParts {
				if !strings.Contains(script, part) {
					t.Errorf("Script missing expected part: %s\nScript:\n%s", part, script)
				}
			}

			// Verify unexpected parts are not present
			if tt.unexpectedPart != "" {
				if strings.Contains(script, tt.unexpectedPart) {
					t.Errorf("Script contains unexpected part: %s\nScript:\n%s", tt.unexpectedPart, script)
				}
			}

			// Verify script starts with set -e
			if !strings.HasPrefix(script, "set -e") {
				t.Errorf("Script should start with 'set -e':\n%s", script)
			}
		})
	}
}

func TestBuildScriptOrder(t *testing.T) {
	script := BuildScript(
		"Start",
		"End",
		"cmd1",
		"cmd2",
		"cmd3",
	)

	// Verify order of components
	startIdx := strings.Index(script, "echo \"Start\"")
	cmd1Idx := strings.Index(script, "cmd1")
	cmd2Idx := strings.Index(script, "cmd2")
	cmd3Idx := strings.Index(script, "cmd3")
	endIdx := strings.Index(script, "echo \"End\"")

	if startIdx == -1 || cmd1Idx == -1 || cmd2Idx == -1 || cmd3Idx == -1 || endIdx == -1 {
		t.Fatal("Script missing expected components")
	}

	if startIdx >= cmd1Idx || cmd1Idx >= cmd2Idx || cmd2Idx >= cmd3Idx || cmd3Idx >= endIdx {
		t.Errorf("Script components not in expected order:\nScript:\n%s", script)
	}
}

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	if builder == nil {
		t.Error("NewBuilder should return non-nil builder")
	}
}

func TestBuildCombined(t *testing.T) {
	builder := NewBuilder()

	claudeConfig := NewClaudeInstallerConfig("latest", "kodama-bin")
	ttydConfig := NewTtydInstallerConfig("1.7.7", "kodama-bin")

	container := builder.BuildCombined("tools-installer", claudeConfig, ttydConfig)

	// Verify container name
	if container.Name != "tools-installer" {
		t.Errorf("Expected container name 'tools-installer', got '%s'", container.Name)
	}

	// Verify image (should use first config's image)
	if container.Image != "ubuntu:24.04" {
		t.Errorf("Expected container image 'ubuntu:24.04', got '%s'", container.Image)
	}

	// Verify command
	if len(container.Command) != 2 || container.Command[0] != "/bin/bash" || container.Command[1] != "-c" {
		t.Errorf("Expected [/bin/bash -c], got %v", container.Command)
	}

	// Verify combined script contains both installers' messages
	if len(container.Args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(container.Args))
	}

	script := container.Args[0]

	// Check for Claude installer messages
	if !strings.Contains(script, "Installing Claude Code CLI...") {
		t.Error("Combined script missing Claude installer start message")
	}
	if !strings.Contains(script, "Claude Code installation complete") {
		t.Error("Combined script missing Claude installer completion message")
	}

	// Check for ttyd installer messages
	if !strings.Contains(script, "Installing ttyd...") {
		t.Error("Combined script missing ttyd installer start message")
	}
	if !strings.Contains(script, "ttyd installation complete") {
		t.Error("Combined script missing ttyd installer completion message")
	}

	// Verify volume mounts are merged (both use kodama-bin)
	if len(container.VolumeMounts) != 1 {
		t.Errorf("Expected 1 volume mount (deduplicated), got %d", len(container.VolumeMounts))
	}

	if container.VolumeMounts[0].Name != "kodama-bin" {
		t.Errorf("Expected volume mount 'kodama-bin', got '%s'", container.VolumeMounts[0].Name)
	}
}

func TestBuildCombinedEmpty(t *testing.T) {
	builder := NewBuilder()
	container := builder.BuildCombined("empty-installer")

	if container.Name != "empty-installer" {
		t.Errorf("Expected container name 'empty-installer', got '%s'", container.Name)
	}
}

func TestBuildCombinedSingle(t *testing.T) {
	builder := NewBuilder()
	claudeConfig := NewClaudeInstallerConfig("latest", "kodama-bin")

	container := builder.BuildCombined("single-installer", claudeConfig)

	if container.Name != "single-installer" {
		t.Errorf("Expected container name 'single-installer', got '%s'", container.Name)
	}

	if len(container.Args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(container.Args))
	}

	script := container.Args[0]
	if !strings.Contains(script, "Installing Claude Code CLI...") {
		t.Error("Script missing Claude installer start message")
	}
}

func TestExtractCommands(t *testing.T) {
	script := `set -e
echo "Starting..."
apt-get update
apt-get install -y curl
echo "Done"`

	commands := extractCommands(script)

	expected := []string{
		"apt-get update",
		"apt-get install -y curl",
	}

	if len(commands) != len(expected) {
		t.Errorf("Expected %d commands, got %d", len(expected), len(commands))
	}

	for i, cmd := range expected {
		if i >= len(commands) || commands[i] != cmd {
			t.Errorf("Expected command[%d] = '%s', got '%s'", i, cmd, commands[i])
		}
	}
}

func TestMergeVolumeMounts(t *testing.T) {
	claudeConfig := NewClaudeInstallerConfig("latest", "kodama-bin")
	ttydConfig := NewTtydInstallerConfig("1.7.7", "kodama-bin")

	configs := []InstallerConfig{claudeConfig, ttydConfig}
	mounts := mergeVolumeMounts(configs)

	// Both use the same volume, should be deduplicated
	if len(mounts) != 1 {
		t.Errorf("Expected 1 volume mount (deduplicated), got %d", len(mounts))
	}

	if mounts[0].Name != "kodama-bin" {
		t.Errorf("Expected volume mount 'kodama-bin', got '%s'", mounts[0].Name)
	}
}

func TestMergeEnvVars(t *testing.T) {
	claudeConfig := NewClaudeInstallerConfig("latest", "kodama-bin")
	ttydConfig := NewTtydInstallerConfig("1.7.7", "kodama-bin")

	configs := []InstallerConfig{claudeConfig, ttydConfig}
	envVars := mergeEnvVars(configs)

	// Neither installer has env vars
	if len(envVars) != 0 {
		t.Errorf("Expected 0 env vars, got %d", len(envVars))
	}
}
