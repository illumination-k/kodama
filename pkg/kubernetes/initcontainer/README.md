# Init Container Package

This package provides a config-based architecture for building Kubernetes init containers in Kodama.

## Overview

The init container system uses an interface-based design to make it easy to add, configure, and test different installers. Each installer is defined by implementing the `InstallerConfig` interface.

## Architecture

```
InstallerConfig (interface)
    ├── ClaudeInstallerConfig  - Claude Code CLI installation
    ├── TtydInstallerConfig    - ttyd web terminal installation
    └── WorkspaceInitializerConfig - Git workspace initialization

Builder - Converts configs to corev1.Container
```

## Core Components

### InstallerConfig Interface

All installers must implement this interface:

```go
type InstallerConfig interface {
    Name() string                      // Container name
    Image() string                     // Container image
    Command() []string                 // Shell command (usually ["/bin/bash", "-c"])
    Args() []string                    // Installation script
    VolumeMounts() []corev1.VolumeMount // Required volume mounts
    EnvVars() []corev1.EnvVar          // Environment variables
    StartMessage() string              // Message shown at start
    CompletionMessage() string         // Message shown on completion
}
```

### Builder

The `Builder` takes installer configs and produces Kubernetes init containers:

**Single installer:**

```go
builder := initcontainer.NewBuilder()
container := builder.Build(config)
```

**Combined installers (recommended for efficiency):**

```go
builder := initcontainer.NewBuilder()
configs := []initcontainer.InstallerConfig{
    initcontainer.NewClaudeInstallerConfig("latest", "kodama-bin"),
    initcontainer.NewTtydInstallerConfig("1.7.7", "kodama-bin"),
}
container := builder.BuildCombined("tools-installer", configs...)
```

`BuildCombined` merges multiple installers into a single init container, which is more efficient than creating separate containers. It automatically:

- Combines scripts sequentially with proper logging
- Deduplicates volume mounts and environment variables
- Uses the first config's image and command

### BuildScript Utility

Helper function to generate bash scripts with consistent logging:

```go
script := BuildScript(
    "Installing tool...",
    "Installation complete",
    "apt-get update",
    "apt-get install -y mytool",
)
```

## Built-in Installers

### ClaudeInstallerConfig

Installs Claude Code CLI with configurable version:

```go
config := initcontainer.NewClaudeInstallerConfig("latest", "kodama-bin")
```

### TtydInstallerConfig

Installs ttyd web terminal with configurable version:

```go
config := initcontainer.NewTtydInstallerConfig("1.7.7", "kodama-bin")
```

### WorkspaceInitializerConfig

Initializes git workspace with clone and branch setup:

```go
opts := &gitcmd.CloneOptions{
    Depth:        1,
    SingleBranch: true,
}
config := initcontainer.NewWorkspaceInitializerConfig(
    "https://github.com/example/repo.git",
    "feature-branch",
    opts,
)
```

## Adding a New Installer

1. Create a new file in this package (e.g., `myinstaller.go`)
2. Implement the `InstallerConfig` interface
3. Add a constructor function (e.g., `NewMyInstallerConfig()`)
4. Update `buildInitContainers()` in `pkg/kubernetes/pod.go` to include your installer

Example:

```go
package initcontainer

import corev1 "k8s.io/api/core/v1"

type MyInstallerConfig struct {
    Version       string
    BinVolumeName string
}

func NewMyInstallerConfig(version, binVolumeName string) *MyInstallerConfig {
    if version == "" {
        version = "latest"
    }
    if binVolumeName == "" {
        binVolumeName = "kodama-bin"
    }
    return &MyInstallerConfig{
        Version:       version,
        BinVolumeName: binVolumeName,
    }
}

func (m *MyInstallerConfig) Name() string {
    return "my-installer"
}

func (m *MyInstallerConfig) Image() string {
    return "ubuntu:24.04"
}

func (m *MyInstallerConfig) Command() []string {
    return []string{"/bin/bash", "-c"}
}

func (m *MyInstallerConfig) Args() []string {
    script := BuildScript(
        m.StartMessage(),
        m.CompletionMessage(),
        "apt-get update -qq",
        "apt-get install -y -qq mytool",
        "cp /usr/bin/mytool /kodama/bin/",
    )
    return []string{script}
}

func (m *MyInstallerConfig) VolumeMounts() []corev1.VolumeMount {
    return []corev1.VolumeMount{
        {
            Name:      m.BinVolumeName,
            MountPath: "/kodama/bin",
        },
    }
}

func (m *MyInstallerConfig) EnvVars() []corev1.EnvVar {
    return []corev1.EnvVar{}
}

func (m *MyInstallerConfig) StartMessage() string {
    return "Installing MyTool..."
}

func (m *MyInstallerConfig) CompletionMessage() string {
    return "MyTool installation complete"
}
```

## Benefits of This Design

1. **Efficiency**: `BuildCombined` reduces init container count and startup time
2. **Scalability**: Easy to add new installers without modifying existing code
3. **Testability**: Each installer can be unit tested independently
4. **Configurability**: Version and other options configurable per session
5. **Consistency**: Standardized logging and error handling
6. **Maintainability**: Clear separation of concerns
7. **Extensibility**: Simple interface makes extension straightforward
