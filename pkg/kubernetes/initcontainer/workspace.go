package initcontainer

import (
	"github.com/illumination-k/kodama/pkg/gitcmd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// WorkspaceInitializerConfig configures workspace initialization (git clone)
type WorkspaceInitializerConfig struct {
	// Git repository URL (if empty, workspace initializer is not created)
	GitRepo string

	// Git branch to create/checkout
	GitBranch string

	// Git clone options
	CloneDepth   int
	SingleBranch bool
	ExtraArgs    string

	// Git secret for authentication
	GitSecretName string

	// WorkspaceVolumeName is the name of the volume to mount at /workspace
	WorkspaceVolumeName string
}

// NewWorkspaceInitializerConfig creates a new workspace initializer configuration
func NewWorkspaceInitializerConfig(gitRepo, gitBranch string, opts *gitcmd.CloneOptions) *WorkspaceInitializerConfig {
	config := &WorkspaceInitializerConfig{
		GitRepo:             gitRepo,
		GitBranch:           gitBranch,
		WorkspaceVolumeName: "workspace",
	}

	if opts != nil {
		config.CloneDepth = opts.Depth
		config.SingleBranch = opts.SingleBranch
		config.ExtraArgs = opts.ExtraArgs
	}

	return config
}

// WithGitSecret sets the git secret name for authentication
func (w *WorkspaceInitializerConfig) WithGitSecret(secretName string) *WorkspaceInitializerConfig {
	w.GitSecretName = secretName
	return w
}

// WithWorkspaceVolume sets the workspace volume name
func (w *WorkspaceInitializerConfig) WithWorkspaceVolume(volumeName string) *WorkspaceInitializerConfig {
	w.WorkspaceVolumeName = volumeName
	return w
}

// IsEnabled returns true if workspace initialization should be performed
func (w *WorkspaceInitializerConfig) IsEnabled() bool {
	return w.GitRepo != ""
}

// Name returns the init container name
func (w *WorkspaceInitializerConfig) Name() string {
	return "workspace-initializer"
}

// Image returns the container image
func (w *WorkspaceInitializerConfig) Image() string {
	return "ubuntu:24.04"
}

// Command returns the shell command
func (w *WorkspaceInitializerConfig) Command() []string {
	return []string{"/bin/bash", "-c"}
}

// Args returns the git initialization script
func (w *WorkspaceInitializerConfig) Args() []string {
	opts := &gitcmd.CloneOptions{
		Depth:        w.CloneDepth,
		SingleBranch: w.SingleBranch,
		ExtraArgs:    w.ExtraArgs,
	}

	script := gitcmd.BuildGitInitScript(w.GitRepo, w.GitBranch, opts)
	return []string{script}
}

// VolumeMounts returns required volume mounts
func (w *WorkspaceInitializerConfig) VolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      w.WorkspaceVolumeName,
			MountPath: "/workspace",
		},
	}
}

// EnvVars returns environment variables for git authentication
func (w *WorkspaceInitializerConfig) EnvVars() []corev1.EnvVar {
	if w.GitSecretName == "" {
		return []corev1.EnvVar{}
	}

	return []corev1.EnvVar{
		{
			Name: "GH_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: w.GitSecretName,
					},
					Key:      "token",
					Optional: ptr.To(true),
				},
			},
		},
	}
}

// StartMessage returns the initialization start message
func (w *WorkspaceInitializerConfig) StartMessage() string {
	return "Initializing workspace..."
}

// CompletionMessage returns the initialization completion message
func (w *WorkspaceInitializerConfig) CompletionMessage() string {
	return "Workspace initialization complete"
}
