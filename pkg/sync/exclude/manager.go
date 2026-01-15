package exclude

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Manager handles exclude pattern matching for file sync
type Manager struct {
	gitignoreMatcher *ignore.GitIgnore
	basePath         string
	configPatterns   []string
}

// Config holds configuration for the exclude manager
type Config struct {
	// BasePath is the root directory for sync (for .gitignore location)
	BasePath string

	// Patterns are explicit exclude patterns (gitignore syntax)
	Patterns []string

	// UseGitignore enables automatic .gitignore loading
	UseGitignore bool
}

// NewManager creates a new exclude pattern manager
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		basePath:       cfg.BasePath,
		configPatterns: cfg.Patterns,
	}

	// Load .gitignore if enabled
	if cfg.UseGitignore {
		gitignorePath := filepath.Join(cfg.BasePath, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			// .gitignore exists, try to compile it
			matcher, err := ignore.CompileIgnoreFile(gitignorePath)
			if err != nil {
				// .gitignore is malformed - continue without it
				m.gitignoreMatcher = nil
			} else {
				m.gitignoreMatcher = matcher
			}
		}
		// If .gitignore doesn't exist, that's fine - just continue without it
	}

	return m, nil
}

// ShouldExclude returns true if the path should be excluded from sync
// absPath should be the absolute file path
func (m *Manager) ShouldExclude(absPath string) bool {
	// Get path relative to base
	relPath, err := filepath.Rel(m.basePath, absPath)
	if err != nil {
		// If we can't get relative path, don't exclude
		return false
	}

	// Check config patterns first (these take precedence)
	if m.matchesConfigPatterns(relPath) {
		return true
	}

	// Check gitignore patterns
	if m.gitignoreMatcher != nil && m.gitignoreMatcher.MatchesPath(relPath) {
		return true
	}

	return false
}

// ShouldExcludeDir returns true if the directory should be excluded
// This is optimized for directory traversal (uses filepath.SkipDir)
func (m *Manager) ShouldExcludeDir(absPath string) bool {
	return m.ShouldExclude(absPath)
}

// matchesConfigPatterns checks if path matches any config pattern
func (m *Manager) matchesConfigPatterns(relPath string) bool {
	for _, pattern := range m.configPatterns {
		if m.matchPattern(pattern, relPath) {
			return true
		}
	}
	return false
}

// matchPattern matches a single gitignore-style pattern
func (m *Manager) matchPattern(pattern, path string) bool {
	// Handle directory-only patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		// Only match directories - check if path contains this as dir
		if strings.Contains(path, pattern+"/") ||
			strings.HasPrefix(path, pattern+"/") ||
			path == pattern {
			return true
		}
	}

	// Handle ** wildcards for any directory depth
	if strings.Contains(pattern, "**") {
		pattern = strings.ReplaceAll(pattern, "**", "*")
	}

	// Use filepath.Match for glob patterns
	matched, err := filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Also check if pattern matches any path component
	// (e.g., "node_modules" should match "foo/node_modules/bar")
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		matched, err := filepath.Match(pattern, part)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// GetTarExcludeArgs returns --exclude arguments for tar command
func (m *Manager) GetTarExcludeArgs() []string {
	args := []string{}

	// Add config patterns
	for _, pattern := range m.configPatterns {
		args = append(args, "--exclude="+pattern)
	}

	// For gitignore patterns, we have a limitation:
	// The go-gitignore library doesn't expose raw patterns,
	// and walking the tree ourselves would be slow.
	// For now, we always exclude .git as a safety measure
	// if it's not already in config patterns.
	if !m.hasPattern(".git") && !m.hasPattern(".git/") {
		args = append(args, "--exclude=.git")
	}

	return args
}

// hasPattern checks if a pattern already exists in config
func (m *Manager) hasPattern(pattern string) bool {
	for _, p := range m.configPatterns {
		if p == pattern {
			return true
		}
	}
	return false
}
