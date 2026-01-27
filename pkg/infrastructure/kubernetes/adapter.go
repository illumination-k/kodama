package kubernetes

import (
	"context"
	"os/exec"
	"time"

	"github.com/illumination-k/kodama/pkg/application/port"
	k8s "github.com/illumination-k/kodama/pkg/kubernetes"
)

// Adapter implements port.KubernetesClient using the existing kubernetes.Client
type Adapter struct {
	client *k8s.Client
}

// NewAdapter creates a new Kubernetes adapter
func NewAdapter(kubeconfigPath string) (port.KubernetesClient, error) {
	client, err := k8s.NewClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client}, nil
}

// Pod operations

// CreatePod creates a pod with the given specification
func (a *Adapter) CreatePod(ctx context.Context, spec *k8s.PodSpec) error {
	return a.client.CreatePod(ctx, spec)
}

// GetPod retrieves the status of a pod
func (a *Adapter) GetPod(ctx context.Context, name, namespace string) (*k8s.PodStatus, error) {
	return a.client.GetPod(ctx, name, namespace)
}

// WaitForPodReady waits for a pod to become ready
func (a *Adapter) WaitForPodReady(ctx context.Context, name, namespace string, timeout time.Duration) error {
	return a.client.WaitForPodReady(ctx, name, namespace, timeout)
}

// DeletePod deletes a pod
func (a *Adapter) DeletePod(ctx context.Context, name, namespace string) error {
	return a.client.DeletePod(ctx, name, namespace)
}

// WaitForPodDeleted waits for a pod to be deleted
func (a *Adapter) WaitForPodDeleted(ctx context.Context, name, namespace string, timeout time.Duration) error {
	return a.client.WaitForPodDeleted(ctx, name, namespace, timeout)
}

// GetPodIP retrieves the IP address of a pod
func (a *Adapter) GetPodIP(ctx context.Context, name, namespace string) (string, error) {
	return a.client.GetPodIP(ctx, name, namespace)
}

// Secret operations

// CreateSecret creates a secret with the given data
func (a *Adapter) CreateSecret(ctx context.Context, name, namespace string, data map[string]string) error {
	return a.client.CreateSecret(ctx, name, namespace, data)
}

// DeleteSecret deletes a secret
func (a *Adapter) DeleteSecret(ctx context.Context, name, namespace string) error {
	return a.client.DeleteSecret(ctx, name, namespace)
}

// SecretExists checks if a secret exists
func (a *Adapter) SecretExists(ctx context.Context, name, namespace string) (bool, error) {
	return a.client.SecretExists(ctx, name, namespace)
}

// CreateFileSecret creates a secret from files
func (a *Adapter) CreateFileSecret(ctx context.Context, name, namespace string, files map[string][]byte) error {
	return a.client.CreateFileSecret(ctx, name, namespace, files)
}

// Port forwarding

// StartPortForward starts port forwarding to a pod
func (a *Adapter) StartPortForward(ctx context.Context, podName string, localPort, remotePort int) (*exec.Cmd, error) {
	return a.client.StartPortForward(ctx, podName, localPort, remotePort)
}

// Utility operations

// GetCurrentNamespace returns the current namespace from kubeconfig
func (a *Adapter) GetCurrentNamespace() (string, error) {
	return a.client.GetCurrentNamespace()
}

// Ping verifies connectivity to the Kubernetes cluster
func (a *Adapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx)
}
