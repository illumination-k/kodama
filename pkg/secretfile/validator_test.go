package secretfile

import (
	"strings"
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid absolute path",
			path:    "/etc/config/app.conf",
			wantErr: false,
		},
		{
			name:    "valid root path",
			path:    "/app",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "relative path",
			path:    "config/app.conf",
			wantErr: true,
		},
		{
			name:    "path with directory traversal",
			path:    "/app/../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path starting with ..",
			path:    "../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSecretSize(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string][]byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    map[string][]byte{},
			wantErr: false,
		},
		{
			name: "small files",
			data: map[string][]byte{
				"/app/config.txt": []byte("small content"),
				"/app/data.json":  []byte(`{"key": "value"}`),
			},
			wantErr: false,
		},
		{
			name: "file at limit",
			data: map[string][]byte{
				"/app/large.bin": make([]byte, MaxSecretSize-100), // Leave room for key overhead
			},
			wantErr: false,
		},
		{
			name: "file exceeds limit",
			data: map[string][]byte{
				"/app/toolarge.bin": make([]byte, MaxSecretSize+1),
			},
			wantErr: true,
		},
		{
			name: "multiple files exceed limit",
			data: map[string][]byte{
				"/app/file1.bin": make([]byte, MaxSecretSize/2),
				"/app/file2.bin": make([]byte, MaxSecretSize/2),
				"/app/file3.bin": make([]byte, 1000),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecretSize(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSecretSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings []FileMapping
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "empty mappings",
			mappings: []FileMapping{},
			wantErr:  false,
		},
		{
			name: "valid single mapping",
			mappings: []FileMapping{
				{Source: "~/config.txt", Destination: "/app/config.txt"},
			},
			wantErr: false,
		},
		{
			name: "valid multiple mappings",
			mappings: []FileMapping{
				{Source: "~/config.txt", Destination: "/app/config.txt"},
				{Source: "~/.ssh/id_rsa", Destination: "/root/.ssh/id_rsa"},
			},
			wantErr: false,
		},
		{
			name: "duplicate destinations",
			mappings: []FileMapping{
				{Source: "~/file1.txt", Destination: "/app/config.txt"},
				{Source: "~/file2.txt", Destination: "/app/config.txt"},
			},
			wantErr: true,
			errMsg:  "duplicate destination",
		},
		{
			name: "empty source",
			mappings: []FileMapping{
				{Source: "", Destination: "/app/config.txt"},
			},
			wantErr: true,
			errMsg:  "source path cannot be empty",
		},
		{
			name: "empty destination",
			mappings: []FileMapping{
				{Source: "~/config.txt", Destination: ""},
			},
			wantErr: true,
			errMsg:  "destination path cannot be empty",
		},
		{
			name: "relative destination",
			mappings: []FileMapping{
				{Source: "~/config.txt", Destination: "config.txt"},
			},
			wantErr: true,
			errMsg:  "must be absolute",
		},
		{
			name: "destination with traversal",
			mappings: []FileMapping{
				{Source: "~/config.txt", Destination: "/app/../etc/passwd"},
			},
			wantErr: true,
			errMsg:  "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMappings(tt.mappings)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMappings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateMappings() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestEncodeDecodeSecretKey(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "simple path",
			path: "/app/config.txt",
		},
		{
			name: "path with special chars",
			path: "/app/my-file_123.txt",
		},
		{
			name: "path with spaces",
			path: "/app/my file.txt",
		},
		{
			name: "deeply nested path",
			path: "/root/.ssh/config/keys/id_rsa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeSecretKey(tt.path)

			// Verify encoding doesn't contain problematic characters
			if strings.Contains(encoded, "/") || strings.Contains(encoded, " ") {
				t.Errorf("EncodeSecretKey() contains invalid characters: %s", encoded)
			}

			// Verify round-trip
			decoded, err := DecodeSecretKey(encoded)
			if err != nil {
				t.Errorf("DecodeSecretKey() error = %v", err)
				return
			}

			if decoded != tt.path {
				t.Errorf("DecodeSecretKey() = %v, want %v", decoded, tt.path)
			}
		})
	}
}

func TestDecodeSecretKeyInvalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "invalid base64",
			key:  "not-valid-base64!@#$",
		},
		{
			name: "empty key",
			key:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeSecretKey(tt.key)
			if err == nil {
				t.Errorf("DecodeSecretKey() expected error for invalid input")
			}
		})
	}
}
