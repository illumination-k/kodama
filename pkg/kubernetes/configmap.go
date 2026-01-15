package kubernetes

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed defaults/helix-config.toml
var defaultHelixConfig string

//go:embed defaults/helix-languages.toml
var defaultHelixLanguages string

//go:embed defaults/zellij-config.kdl
var defaultZellijConfig string

const (
	configMapKeyHelixConfig    = "helix-config.toml"
	configMapKeyHelixLanguages = "helix-languages.toml"
	configMapKeyZellijConfig   = "zellij-config.kdl"
)

// CreateEditorConfigMap creates a ConfigMap with editor configurations
func (c *Client) CreateEditorConfigMap(ctx context.Context, namespace, name, configPath string) error {
	// Read config files (custom or defaults)
	configs := readConfigFiles(configPath)

	// Extract session name from ConfigMap name (format: kodama-editor-config-{session-name})
	sessionName := name
	if len(name) > 23 { // len("kodama-editor-config-")
		sessionName = name[22:]
	}

	// Create ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "kodama",
				"session": sessionName,
			},
		},
		Data: configs,
	}

	// Delete existing ConfigMap if it exists
	existingCM, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil && existingCM != nil {
		fmt.Printf("⚠️  Warning: ConfigMap %s already exists, replacing it\n", name)
		if deleteErr := c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); deleteErr != nil {
			return fmt.Errorf("failed to delete existing ConfigMap: %w", deleteErr)
		}
	}

	// Create new ConfigMap
	_, err = c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	return nil
}

// DeleteEditorConfigMap removes the editor ConfigMap
func (c *Client) DeleteEditorConfigMap(ctx context.Context, namespace, name string) error {
	err := c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// ConfigMap doesn't exist, not an error
			return nil
		}
		return fmt.Errorf("failed to delete ConfigMap: %w", err)
	}
	return nil
}

// readConfigFiles reads config files from .kodama/configs/ or uses defaults
func readConfigFiles(configPath string) map[string]string {
	configs := make(map[string]string)

	if configPath == "" {
		// No custom config path, use all defaults
		return getDefaultConfigs()
	}

	// Try to read custom configs
	configDir := filepath.Join(configPath, ".kodama", "configs")

	// Read Helix config
	helixConfigPath := filepath.Join(configDir, "helix", "config.toml")
	// #nosec G304 -- configPath is controlled input from CLI flags, paths are constructed safely with filepath.Join
	if content, err := os.ReadFile(helixConfigPath); err == nil {
		configs[configMapKeyHelixConfig] = string(content)
	} else {
		// Use default if file doesn't exist
		configs[configMapKeyHelixConfig] = defaultHelixConfig
	}

	// Read Helix languages config
	helixLanguagesPath := filepath.Join(configDir, "helix", "languages.toml")
	// #nosec G304 -- configPath is controlled input from CLI flags, paths are constructed safely with filepath.Join
	if content, err := os.ReadFile(helixLanguagesPath); err == nil {
		configs[configMapKeyHelixLanguages] = string(content)
	} else {
		// Use default if file doesn't exist
		configs[configMapKeyHelixLanguages] = defaultHelixLanguages
	}

	// Read Zellij config
	zellijConfigPath := filepath.Join(configDir, "zellij", "config.kdl")
	// #nosec G304 -- configPath is controlled input from CLI flags, paths are constructed safely with filepath.Join
	if content, err := os.ReadFile(zellijConfigPath); err == nil {
		configs[configMapKeyZellijConfig] = string(content)
	} else {
		// Use default if file doesn't exist
		configs[configMapKeyZellijConfig] = defaultZellijConfig
	}

	return configs
}

// getDefaultConfigs returns embedded default configurations
func getDefaultConfigs() map[string]string {
	return map[string]string{
		configMapKeyHelixConfig:    defaultHelixConfig,
		configMapKeyHelixLanguages: defaultHelixLanguages,
		configMapKeyZellijConfig:   defaultZellijConfig,
	}
}
