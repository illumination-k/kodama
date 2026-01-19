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
	Name            string
	Namespace       string
	Image           string
	WorkspacePVC    string
	ClaudeHomePVC   string
	CPULimit        string
	MemoryLimit     string
	CustomResources map[string]string // e.g., "nvidia.com/gpu": "1"
	GitSecretName   string
	Command         []string

	// Claude authentication
	ClaudeAuthType   string            // "token", "file"
	ClaudeSecretName string            // K8s secret for token
	ClaudeSecretKey  string            // Key in secret (default: "token")
	ClaudeAuthFile   string            // Path to auth file (for file auth)
	ClaudeEnvVars    map[string]string // Additional env vars for auth

	// Git repository configuration for workspace-initializer init container
	GitRepo         string // Git repository URL (empty if no repo)
	GitBranch       string // Feature branch name to create
	GitCloneDepth   int    // Clone depth (0 for full clone)
	GitSingleBranch bool   // Whether to clone single branch only
	GitCloneArgs    string // Additional git clone arguments

	// Ttyd (Web-based terminal) configuration
	TtydEnabled  bool
	TtydPort     int
	TtydOptions  string
	TtydWritable bool

	// DiffViewer sidecar configuration
	DiffViewer *DiffViewerSpec // DiffViewer sidecar settings (nil = disabled)
}

// DiffViewerSpec contains specifications for the difit diff viewer sidecar
type DiffViewerSpec struct {
	Enabled bool   // Whether to enable the diff viewer sidecar
	Image   string // Container image for difit (default: node:21-slim)
	Port    int32  // Port for difit web server (default: 4966)
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
