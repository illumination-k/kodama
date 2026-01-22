package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotenvFiles loads and merges multiple dotenv files with last-wins precedence
func LoadDotenvFiles(files []string) (map[string]string, error) {
	if len(files) == 0 {
		return make(map[string]string), nil
	}

	result := make(map[string]string)

	for _, file := range files {
		// Expand ~ to home directory
		if file[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to expand home directory: %w", err)
			}
			file = filepath.Join(home, file[1:])
		}

		// Check if file exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// Warn but continue - user might use conditional files
			fmt.Printf("⚠️  Warning: Dotenv file not found: %s (skipping)\n", file)
			continue
		}

		// Load the dotenv file
		env, err := godotenv.Read(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse dotenv file %s: %w", file, err)
		}

		// Merge with last-wins precedence
		for key, value := range env {
			result[key] = value
		}
	}

	return result, nil
}

// ApplyExclusions filters out excluded variables from the env map
func ApplyExclusions(vars map[string]string, exclude []string) map[string]string {
	if len(exclude) == 0 {
		return vars
	}

	// Create exclusion set for O(1) lookups
	excludeSet := make(map[string]bool)
	for _, v := range exclude {
		excludeSet[v] = true
	}

	// Filter variables
	result := make(map[string]string)
	for key, value := range vars {
		if !excludeSet[key] {
			result[key] = value
		}
	}

	return result
}
