# Kodama

A kubectl plugin for managing Claude Code sessions in Kubernetes.

## Overview

Kodama makes it easy to run Claude Code development sessions in isolated Kubernetes environments. It provides a simple CLI interface for managing session lifecycles without requiring cluster admin privileges.

**Key Features:**
- 🚀 Simple kubectl plugin - no CRDs or controllers required
- 💾 Persistent workspaces across sessions
- 🔄 Bidirectional file synchronization with mutagen
- 🌿 Automatic branch creation
- 🔒 Isolated environments in Kubernetes
- ⚙️  Easy configuration management

## Architecture

Kodama uses a **kubectl plugin approach** (no CRDs/controllers):
- Stores session state locally in `~/.kodama/`
- Directly manages K8s resources via client-go
- No admin privileges required

## Installation

### Prerequisites

- **Go 1.25** or later
- **kubectl** configured with access to a Kubernetes cluster
- **mise** (for development) - optional

### Install from Source

```bash
git clone https://github.com/illumination-k/kodama.git
cd kodama
mise run build
mise run dev-install
```

Or using Go directly:

```bash
git clone https://github.com/illumination-k/kodama.git
cd kodama
go build -o kubectl-kodama ./cmd/kubectl-kodama
cp kubectl-kodama ~/.local/bin/
```

### Verify Installation

```bash
kubectl plugin list | grep kodama
kubectl kodama version
```

## Quick Start

> **Note:** Phase 1 (current) provides the foundation with config management. Session management commands (`start`, `list`, `attach`, etc.) will be implemented in Phase 2.

### Configuration

Kodama stores configuration in `~/.kodama/`:

- `~/.kodama/config.yaml` - Global configuration
- `~/.kodama/sessions/*.yaml` - Session configurations

### Global Configuration Example

Create `~/.kodama/config.yaml`:

```yaml
defaults:
  namespace: default
  image: ghcr.io/illumination-k/kodama-claude:latest
  resources:
    cpu: "1"
    memory: "2Gi"
  storage:
    workspace: "10Gi"
    claudeHome: "1Gi"
  branchPrefix: "kodama/"

git:
  secretName: git-ssh-key  # Optional: K8s secret for git authentication
```

### Session Configuration Example

Sessions are stored in `~/.kodama/sessions/<name>.yaml`:

```yaml
name: my-work
namespace: default
repo: github.com/myorg/myrepo
branch: kodama/my-work-20250115143000
baseBranch: main
autoBranch: true

podName: kodama-my-work
workspacePVC: kodama-workspace-my-work
claudeHomePVC: kodama-claude-home-my-work

commitHash: abc123def456

sync:
  enabled: true
  localPath: /Users/shogo/projects/myrepo
  mutagenSession: kodama-my-work-sync

status: Running
createdAt: "2025-01-15T14:30:00Z"
updatedAt: "2025-01-15T14:35:00Z"

resources:
  cpu: "2"
  memory: "4Gi"
```

## Development

### Setup

```bash
mise install
```

### Build

```bash
mise run build
```

### Test

```bash
mise run test
```

### Lint

```bash
mise run lint
```

### Format Code

```bash
mise run fmt
```

### Test Coverage

```bash
mise run coverage
```

## Project Structure

```
kodama/
├── cmd/kubectl-kodama/     # CLI entry point
├── pkg/
│   ├── config/             # Configuration management
│   ├── kubernetes/         # K8s client wrapper
│   └── commands/           # CLI commands
├── internal/version/       # Version info
├── examples/sessions/      # Example configurations
└── design_docs/            # Design documentation
```

## Roadmap

### ✅ Phase 1: Foundation (Current)
- [x] Go module initialization
- [x] CLI framework (cobra)
- [x] Configuration management
- [x] Kubernetes client setup
- [x] Build infrastructure

### 🚧 Phase 2: Session Management (Next)
- [ ] `start` command - Create new sessions
- [ ] `list` command - List all sessions
- [ ] `get` command - Show session details
- [ ] `delete` command - Remove sessions

### 📅 Phase 3: Advanced Features
- [ ] `stop` / `resume` / `attach` commands
- [ ] Mutagen sync integration
- [ ] End-to-end testing
- [ ] Complete documentation

## Configuration Reference

### Session Status

Sessions can be in one of these states:
- `Pending` - Session created, not yet started
- `Starting` - Resources being provisioned
- `Running` - Session active and ready
- `Stopped` - Pod deleted, PVCs preserved
- `Failed` - Session encountered an error

### Resource Configuration

Default resource limits:
```yaml
resources:
  cpu: "1"      # CPU limit
  memory: "2Gi" # Memory limit
```

### Storage Configuration

Default storage sizes:
```yaml
storage:
  workspace: "10Gi"   # Workspace PVC size
  claudeHome: "1Gi"   # Claude home directory size
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

## Acknowledgments

Inspired by [DevPod](https://github.com/loft-sh/devpod) and [Devspace](https://github.com/devspace-sh/devspace).
