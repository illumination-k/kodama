package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotenvFiles(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string // filename -> content
		fileOrder   []string          // order to load files
		want        map[string]string
		expectError bool
	}{
		{
			name:      "empty file list",
			files:     map[string]string{},
			fileOrder: []string{},
			want:      map[string]string{},
		},
		{
			name: "single valid file",
			files: map[string]string{
				"test.env": "VAR1=value1\nVAR2=value2\n",
			},
			fileOrder: []string{"test.env"},
			want: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		{
			name: "multiple files with override (last wins)",
			files: map[string]string{
				"first.env":  "VAR1=first\nVAR2=from_first\n",
				"second.env": "VAR1=second\nVAR3=from_second\n",
			},
			fileOrder: []string{"first.env", "second.env"},
			want: map[string]string{
				"VAR1": "second", // overridden by second.env
				"VAR2": "from_first",
				"VAR3": "from_second",
			},
		},
		{
			name: "quoted values",
			files: map[string]string{
				"quoted.env": "VAR1=\"quoted value\"\nVAR2='single quoted'\n",
			},
			fileOrder: []string{"quoted.env"},
			want: map[string]string{
				"VAR1": "quoted value",
				"VAR2": "single quoted",
			},
		},
		{
			name: "comments and empty lines",
			files: map[string]string{
				"comments.env": "# Comment\nVAR1=value1\n\nVAR2=value2\n",
			},
			fileOrder: []string{"comments.env"},
			want: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		{
			name: "malformed dotenv",
			files: map[string]string{
				"bad.env": "INVALID LINE WITHOUT EQUALS\n",
			},
			fileOrder:   []string{"bad.env"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "kodama-env-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			var filePaths []string
			for name, content := range tt.files {
				path := filepath.Join(tmpDir, name)
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("failed to create test file %s: %v", name, err)
				}
			}

			// Build file paths in correct order
			for _, name := range tt.fileOrder {
				filePaths = append(filePaths, filepath.Join(tmpDir, name))
			}

			// Test LoadDotenvFiles
			got, err := LoadDotenvFiles(filePaths)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare results
			if len(got) != len(tt.want) {
				t.Errorf("got %d variables, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				gotValue, ok := got[key]
				if !ok {
					t.Errorf("missing variable %s", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("variable %s: got %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestLoadDotenvFiles_MissingFile(t *testing.T) {
	// Missing files should warn but continue
	result, err := LoadDotenvFiles([]string{"/nonexistent/file.env"})
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d variables", len(result))
	}
}

func TestApplyExclusions(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		exclude []string
		want    map[string]string
	}{
		{
			name: "no exclusions",
			vars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			exclude: []string{},
			want: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		{
			name: "exclude some variables",
			vars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
			exclude: []string{"VAR2"},
			want: map[string]string{
				"VAR1": "value1",
				"VAR3": "value3",
			},
		},
		{
			name: "exclude all variables",
			vars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			exclude: []string{"VAR1", "VAR2"},
			want:    map[string]string{},
		},
		{
			name: "exclude non-existent variable",
			vars: map[string]string{
				"VAR1": "value1",
			},
			exclude: []string{"VAR2"},
			want: map[string]string{
				"VAR1": "value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyExclusions(tt.vars, tt.exclude)

			if len(got) != len(tt.want) {
				t.Errorf("got %d variables, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				gotValue, ok := got[key]
				if !ok {
					t.Errorf("missing variable %s", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("variable %s: got %q, want %q", key, gotValue, wantValue)
				}
			}

			// Ensure excluded variables are not present
			for _, excludedVar := range tt.exclude {
				if _, exists := got[excludedVar]; exists {
					t.Errorf("excluded variable %s is still present", excludedVar)
				}
			}
		})
	}
}

func TestValidateVarName(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		wantErr bool
	}{
		{"valid uppercase", "VAR_NAME", false},
		{"valid with numbers", "VAR_NAME_123", false},
		{"valid underscore prefix", "_VAR_NAME", false},
		{"valid single char", "X", false},
		{"invalid lowercase", "var_name", true},
		{"invalid starts with number", "123VAR", true},
		{"invalid with dash", "VAR-NAME", true},
		{"invalid with dot", "VAR.NAME", true},
		{"invalid with space", "VAR NAME", true},
		{"invalid empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVarName(tt.varName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVarName(%q) error = %v, wantErr %v", tt.varName, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSecretSize(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]string
		wantErr bool
	}{
		{
			name: "small secret",
			data: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			wantErr: false,
		},
		{
			name:    "empty secret",
			data:    map[string]string{},
			wantErr: false,
		},
		{
			name: "large but acceptable secret",
			data: func() map[string]string {
				data := make(map[string]string)
				// Create ~500KB of data (well under 1MB limit)
				for i := 0; i < 1000; i++ {
					data[string(rune('A'+i%26))+string(rune('0'+i%10))] = string(make([]byte, 500))
				}
				return data
			}(),
			wantErr: false,
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

func TestIsSystemVar(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"PATH", true},
		{"HOME", true},
		{"KUBERNETES_SERVICE_HOST", true},
		{"MY_CUSTOM_VAR", false},
		{"DATABASE_URL", false},
		{"GITHUB_TOKEN", false},           // Not a system var - allowed in .env
		{"CLAUDE_CODE_AUTH_TOKEN", false}, // Not a system var - allowed in .env
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSystemVar(tt.name); got != tt.want {
				t.Errorf("IsSystemVar(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
