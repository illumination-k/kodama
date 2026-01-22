package env

import (
	"encoding/json"
	"fmt"
	"regexp"
)

const (
	// MaxSecretSize is the Kubernetes secret size limit (1MB)
	MaxSecretSize = 1 * 1024 * 1024 // 1MB in bytes
)

// varNamePattern validates environment variable names
// Must start with letter or underscore, followed by alphanumeric or underscore
var varNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// ValidateVarName checks if the variable name is valid
func ValidateVarName(name string) error {
	if !varNamePattern.MatchString(name) {
		return fmt.Errorf("variable name must match pattern ^[A-Z_][A-Z0-9_]*$")
	}
	return nil
}

// ValidateSecretSize checks if the environment variables will fit in a K8s secret
func ValidateSecretSize(data map[string]string) error {
	// Estimate secret size by JSON marshaling
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to estimate secret size: %w", err)
	}

	size := len(jsonData)
	if size > MaxSecretSize {
		return fmt.Errorf("environment variables exceed Kubernetes secret size limit (1MB): current size is %d bytes", size)
	}

	return nil
}

// IsSystemVar checks if the variable is in the default exclusion list
func IsSystemVar(name string) bool {
	for _, v := range DefaultExcludedVars {
		if v == name {
			return true
		}
	}
	return false
}
