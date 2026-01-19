# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Kodama is a kubectl plugin written in Go that manages Claude Code development sessions in Kubernetes. It uses a **direct kubectl plugin approach** (no CRDs or controllers) - storing session state locally and managing K8s resources via client-go.

## Build and Development Commands

### Build

```bash
mise run build              # Builds to bin/kubectl-kodama
go build -o bin/kubectl-kodama ./cmd/kubectl-kodama  # Alternative without mise
```

### Testing

```bash
mise run test               # Run all tests
mise run coverage           # Run tests with coverage report
go test ./...              # Alternative without mise
go test -cover ./...       # Alternative coverage without mise
```

### Linting and Formatting

```bash
mise run fmt               # Format with dprint + golangci-lint
mise run lint              # Run all linters (Go + GitHub Actions)
mise run lint:go           # Go linting only
mise run lint:github       # GitHub Actions workflow linting
mise run fix               # Auto-fix issues (fmt + golangci-lint --fix)
```

### Installation

```bash
mise run dev-install       # Install to ~/.local/bin/
mise run install           # Install to /usr/local/bin/
```

### Clean

```bash
mise run clean             # Remove build artifacts
```

## Architecture Overview

Kodama follows a clean architecture with clear separation of concerns:

### Layer Structure

```
CLI Commands (cobra)
    ↓
Usecase Layer (orchestration)
    ↓
┌────────────────────────────────────────────────┐
│  Config    - Session/global state management   │
│  Kubernetes - Pod lifecycle & K8s operations   │
│  Gitcmd    - Git script generation             │
│  Sync      - File synchronization              │
│  Agent     - Coding agent execution            │
└────────────────────────────────────────────────┘
```

### Key Packages

#### `pkg/commands/`

Cobra CLI commands that translate user input to usecase options. Each command (start, attach, list, delete, dev) handles flag parsing and delegates to usecases.

#### `pkg/usecase/`

Orchestration layer coordinating all components:

- **StartSession**: Creates pod with init containers (Claude installer, ttyd, git clone), performs file sync, starts agent tasks
- **AttachSession**: Routes to ttyd (web) or TTY (exec) mode for interactive access
- Uses deferred cleanup to remove resources on failure

#### `pkg/config/`

Multi-tier configuration system:

- **Store**: Manages YAML session configs in `~/.kodama/sessions/`
- **Priority**: CLI flags > template config (`.kodama.yaml`) > global config > defaults
- Session state tracks pod, PVCs, sync status, agent history

#### `pkg/kubernetes/`

Kubernetes abstraction layer:

- Pod creation with multi-init-container strategy
- Status monitoring via pod watch API
- Three auth injection methods: token secret, file-based, environment variables
- Port forwarding for ttyd web terminal
- Command execution wrapper (kubectl exec)

#### `pkg/kubernetes/initcontainer/`

Config-based init container builder system:

- **InstallerConfig interface**: Common interface for all init container installers
- **Builder**: Converts installer configs to Kubernetes init containers
- **ClaudeInstaller**: Claude Code CLI installation with configurable version
- **TtydInstaller**: ttyd web terminal installation with configurable version
- **WorkspaceInitializer**: Git clone with branch setup and auth injection
- **BuildScript utility**: Generates bash scripts with consistent logging
- Pluggable design allows easy addition of new installers
- Each installer encapsulates: image, commands, volume mounts, env vars, and logging messages

#### `pkg/gitcmd/`

Git initialization script builder:

- Generates bash scripts for git clone operations
- Injects GitHub tokens into HTTPS URLs
- Auto-creates feature branches when on protected branches (main/master/trunk)
- Validates clone arguments to prevent injection attacks

#### `pkg/sync/`

Pluggable file synchronization:

- **Initial sync**: Tar-based bulk transfer (efficient)
- **Continuous sync**: fsnotify + kubectl cp with debouncing
- **Custom directory sync**: Additional directories like dotfiles
- **Exclude manager**: Respects `.gitignore` and `.kodamaignore` patterns
- Interface-based design allows future mutagen integration

#### `pkg/agent/`

Coding agent execution system:

- Interface-based design (`CodingAgentExecutor`)
- Auth provider factory: Token → file → error fallback
- Task ID tracking in session config
- Token sanitization in error messages

## Key Design Patterns

### Configuration Hierarchy

CLI flags override template config (`.kodama.yaml` in repo), which overrides global config (`~/.kodama/config.yaml`), which overrides hardcoded defaults.

### Init Container Strategy

Pods use init containers for setup, built using a config-based architecture:

1. **tools-installer**: Combined installer for Claude Code CLI and ttyd (if enabled) - more efficient than separate containers
2. **workspace-initializer**: Git clone with token injection and branch setup (if repo specified)

The config-based design (`pkg/kubernetes/initcontainer/`) provides:

- **BuildCombined**: Merges multiple installers into a single init container for efficiency
- **InstallerConfig interface**: Add new tools by implementing this interface
- **Independent testing**: Each installer config can be unit tested
- **Configurable versions**: Tool versions and options configurable per session
- **Consistent logging**: Standardized start/completion messages across all installers

