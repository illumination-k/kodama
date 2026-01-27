package secretfile

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	// MaxSecretSize is the Kubernetes secret size limit (1MB)
	MaxSecretSize = 1024 * 1024
)

// ValidateFilePath ensures the destination path is absolute and safe
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	if !filepath.IsAbs(path) {
		return fmt.Errorf("destination path must be absolute: %s", path)
	}

	// Check for directory traversal attempts in original path
	if strings.Contains(path, "..") {
		return fmt.Errorf("destination path contains directory traversal: %s", path)
	}

	return nil
}

// ValidateSecretSize checks that the total secret data doesn't exceed Kubernetes limits
func ValidateSecretSize(data map[string][]byte) error {
	totalSize := 0
	for key, value := range data {
		// Account for base64 encoding overhead in secret keys
		keySize := len(base64.URLEncoding.EncodeToString([]byte(key)))
		totalSize += keySize + len(value)
	}

	if totalSize > MaxSecretSize {
		return fmt.Errorf("total secret file size (%d bytes) exceeds Kubernetes limit (%d bytes)", totalSize, MaxSecretSize)
	}

	return nil
}

// ValidateMappings checks for duplicate destinations and validates all paths
func ValidateMappings(mappings []FileMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	destinations := make(map[string]bool)

	for _, mapping := range mappings {
		if mapping.Source == "" {
			return fmt.Errorf("source path cannot be empty")
		}

		if err := ValidateFilePath(mapping.Destination); err != nil {
			return err
		}

		if destinations[mapping.Destination] {
			return fmt.Errorf("duplicate destination path: %s", mapping.Destination)
		}
		destinations[mapping.Destination] = true
	}

	return nil
}

// EncodeSecretKey encodes a file path as a valid Kubernetes secret key
// Uses URL-safe base64 encoding to handle special characters and path separators
func EncodeSecretKey(path string) string {
	return base64.URLEncoding.EncodeToString([]byte(path))
}

// DecodeSecretKey decodes a Kubernetes secret key back to a file path
func DecodeSecretKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("secret key cannot be empty")
	}

	decoded, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("failed to decode secret key: %w", err)
	}

	if len(decoded) == 0 {
		return "", fmt.Errorf("decoded secret key is empty")
	}

	return string(decoded), nil
}
