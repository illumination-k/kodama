package initcontainer

import (
	corev1 "k8s.io/api/core/v1"
)

// TtydInstallerConfig configures ttyd (web terminal) installation
type TtydInstallerConfig struct {
	// Version specifies the ttyd release version (e.g., "1.7.7")
	Version string

	// BinVolumeName is the name of the volume to mount at /kodama/bin
	BinVolumeName string
}

// NewTtydInstallerConfig creates a new ttyd installer configuration
func NewTtydInstallerConfig(version, binVolumeName string) *TtydInstallerConfig {
	if version == "" {
		version = "1.7.7"
	}
	if binVolumeName == "" {
		binVolumeName = "kodama-bin"
	}

	return &TtydInstallerConfig{
		Version:       version,
		BinVolumeName: binVolumeName,
	}
}

// Name returns the init container name
func (t *TtydInstallerConfig) Name() string {
	return "ttyd-installer"
}

// Image returns the container image
func (t *TtydInstallerConfig) Image() string {
	return "ubuntu:24.04"
}

// Command returns the shell command
func (t *TtydInstallerConfig) Command() []string {
	return []string{"/bin/bash", "-c"}
}

// Args returns the installation script
func (t *TtydInstallerConfig) Args() []string {
	downloadURL := "https://github.com/tsl0922/ttyd/releases/download/" + t.Version + "/ttyd.x86_64"
	script := BuildScript(
		t.StartMessage(),
		t.CompletionMessage(),
		"apt-get update -qq && apt-get install -y -qq curl ca-certificates",
		"curl -fsSL "+downloadURL+" -o /tmp/ttyd",
		"chmod +x /tmp/ttyd",
		"mkdir -p /kodama/bin",
		"cp /tmp/ttyd /kodama/bin/ttyd",
	)
	return []string{script}
}

// VolumeMounts returns required volume mounts
func (t *TtydInstallerConfig) VolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      t.BinVolumeName,
			MountPath: "/kodama/bin",
		},
	}
}

// EnvVars returns environment variables (none needed for ttyd installer)
func (t *TtydInstallerConfig) EnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{}
}

// StartMessage returns the installation start message
func (t *TtydInstallerConfig) StartMessage() string {
	return "Installing ttyd..."
}

// CompletionMessage returns the installation completion message
func (t *TtydInstallerConfig) CompletionMessage() string {
	return "ttyd installation complete"
}
