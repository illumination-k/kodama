package port

import (
	"context"
	"os/exec"
	"time"

	"github.com/illumination-k/kodama/pkg/kubernetes"
)

// KubernetesClient provides interface for Kubernetes operations
type KubernetesClient interface {
	// Pod operations
	CreatePod(ctx context.Context, spec *kubernetes.PodSpec) error
	GetPod(ctx context.Context, name, namespace string) (*kubernetes.PodStatus, error)
	WaitForPodReady(ctx context.Context, name, namespace string, timeout time.Duration) error
	DeletePod(ctx context.Context, name, namespace string) error
	WaitForPodDeleted(ctx context.Context, name, namespace string, timeout time.Duration) error
	GetPodIP(ctx context.Context, name, namespace string) (string, error)

	// Secret operations
	CreateSecret(ctx context.Context, name, namespace string, data map[string]string) error
	DeleteSecret(ctx context.Context, name, namespace string) error
	SecretExists(ctx context.Context, name, namespace string) (bool, error)
	CreateFileSecret(ctx context.Context, name, namespace string, files map[string][]byte) error

	// Port forwarding
	StartPortForward(ctx context.Context, podName string, localPort, remotePort int) (*exec.Cmd, error)

	// Utility operations
	GetCurrentNamespace() (string, error)
	Ping(ctx context.Context) error
}