### Deferred Cleanup

The usecase layer tracks resources created during session start and cleans them up automatically if any step fails.

### Interface-Based Design

Key components use interfaces to enable testing and future extensibility:

- `SyncManager` - allows alternative sync implementations
- `CodingAgentExecutor` - allows mock execution in tests
- `AuthProvider` - supports multiple auth methods

## Session Lifecycle

### Start Flow

```
1. Load/merge configs (global + template + CLI flags)
2. Validate mutual exclusivity (--repo vs --sync)
3. Create ConfigMap with editor configs
4. Create Pod with init containers
5. Wait for pod ready (monitors init container events)
6. Perform initial file sync (if enabled)
7. Sync custom directories (dotfiles, etc.)
8. Start agent task (if prompt provided)
9. Save session state to ~/.kodama/sessions/<name>.yaml
```

### Attach Flow

```
1. Load session config from ~/.kodama/sessions/<name>.yaml
2. Route to ttyd (web) or TTY (exec) mode
3. TTY: kubectl exec with interactive shell
4. Ttyd: kubectl port-forward + browser launch
```

### Session State Tracking

Sessions track:

- Pod name, namespace, status
- Git repo, branch, commit hash
- Sync configuration (local path, mutagen session)
- Resource allocation (CPU, memory)
- Agent execution history (task ID, status, prompt)

## Important Implementation Details

### Git Authentication

Three methods supported:

1. **Token in environment**: `GITHUB_TOKEN` injected into init container
2. **Token via K8s secret**: Referenced in global config
3. **SSH keys**: Mounted from K8s secret

Tokens are automatically injected into HTTPS URLs during git clone.

### File Sync Exclusions

Priority order for exclusions:

1. Config patterns (from session/global config)
2. `.kodamaignore` file patterns
3. `.gitignore` patterns (if `useGitignore: true`)

### Error Handling

- Detailed error context with troubleshooting hints
- Automatic cleanup of resources on failure
- Token sanitization prevents leaking secrets in error messages

### Port Management

Ttyd mode uses kubectl port-forward with automatic port allocation (default 7681) and browser launch via OS-specific commands.

## Testing Strategy

Tests exist for key packages:

- `pkg/config`: Configuration loading and merging
- `pkg/kubernetes`: Pod spec generation
- `pkg/gitcmd`: Git script generation and validation
- `pkg/sync/exclude`: Pattern matching logic

Run tests with `mise run test` or `go test ./...`.

## Configuration Files

### Global Config (`~/.kodama/config.yaml`)

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
  secretName: git-ssh-key  # Optional K8s secret

sync:
  useGitignore: true
  excludePatterns: ["*.log", "tmp/"]
```

### Session Template (`.kodama.yaml` in repo root)

Per-repository defaults that override global config. Used when starting sessions in that repo.

### Session State (`~/.kodama/sessions/<name>.yaml`)

Tracks complete session state including pod, PVCs, git info, sync status, agent history.

## Dependencies

- **Go 1.25.5**
- **client-go**: Kubernetes API interactions
- **cobra**: CLI framework
- **fsnotify**: File watching for continuous sync
- **go-gitignore**: Gitignore pattern matching
- **mise**: Task runner (optional, can use `go` directly)

## Extension Points

To add new features:

### New Init Container Installer

Implement `initcontainer.InstallerConfig` interface in `pkg/kubernetes/initcontainer/`. Example:

```go
type MyInstallerConfig struct {
    Version       string
    BinVolumeName string
}

func (m *MyInstallerConfig) Name() string { return "my-installer" }
func (m *MyInstallerConfig) Image() string { return "ubuntu:24.04" }
func (m *MyInstallerConfig) Command() []string { return []string{"/bin/bash", "-c"} }
func (m *MyInstallerConfig) Args() []string {
    return []string{BuildScript("Installing...", "Done", "apt-get install -y mytool")}
}
func (m *MyInstallerConfig) VolumeMounts() []corev1.VolumeMount { /* ... */ }
func (m *MyInstallerConfig) EnvVars() []corev1.EnvVar { return []corev1.EnvVar{} }
func (m *MyInstallerConfig) StartMessage() string { return "Installing..." }
func (m *MyInstallerConfig) CompletionMessage() string { return "Done" }
```

Then add to `buildInitContainers()` in `pkg/kubernetes/pod.go`.

### New Sync Implementation

Implement `sync.SyncManager` interface in `pkg/sync/`. Update `pkg/usecase/start.go` to instantiate your implementation.

### New Auth Provider

Implement `auth.AuthProvider` interface in `pkg/agent/auth/`. Add to factory logic in `provider.go`.

### New Command

Create file in `pkg/commands/`, implement cobra command, register in `root.go`.

## Git Workflow

Kodama automatically manages git branches:

- If starting on `main`/`master`/`trunk`, creates a new branch with `branchPrefix` + session name + timestamp
- Otherwise uses the current branch
- Branch and commit hash tracked in session config
