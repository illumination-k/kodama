package secretfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFiles(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.bin")

	if err := os.WriteFile(file1, []byte("text content"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte{0x00, 0x01, 0x02, 0xff}, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		mappings    []FileMapping
		wantFiles   int
		wantErr     bool
		checkBinary bool
	}{
		{
			name: "load single text file",
			mappings: []FileMapping{
				{Source: file1, Destination: "/app/config.txt"},
			},
			wantFiles: 1,
			wantErr:   false,
		},
		{
			name: "load multiple files",
			mappings: []FileMapping{
				{Source: file1, Destination: "/app/config.txt"},
				{Source: file2, Destination: "/app/data.bin"},
			},
			wantFiles:   2,
			wantErr:     false,
			checkBinary: true,
		},
		{
			name: "missing file (soft failure)",
			mappings: []FileMapping{
				{Source: file1, Destination: "/app/config.txt"},
				{Source: filepath.Join(tempDir, "nonexistent.txt"), Destination: "/app/missing.txt"},
			},
			wantFiles: 1, // Only file1 should be loaded
			wantErr:   false,
		},
		{
			name:      "empty mappings",
			mappings:  []FileMapping{},
			wantFiles: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := LoadFiles(tt.mappings)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantFiles {
				t.Errorf("LoadFiles() loaded %d files, want %d", len(result), tt.wantFiles)
			}

			// Verify text content
			if content, ok := result["/app/config.txt"]; ok {
				if string(content) != "text content" {
					t.Errorf("LoadFiles() content mismatch, got %q, want %q", string(content), "text content")
				}
			}

			// Verify binary content
			if tt.checkBinary {
				if content, ok := result["/app/data.bin"]; ok {
					expected := []byte{0x00, 0x01, 0x02, 0xff}
					if len(content) != len(expected) {
						t.Errorf("LoadFiles() binary content length mismatch, got %d, want %d", len(content), len(expected))
					}
					for i, b := range content {
						if i < len(expected) && b != expected[i] {
							t.Errorf("LoadFiles() binary content[%d] = %x, want %x", i, b, expected[i])
						}
					}
				}
			}
		})
	}
}

func TestExpandHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot determine home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde only",
			path: "~",
			want: home,
		},
		{
			name: "tilde with path",
			path: "~/.ssh/id_rsa",
			want: filepath.Join(home, ".ssh/id_rsa"),
		},
		{
			name: "absolute path (no expansion)",
			path: "/etc/config",
			want: "/etc/config",
		},
		{
			name: "relative path (no expansion)",
			path: "config/file.txt",
			want: "config/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHomeDir(tt.path)
			if got != tt.want {
				t.Errorf("expandHomeDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
