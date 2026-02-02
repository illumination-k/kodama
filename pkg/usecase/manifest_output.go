package usecase

import (
	"encoding/json"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// WriteManifestsYAML outputs manifests in YAML multi-document format
// Manifests are separated by "---" as per Kubernetes convention
func WriteManifestsYAML(manifests *ManifestCollection, w io.Writer) error {
	if manifests == nil {
		return fmt.Errorf("manifests collection is nil")
	}

	// Track if we need separators
	needsSeparator := false

	// Write env secret if present
	if manifests.EnvSecret != nil {
		if needsSeparator {
			if _, err := fmt.Fprintln(w, "---"); err != nil {
				return fmt.Errorf("failed to write separator: %w", err)
			}
		}
		if err := writeYAML(manifests.EnvSecret, w); err != nil {
			return fmt.Errorf("failed to write env secret: %w", err)
		}
		needsSeparator = true
	}

	// Write file secret if present
	if manifests.FileSecret != nil {
		if needsSeparator {
			if _, err := fmt.Fprintln(w, "---"); err != nil {
				return fmt.Errorf("failed to write separator: %w", err)
			}
		}
		if err := writeYAML(manifests.FileSecret, w); err != nil {
			return fmt.Errorf("failed to write file secret: %w", err)
		}
		needsSeparator = true
	}

	// Write pod (required)
	if manifests.Pod == nil {
		return fmt.Errorf("pod manifest is required but not present")
	}
	if needsSeparator {
		if _, err := fmt.Fprintln(w, "---"); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}
	}
	if err := writeYAML(manifests.Pod, w); err != nil {
		return fmt.Errorf("failed to write pod: %w", err)
	}

	return nil
}

// WriteManifestsJSON outputs manifests in JSON format as a Kubernetes List
func WriteManifestsJSON(manifests *ManifestCollection, w io.Writer) error {
	if manifests == nil {
		return fmt.Errorf("manifests collection is nil")
	}

	if manifests.Pod == nil {
		return fmt.Errorf("pod manifest is required but not present")
	}

	// Build items list
	items := []interface{}{}

	if manifests.EnvSecret != nil {
		items = append(items, manifests.EnvSecret)
	}

	if manifests.FileSecret != nil {
		items = append(items, manifests.FileSecret)
	}

	items = append(items, manifests.Pod)

	// Create Kubernetes List object
	list := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      items,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(list); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// writeYAML writes a Kubernetes object to the writer in YAML format
func writeYAML(obj interface{}, w io.Writer) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write YAML: %w", err)
	}

	return nil
}

// RedactSecrets replaces secret data values with "<REDACTED>" placeholder
// This is the default behavior for security - use --show-secrets to reveal
func RedactSecrets(manifests *ManifestCollection) *ManifestCollection {
	if manifests == nil {
		return nil
	}

	// Create a deep copy to avoid modifying original
	redacted := &ManifestCollection{
		Pod: manifests.Pod.DeepCopy(),
	}

	if manifests.EnvSecret != nil {
		redacted.EnvSecret = redactSecret(manifests.EnvSecret)
	}

	if manifests.FileSecret != nil {
		redacted.FileSecret = redactSecret(manifests.FileSecret)
	}

	return redacted
}

// redactSecret creates a copy of the secret with data values redacted
func redactSecret(secret *corev1.Secret) *corev1.Secret {
	redacted := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secret.Name,
			Namespace:   secret.Namespace,
			Labels:      secret.Labels,
			Annotations: secret.Annotations,
		},
		Type: secret.Type,
		Data: make(map[string][]byte),
	}

	// Replace all data values with redacted placeholder
	for key := range secret.Data {
		redacted.Data[key] = []byte("<REDACTED>")
	}

	return redacted
}
