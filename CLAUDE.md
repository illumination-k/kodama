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

Kodama follows **Hexagonal Architecture (Ports & Adapters)** with clear separation of concerns and dependency inversion:

### Layer Structure

```
Presentation Layer (pkg/presentation)
  ↓ uses
Application Layer (pkg/application)
  ↓ uses (via port interfaces)
Infrastructure Layer (pkg/infrastructure)
  ↓ implements
Port Interfaces (pkg/application/port)
  ↑ defined by application needs
```

**Dependency Rule**: Outer layers depend on inner layers. Infrastructure implements interfaces defined by the application layer.

### Directory Structure

```
pkg/
├── presentation/          # CLI layer (Cobra commands)
│   └── commands/         # delete.go, list.go (refactored)
│
├── application/          # Application services & orchestration
│   ├── port/            # Port interfaces (dependency inversion)
│   │   ├── repository.go    # SessionRepository, ConfigRepository
│   │   ├── kubernetes.go    # KubernetesClient
│   │   ├── sync.go         # SyncManager
│   │   └── agent.go        # AgentExecutor
│   ├── service/         # Service layer
│   │   └── session.go       # SessionService (DI container)
│   └── app.go           # Dependency injection wiring
│
├── infrastructure/       # Adapters implementing ports
│   ├── kubernetes/      # Wraps pkg/kubernetes
│   │   └── adapter.go
│   ├── sync/            # Wraps pkg/sync
│   │   └── adapter.go
│   ├── agent/           # Wraps pkg/agent
│   │   └── adapter.go
│   └── repository/      # File-based persistence
│       ├── session_file.go
│       └── config_file.go
│
├── commands/            # Legacy commands (being migrated)
│   ├── start.go        # TODO: Refactor to use SessionService
│   ├── attach.go       # TODO: Refactor to use SessionService
│   └── dev.go          # TODO: Refactor to use SessionService
│
├── usecase/            # Legacy orchestration (being migrated)
│   └── session.go      # TODO: Split into focused use cases
│
└── [domain packages]   # Core business logic
    ├── config/         # Session & global configuration
    ├── kubernetes/     # K8s client implementation
    ├── sync/           # File synchronization
    ├── agent/          # Coding agent execution
    ├── env/            # Environment variable handling
    ├── gitcmd/         # Git command generation
    └── secretfile/     # Secret file handling
```

### Key Components

#### `pkg/application/app.go` - Dependency Injection

Single source of truth for wiring all dependencies:

```go
type App struct {
    SessionService *service.SessionService
}

func NewApp(kubeconfigPath string) (*App, error) {
    // Create infrastructure adapters
    k8sClient, _ := kubernetesAdapter.NewAdapter(kubeconfigPath)
    syncMgr := syncAdapter.NewAdapter()
    agentExec := agentAdapter.NewAdapter()
    sessionRepo, _ := repository.NewSessionFileRepository()
    configRepo, _ := repository.NewConfigFileRepository()

    // Wire services with constructor injection
    sessionService := service.NewSessionService(
        sessionRepo, configRepo, k8sClient, syncMgr, agentExec,
    )

    return &App{SessionService: sessionService}, nil
}
```

#### `pkg/application/service/session.go` - SessionService

Provides high-level session operations using dependency injection:

- Accepts all dependencies as port interfaces
- Provides session CRUD operations
- Delegates to infrastructure adapters
- Acts as facade for presentation layer

#### `pkg/application/port/` - Port Interfaces

Dependency inversion interfaces defining contracts:

- **SessionRepository**: Session persistence (LoadSession, SaveSession, DeleteSession, ListSessions)
- **ConfigRepository**: Global config persistence (LoadGlobalConfig, SaveGlobalConfig)
- **KubernetesClient**: K8s operations (CreatePod, DeletePod, GetPod, CreateSecret, etc.)
- **SyncManager**: File synchronization (InitialSync, Start, Stop, Status)
- **AgentExecutor**: Coding agent tasks (TaskStart)

