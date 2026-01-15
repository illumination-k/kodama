package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Client wraps the Kubernetes clientset and provides convenience methods
type Client struct {
	clientset *kubernetes.Clientset
	config    *Config
}

// Config holds configuration for the Kubernetes client
type Config struct {
	KubeconfigPath string
	Namespace      string
}

// PodSpec contains specifications for creating a pod
type PodSpec struct {
	Name          string
	Namespace     string
	Image         string
	WorkspacePVC  string
	ClaudeHomePVC string
	CPULimit      string
	MemoryLimit   string
	GitSecretName string
	Command       []string

	// Claude authentication
	ClaudeAuthType   string            // "token", "file"
	ClaudeSecretName string            // K8s secret for token
	ClaudeSecretKey  string            // Key in secret (default: "token")
	ClaudeAuthFile   string            // Path to auth file (for file auth)
	ClaudeEnvVars    map[string]string // Additional env vars for auth

	// Editor configuration
	EditorConfigMapName string // ConfigMap name for editor configs (Helix, Zellij)
}

// PVCSpec contains specifications for creating a PersistentVolumeClaim
type PVCSpec struct {
	Name      string
	Namespace string
	Size      string
}

// JobSpec contains specifications for creating a job
type JobSpec struct {
	Name          string
	Namespace     string
	Image         string
	WorkspacePVC  string
	ClaudeHomePVC string
	Command       []string
}

// PodStatus represents the current state of a pod
type PodStatus struct {
	Phase      corev1.PodPhase
	IP         string
	StartTime  string
	Conditions []corev1.PodCondition
	Ready      bool
}
