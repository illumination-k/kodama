package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSecret creates a Kubernetes secret with the given data
// The secret is labeled with app=kodama and session=<name> for easy management
// If dryRun is true, returns the manifest without creating it
func (c *Client) CreateSecret(ctx context.Context, name, namespace string, data map[string]string, dryRun bool) (*corev1.Secret, error) {
	// Convert string map to byte map (K8s expects []byte values)
	secretData := make(map[string][]byte)
	for key, value := range data {
		secretData[key] = []byte(value)
	}

	// Extract session name from secret name (format: kodama-env-<session-name>)
	sessionName := ""
	if len(name) > len("kodama-env-") {
		sessionName = name[len("kodama-env-"):]
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "kodama",
				"session":    sessionName,
				"managed-by": "kodama",
			},
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	// If dry-run, return the manifest without creating
	if dryRun {
		return secret, nil
	}

	_, err := c.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	return secret, nil
}

// DeleteSecret deletes a Kubernetes secret
// Ignores "not found" errors (secret already deleted)
func (c *Client) DeleteSecret(ctx context.Context, name, namespace string) error {
	gracePeriodSeconds := int64(30)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	err := c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		// Ignore "not found" errors - secret already deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// SecretExists checks if a secret exists in the given namespace
func (c *Client) SecretExists(ctx context.Context, name, namespace string) (bool, error) {
	_, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if secret exists: %w", err)
	}

	return true, nil
}
