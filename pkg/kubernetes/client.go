package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*Client, error) {
	config, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config: &Config{
			KubeconfigPath: kubeconfigPath,
		},
	}, nil
}

// buildConfig creates a Kubernetes REST config from kubeconfig
func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	if kubeconfigPath == "" {
		kubeconfigPath = getDefaultKubeconfigPath()
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// getDefaultKubeconfigPath returns the default kubeconfig file path
func getDefaultKubeconfigPath() string {
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		return kubeconfigEnv
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".kube", "config")
}

// GetCurrentNamespace returns the current namespace from kubeconfig context
func (c *Client) GetCurrentNamespace() (string, error) {
	kubeconfigPath := c.config.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = getDefaultKubeconfigPath()
	}

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	contextName := config.CurrentContext
	if contextName == "" {
		return "default", nil
	}

	context, exists := config.Contexts[contextName]
	if !exists {
		return "default", nil
	}

	if context.Namespace == "" {
		return "default", nil
	}

	return context.Namespace, nil
}

// Ping verifies connectivity to the Kubernetes cluster
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}
	return nil
}
