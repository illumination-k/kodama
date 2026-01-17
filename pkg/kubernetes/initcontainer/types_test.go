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

	if !(startIdx < cmd1Idx && cmd1Idx < cmd2Idx && cmd2Idx < cmd3Idx && cmd3Idx < endIdx) {
		t.Errorf("Script components not in expected order:\nScript:\n%s", script)
	}
}

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	if builder == nil {
		t.Error("NewBuilder should return non-nil builder")
	}
}
