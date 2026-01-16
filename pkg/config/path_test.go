package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(string) bool
	}{
		{
			name:      "empty path",
			input:     "",
			wantError: true,
		},
		{
			name:  "tilde only",
			input: "~",
			checkFunc: func(result string) bool {
				return result == homeDir
			},
		},
		{
			name:  "tilde with path",
			input: "~/test/path",
			checkFunc: func(result string) bool {
				expected := filepath.Join(homeDir, "test/path")
				return result == expected
			},
		},
		{
			name:  "absolute path",
			input: "/absolute/path",
			checkFunc: func(result string) bool {
				return result == "/absolute/path"
			},
		},
		{
			name:      "relative path",
			input:     "relative/path",
			checkFunc: filepath.IsAbs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolvePath(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if tt.checkFunc != nil && !tt.checkFunc(result) {
				t.Errorf("Result check failed for input %q, got: %s", tt.input, result)
			}
		})
	}
}

func TestCustomDirSync_Validate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		customDir CustomDirSync
		wantError bool
	}{
		{
			name: "valid config",
			customDir: CustomDirSync{
				Source:      tmpFile,
				Destination: "/root/.bashrc",
			},
			wantError: false,
		},
		{
			name: "empty source",
			customDir: CustomDirSync{
				Source:      "",
				Destination: "/root/.bashrc",
			},
			wantError: true,
		},
		{
			name: "empty destination",
			customDir: CustomDirSync{
				Source:      tmpFile,
				Destination: "",
			},
			wantError: true,
		},
		{
			name: "relative destination",
			customDir: CustomDirSync{
				Source:      tmpFile,
				Destination: "relative/path",
			},
			wantError: true,
		},
		{
			name: "non-existent source",
			customDir: CustomDirSync{
				Source:      "/nonexistent/path",
				Destination: "/root/.bashrc",
			},
			wantError: true,
		},
		{
			name: "recursive with file source",
			customDir: CustomDirSync{
				Source:      tmpFile,
				Destination: "/root/.config",
				Recursive:   true,
			},
			wantError: true,
		},
		{
			name: "recursive with directory source",
			customDir: CustomDirSync{
				Source:      tmpDir,
				Destination: "/root/.config",
				Recursive:   true,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.customDir.Validate()
			if tt.wantError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCustomDirSync_ResolveSource(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name      string
		source    string
		wantError bool
		checkFunc func(string) bool
	}{
		{
			name:   "tilde expansion",
			source: "~/.bashrc",
			checkFunc: func(result string) bool {
				expected := filepath.Join(homeDir, ".bashrc")
				return result == expected
			},
		},
		{
			name:   "absolute path",
			source: "/absolute/path",
			checkFunc: func(result string) bool {
				return result == "/absolute/path"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customDir := CustomDirSync{
				Source:      tt.source,
				Destination: "/root/test",
			}
			result, err := customDir.ResolveSource()
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if tt.checkFunc != nil && !tt.checkFunc(result) {
				t.Errorf("Result check failed for source %q, got: %s", tt.source, result)
			}
		})
	}
}
