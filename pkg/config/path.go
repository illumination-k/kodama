package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath resolves a path, handling ~ expansion and converting to absolute path
func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	} else if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = homeDir
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}

// CustomDirSync represents a custom directory sync configuration
type CustomDirSync struct {
	Source       string   `yaml:"source"`
	Destination  string   `yaml:"destination"`
	Exclude      []string `yaml:"exclude,omitempty"`
	UseGitignore *bool    `yaml:"useGitignore,omitempty"`
	Recursive    bool     `yaml:"recursive,omitempty"`
}

// Validate checks if the custom directory sync configuration is valid
func (c *CustomDirSync) Validate() error {
	if c.Source == "" {
		return fmt.Errorf("source path cannot be empty")
	}

	if c.Destination == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	// Destination must be an absolute path
	if !filepath.IsAbs(c.Destination) {
		return fmt.Errorf("destination must be an absolute path, got: %s", c.Destination)
	}

	// Resolve source path
	resolvedSource, err := ResolvePath(c.Source)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}

	// Check if source exists
	info, err := os.Stat(resolvedSource)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source path does not exist: %s", resolvedSource)
		}
		return fmt.Errorf("failed to access source path: %w", err)
	}

	// If recursive is set, source must be a directory
	if c.Recursive && !info.IsDir() {
		return fmt.Errorf("recursive source must be a directory, got file: %s", resolvedSource)
	}

	return nil
}

// ResolveSource resolves the source path with ~ expansion
func (c *CustomDirSync) ResolveSource() (string, error) {
	return ResolvePath(c.Source)
}
