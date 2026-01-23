package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/illumination-k/kodama/pkg/secretfile"
)

// CreateFileSecret creates a Kubernetes secret containing file contents
// The secret is labeled with app=kodama, session=<name>, and managed-by=kodama for easy management
// File paths are base64-encoded to meet K8s secret key naming restrictions
// Original paths are stored in annotations for reconstruction if needed
func (c *Client) CreateFileSecret(ctx context.Context, name, namespace string, files map[string][]byte) error {
	// Convert file paths to base64-encoded secret keys
	secretData := make(map[string][]byte)
	annotations := make(map[string]string)

	for destPath, content := range files {
		secretKey := secretfile.EncodeSecretKey(destPath)
		secretData[secretKey] = content
		// Store original path in annotation for reference (key format: path-<encoded>)
		annotations["path-"+secretKey] = destPath
	}

	// Extract session name from secret name (format: kodama-secret-files-<session-name>)
	sessionName := ""
	if len(name) > len("kodama-secret-files-") {
		sessionName = name[len("kodama-secret-files-"):]
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
			Annotations: annotations,
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	_, err := c.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create file secret: %w", err)
	}

	return nil
}
