package secretfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadFiles loads file contents from the local machine
// Returns a map of destination paths to file contents
// Soft failure: warns but continues if a file is not found
func LoadFiles(mappings []FileMapping) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for _, mapping := range mappings {
		// Expand ~ to home directory
		sourcePath := expandHomeDir(mapping.Source)

		// Read file contents (binary-safe)
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			if os.IsNotExist(err) {
				// Soft failure: warn but continue
				fmt.Printf("⚠️  Warning: Secret file not found: %s\n", sourcePath)
				continue
			}
			return nil, fmt.Errorf("failed to read %s: %w", sourcePath, err)
		}

		result[mapping.Destination] = content
	}

	return result, nil
}

// expandHomeDir expands ~ to the user's home directory
func expandHomeDir(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}
