package exclude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldExclude_SimplePattern(t *testing.T) {
	m := &Manager{
		basePath:       "/tmp/test",
		configPatterns: []string{"*.log"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/test/test.log", true},
		{"/tmp/test/test.txt", false},
		{"/tmp/test/subdir/app.log", true},
		{"/tmp/test/subdir/app.txt", false},
	}

	for _, tt := range tests {
		got := m.ShouldExclude(tt.path)
		if got != tt.want {
			t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestShouldExclude_DirectoryPattern(t *testing.T) {
	m := &Manager{
		basePath:       "/tmp/test",
		configPatterns: []string{"node_modules"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/test/node_modules/pkg/file.js", true},
		{"/tmp/test/src/node_modules/file.js", true},
		{"/tmp/test/src/file.js", false},
	}

	for _, tt := range tests {
		got := m.ShouldExclude(tt.path)
		if got != tt.want {
			t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestShouldExclude_DirectorySlashPattern(t *testing.T) {
	m := &Manager{
		basePath:       "/tmp/test",
		configPatterns: []string{".vscode/"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/test/.vscode", true},
		{"/tmp/test/.vscode/settings.json", true},
		{"/tmp/test/src/.vscode/tasks.json", true},
		{"/tmp/test/src/file.js", false},
	}

	for _, tt := range tests {
		got := m.ShouldExclude(tt.path)
		if got != tt.want {
			t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestShouldExclude_MultiplePatterns(t *testing.T) {
	m := &Manager{
		basePath:       "/tmp/test",
		configPatterns: []string{"*.log", "*.tmp", "node_modules"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/test/app.log", true},
		{"/tmp/test/data.tmp", true},
		{"/tmp/test/node_modules/pkg.js", true},
		{"/tmp/test/app.js", false},
	}

	for _, tt := range tests {
		got := m.ShouldExclude(tt.path)
		if got != tt.want {
			t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestNewManager_WithoutGitignore(t *testing.T) {
	// Create temp dir without .gitignore
	tmpDir, err := os.MkdirTemp("", "test-exclude-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg := Config{
		BasePath:     tmpDir,
		Patterns:     []string{"*.log"},
		UseGitignore: true,
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.gitignoreMatcher != nil {
		t.Error("Expected gitignoreMatcher to be nil when .gitignore doesn't exist")
	}
}

func TestNewManager_WithGitignore(t *testing.T) {
	// Create temp dir with .gitignore
	tmpDir, err := os.MkdirTemp("", "test-exclude-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	gitignoreContent := "*.log\nnode_modules/\n"
	err = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}

	cfg := Config{
		BasePath:     tmpDir,
		Patterns:     []string{},
		UseGitignore: true,
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.gitignoreMatcher == nil {
		t.Error("Expected gitignoreMatcher to be non-nil when .gitignore exists")
	}

	// Test that gitignore patterns work
	logFile := filepath.Join(tmpDir, "test.log")
	if !m.ShouldExclude(logFile) {
		t.Error("Expected .log file to be excluded by .gitignore")
	}

	jsFile := filepath.Join(tmpDir, "test.js")
	if m.ShouldExclude(jsFile) {
		t.Error("Expected .js file to NOT be excluded")
	}
}

func TestNewManager_GitignoreDisabled(t *testing.T) {
	// Create temp dir with .gitignore
	tmpDir, err := os.MkdirTemp("", "test-exclude-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	gitignoreContent := "*.log\n"
	err = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}

	cfg := Config{
		BasePath:     tmpDir,
		Patterns:     []string{},
		UseGitignore: false, // Disabled
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.gitignoreMatcher != nil {
		t.Error("Expected gitignoreMatcher to be nil when UseGitignore is false")
	}

	// Test that gitignore patterns are NOT applied
	logFile := filepath.Join(tmpDir, "test.log")
	if m.ShouldExclude(logFile) {
		t.Error("Expected .log file to NOT be excluded when gitignore is disabled")
	}
}

func TestGetTarExcludeArgs(t *testing.T) {
	m := &Manager{
		basePath:       "/tmp/test",
		configPatterns: []string{"*.log", "node_modules"},
	}

	args := m.GetTarExcludeArgs()

	// Should contain config patterns
	expected := []string{"--exclude=*.log", "--exclude=node_modules", "--exclude=.git"}
	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(args), args)
	}

	for i, exp := range expected {
		if args[i] != exp {
			t.Errorf("args[%d] = %q, want %q", i, args[i], exp)
		}
	}
}

func TestGetTarExcludeArgs_WithGitPattern(t *testing.T) {
	m := &Manager{
		basePath:       "/tmp/test",
		configPatterns: []string{"*.log", ".git"},
	}

	args := m.GetTarExcludeArgs()

	// Should not duplicate .git
	gitCount := 0
	for _, arg := range args {
		if arg == "--exclude=.git" {
			gitCount++
		}
	}

	if gitCount != 1 {
		t.Errorf("Expected exactly 1 .git exclude, got %d", gitCount)
	}
}

func TestMatchPattern_Wildcards(t *testing.T) {
	m := &Manager{basePath: "/test"}

	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.log", "app.log", true},
		{"*.log", "app.txt", false},
		{"test*", "testing", true},
		{"test*", "app", false},
		{"**/*.js", "src/app.js", true},
		{"build/", "build/output.txt", true},
		{"build/", "src/build.txt", false},
	}

	for _, tt := range tests {
		got := m.matchPattern(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}