#### `pkg/infrastructure/` - Adapters

Thin wrappers implementing port interfaces:

- **kubernetes/adapter.go**: Wraps existing `pkg/kubernetes.Client`
- **sync/adapter.go**: Wraps existing `pkg/sync.SyncManager`
- **agent/adapter.go**: Wraps existing `pkg/agent.CodingAgentExecutor`
- **repository/**: File-based storage for sessions and config

#### `pkg/presentation/commands/` - CLI Layer

Cobra commands using dependency injection:

- **delete.go**: Refactored ✅ - Uses SessionService instead of direct clients
- **list.go**: Refactored ✅ - Uses SessionService instead of direct clients
- **start.go, attach.go, dev.go**: Legacy (in `pkg/commands/`) - TODO: Migrate

#### `pkg/config/` - Domain Configuration

Multi-tier configuration system (domain entities):

- **SessionConfig**: Session state (pod, namespace, repo, sync, resources, agent history)
- **GlobalConfig**: Global defaults and sync configuration
- **ConfigResolver**: Merges global + template + CLI flags
- **Store**: File-based persistence (wrapped by infrastructure/repository)
- **Priority**: CLI flags > template config (`.kodama.yaml`) > global config > defaults

### Migration Status

The codebase is **in transition** to the new Hexagonal Architecture:

#### ✅ Refactored (New Architecture)

- `pkg/application/` - Complete with ports, services, and DI wiring
- `pkg/infrastructure/` - All adapters implemented
- `pkg/presentation/commands/delete.go` - Uses SessionService ✅
- `pkg/presentation/commands/list.go` - Uses SessionService ✅
- `cmd/kubectl-kodama/main.go` - Bootstraps with DI ✅

#### ⏳ Legacy (To Be Migrated)

- `pkg/commands/start.go` - Still uses direct client instantiation
- `pkg/commands/attach.go` - Still uses direct client instantiation
- `pkg/commands/dev.go` - Still uses direct client instantiation
- `pkg/usecase/session.go` - 800-line god function, should be split

#### 📝 Migration Priority

1. Refactor `start.go` - Most complex, highest value
2. Refactor `attach.go` - Moderate complexity
3. Refactor `dev.go` - Depends on start + attach
4. Split `usecase/session.go` into focused use cases
5. Delete `pkg/commands/` (replaced by `pkg/presentation/commands/`)

**Note**: Both old and new commands work simultaneously during migration. The CLI is fully functional.

### Core Domain Packages

The following packages contain domain logic and implementations. In the new architecture, they are accessed via infrastructure adapters (not directly from presentation layer).

#### `pkg/env/`

Dotenv file loading and environment variable injection:

- **LoadDotenvFiles**: Parses dotenv files with last-wins precedence for duplicate variables
- **ApplyExclusions**: Filters out system-critical and user-specified variables
- **ValidateVarName**: Ensures variable names match `^[A-Z_][A-Z0-9_]*$` pattern
- **ValidateSecretSize**: Checks that environment data doesn't exceed 1MB K8s secret limit
- **DefaultExcludedVars**: System variables that should never be overridden (PATH, HOME, K8s vars, Claude auth)
- Files are read from local machine (where `kubectl kodama` runs), not from git repo
- Creates K8s secrets with environment variables and injects via `envFrom`
- Secrets are automatically cleaned up on session deletion

#### `pkg/kubernetes/`

Kubernetes abstraction layer:

- Pod creation with multi-init-container strategy
- Status monitoring via pod watch API
- Environment variable injection via `envFrom` with K8s secrets
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

### Dependency Inversion Principle

All infrastructure dependencies are accessed via port interfaces defined in `pkg/application/port/`. This allows:

- **Testability**: Mock implementations for unit testing
- **Flexibility**: Swap implementations without changing application logic
- **Clear boundaries**: Application layer doesn't depend on infrastructure

Example:

```go
// Application layer defines what it needs (port)
type KubernetesClient interface {
    CreatePod(ctx context.Context, spec *PodSpec) error
    DeletePod(ctx context.Context, name, namespace string) error
}

// Infrastructure layer provides implementation (adapter)
type Adapter struct {
    client *k8s.Client
}

func (a *Adapter) CreatePod(ctx context.Context, spec *PodSpec) error {
    return a.client.CreatePod(ctx, spec)
}
```

### Constructor Injection

All services receive dependencies through constructors:

```go
func NewSessionService(
    sessionRepo port.SessionRepository,
    configRepo port.ConfigRepository,
    k8sClient port.KubernetesClient,
    syncMgr port.SyncManager,
    agentExecutor port.AgentExecutor,
) *SessionService {
    return &SessionService{...}
}
```

Benefits:

- Dependencies are explicit and compile-time checked
- Easy to test with mock implementations
- Single wiring location in `app.go`

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

Port interfaces enable testing and future extensibility:

- `SessionRepository` / `ConfigRepository` - allows alternative storage backends
- `KubernetesClient` - allows mocking K8s operations
- `SyncManager` - allows alternative sync implementations (mutagen, etc.)
- `AgentExecutor` - allows mock execution in tests
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

Git authentication is handled via environment variables from .env files:

- `GITHUB_TOKEN` or `GH_TOKEN` - Automatically injected into HTTPS URLs during git clone
- Tokens are loaded from .env files and made available to init containers via K8s secrets

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
- `pkg/env`: Dotenv parsing, validation, and exclusions
- `pkg/kubernetes`: Pod spec generation and secret management
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

sync:
  useGitignore: true
  excludePatterns: ["*.log", "tmp/"]

env:
  excludeVars: []  # Additional vars to exclude beyond defaults
```

### Session Template (`.kodama.yaml` in repo root)

Per-repository defaults that override global config. Used when starting sessions in that repo.

```yaml
env:
  dotenvFiles:
    - .env
    - .env.local
  excludeVars:
    - VERBOSE  # Example: don't inject this var
```

### Session State (`~/.kodama/sessions/<name>.yaml`)

Tracks complete session state including pod, PVCs, git info, sync status, agent history, and environment secret info (secret name, creation status).

## Dependencies

- **Go 1.25.5**
- **client-go**: Kubernetes API interactions
- **cobra**: CLI framework
- **fsnotify**: File watching for continuous sync
- **go-gitignore**: Gitignore pattern matching
- **mise**: Task runner (optional, can use `go` directly)

## Extension Points

### Adding New Features

#### New Command (Recommended Pattern)

Create command in `pkg/presentation/commands/` using dependency injection:

```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/illumination-k/kodama/pkg/application/service"
)

func NewMyCommand(sessionService *service.SessionService) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "Description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Use sessionService methods
            sessions, err := sessionService.ListSessions()
            // ... business logic
            return nil
        },
    }
    return cmd
}
```

Register in `pkg/presentation/commands/root.go`:

```go
cmd.AddCommand(NewMyCommand(app.SessionService))
```

#### New Port Interface (for new infrastructure)

Define interface in `pkg/application/port/`:

```go
package port

type DatabaseClient interface {
    Query(sql string) ([]Row, error)
    Execute(sql string) error
}
```

Create adapter in `pkg/infrastructure/database/`:

```go
package database

import "github.com/illumination-k/kodama/pkg/application/port"

type Adapter struct {
    db *SomeDB
}

func NewAdapter(connStr string) (port.DatabaseClient, error) {
    db, err := SomeDB.Connect(connStr)
    return &Adapter{db: db}, err
}

func (a *Adapter) Query(sql string) ([]Row, error) {
    return a.db.Query(sql)
}
```

Wire in `pkg/application/app.go`:

```go
dbClient, _ := database.NewAdapter(connStr)
sessionService := service.NewSessionService(
    sessionRepo, configRepo, k8sClient, syncMgr, agentExec, dbClient,
)
```

#### New Init Container Installer

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

#### New Sync Implementation

Implement `port.SyncManager` interface:

```go
package mysync

import "github.com/illumination-k/kodama/pkg/application/port"

type MyAdapter struct {
    // ... fields
}

func NewAdapter() port.SyncManager {
    return &MyAdapter{}
}

func (a *MyAdapter) InitialSync(ctx, localPath, namespace, podName string, cfg *exclude.Config) error {
    // Your implementation
}
// ... implement other methods
```

Update `pkg/application/app.go` to use your adapter:

```go
syncMgr := mysync.NewAdapter()  // Instead of syncAdapter.NewAdapter()
```

#### New Auth Provider

Implement `auth.AuthProvider` interface in `pkg/agent/auth/`. Add to factory logic in `provider.go`.

### Migration Guidelines

When refactoring legacy commands to new architecture:

1. **Create new command** in `pkg/presentation/commands/`
2. **Accept SessionService** via constructor injection
3. **Use SessionService methods** instead of direct client instantiation:
   - ❌ `store, _ := config.NewStore(); session, _ := store.LoadSession(name)`
   - ✅ `session, _ := sessionService.LoadSession(name)`
4. **Update root.go** to pass SessionService to new command
5. **Test** that command works with new architecture
6. **Remove old command** from `pkg/commands/`

### Architecture Validation

Before committing changes, verify layer boundaries:

```bash
# Presentation layer should NOT import infrastructure directly
grep -r "pkg/kubernetes" pkg/presentation/commands/  # Should return nothing
grep -r "pkg/sync" pkg/presentation/commands/        # Should return nothing

# Build and test
go build -o bin/kubectl-kodama ./cmd/kubectl-kodama
go test ./...
```

### Testing with Mocks

Example: Testing a command with mock SessionService

```go
// Create mock implementing port.SessionRepository
type MockSessionRepo struct {
    sessions []*config.SessionConfig
}

func (m *MockSessionRepo) LoadSession(name string) (*config.SessionConfig, error) {
    // Return test data
}

// Wire mock into SessionService
sessionService := service.NewSessionService(
    mockSessionRepo,
    mockConfigRepo,
    mockK8sClient,
    mockSyncMgr,
    mockAgentExec,
)

// Test command using mocked service
```

## Benefits of Hexagonal Architecture

The refactoring to Hexagonal Architecture provides significant improvements:

### 1. **Testability**

- All dependencies are injectable via port interfaces
- Easy to create mocks for unit testing
- Commands can be tested without K8s cluster
- Example: Test delete command with mock SessionService

### 2. **Maintainability**

- Clear separation of concerns (Presentation → Application → Infrastructure)
- No circular dependencies
- Each layer has single responsibility
- Reduced code duplication (session operations centralized in SessionService)

### 3. **Flexibility**

- Swap implementations without changing application logic
- Example: Replace file-based storage with database without touching commands
- Example: Replace kubectl cp sync with mutagen without changing use cases

### 4. **Clear Boundaries**

- Presentation layer cannot import infrastructure directly
- Enforced at compile time
- Architecture violations are easy to detect with grep

### 5. **Dependency Inversion**

- High-level policy (application) doesn't depend on low-level details (infrastructure)
- Infrastructure implements interfaces defined by application needs
- Aligns with SOLID principles

### 6. **Single Source of Truth**

- All dependency wiring happens in one place (`pkg/application/app.go`)
- Easy to understand how components are connected
- Reduces "where is this instantiated?" questions

### 7. **Incremental Migration**

- Old and new architectures coexist during migration
- No big-bang rewrite required
- Commands can be migrated one at a time
- CLI remains fully functional throughout migration

## Git Workflow

Kodama automatically manages git branches:

- If starting on `main`/`master`/`trunk`, creates a new branch with `branchPrefix` + session name + timestamp
- Otherwise uses the current branch
- Branch and commit hash tracked in session config
