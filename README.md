# Kodama

A kubectl plugin for managing Claude Code sessions in Kubernetes.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [kubectl kodama start](#kubectl-kodama-start)
  - [kubectl kodama list](#kubectl-kodama-list)
  - [kubectl kodama attach](#kubectl-kodama-attach)
  - [kubectl kodama delete](#kubectl-kodama-delete)
- [Advanced Usage](#advanced-usage)
  - [Git Authentication](#git-authentication)
  - [File Synchronization](#file-synchronization)
  - [Environment Variables](#environment-variables)
  - [Custom Editor Configuration](#custom-editor-configuration)
  - [Coding Agent Integration](#coding-agent-integration)
  - [Resource Management](#resource-management)
- [Common Workflows](#common-workflows)
- [Configuration Reference](#configuration-reference)
- [Troubleshooting](#troubleshooting)
- [Frequently Asked Questions](#frequently-asked-questions)
- [Development](#development)
- [Roadmap](#roadmap)
- [Contributing](#contributing)

## Overview

Kodama makes it easy to run Claude Code development sessions in isolated Kubernetes environments. It provides a simple CLI interface for managing session lifecycles without requiring cluster admin privileges.

**Key Features:**

- 🚀 Simple kubectl plugin - no CRDs or controllers required
- 💾 Persistent workspaces across sessions
- 🔄 Real-time file synchronization with automatic excludes
- 🌿 Smart git integration with automatic branch management
- 🔒 Isolated Kubernetes environments with resource limits
- ⚙️ Easy configuration management (global + per-session)
- ✏️ Pre-configured development environment (Helix + Zellij)
- 🔐 GitHub token and SSH key authentication support
- 🤖 Coding agent integration framework
- 📊 Session status tracking and monitoring

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

### Basic Workflow

```bash
# 1. Start a new session with git repo
kubectl kodama start my-session --repo https://github.com/myorg/myrepo

# 2. Attach to the session
kubectl kodama attach my-session

# 3. List all sessions
kubectl kodama list

# 4. Delete the session when done
kubectl kodama delete my-session
```

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

## Usage

### `kubectl kodama start`

Create and start a new Claude Code session.

```bash
kubectl kodama start <session-name> [flags]
```

**Flags:**

- `--repo <url>` - Git repository URL to clone (supports HTTPS and SSH)
- `--branch <name>` - Git branch to checkout (default: detects current branch)
- `--sync <path>` - Local directory to sync (default: current directory)
- `--no-sync` - Disable file synchronization
- `--cpu <limit>` - CPU limit (default: from config or "1")
- `--memory <limit>` - Memory limit (default: from config or "2Gi")
- `--namespace, -n <name>` - Kubernetes namespace (default: "default")
- `--prompt, -p <text>` - Coding agent prompt to execute
- `--prompt-file <path>` - File containing coding agent prompt

**Examples:**

```bash
# Start basic session
kubectl kodama start my-work

# Start with git repository
kubectl kodama start feature-work --repo git@github.com:myorg/myrepo.git

# Start with specific branch and custom resources
kubectl kodama start hotfix --repo https://github.com/myorg/myrepo \
  --branch develop --cpu 2 --memory 4Gi

# Start without sync (useful for large repos)
kubectl kodama start data-analysis --repo https://github.com/myorg/data \
  --no-sync

# Start with local directory sync
kubectl kodama start local-dev --sync /path/to/project

# Start and execute coding agent with prompt
kubectl kodama start automation --repo https://github.com/myorg/api \
  --prompt "Add unit tests for the authentication module"

# Start with prompt from file
kubectl kodama start refactor --repo https://github.com/myorg/app \
  --prompt-file ./tasks/refactor-plan.txt
```

**What happens during start:**

1. Validates session doesn't already exist
2. Creates editor configuration ConfigMap (Helix + Zellij)
3. Creates Kubernetes pod with claude-code image
4. Waits for pod to become ready (up to 5 minutes)
5. Clones git repository (if `--repo` specified)
6. Performs initial file sync from local to pod (if enabled)
7. Starts coding agent (if `--prompt` or `--prompt-file` specified)
8. Saves session state to `~/.kodama/sessions/<name>.yaml`

### `kubectl kodama list`

List all Kodama sessions.

```bash
kubectl kodama list [flags]
```

**Aliases:** `ls`

**Flags:**

- `--all-namespaces, -A` - List sessions across all namespaces
- `--output, -o <format>` - Output format: `table` (default), `yaml`, `json`

**Examples:**

```bash
# List sessions in default namespace
kubectl kodama list

# List sessions across all namespaces
kubectl kodama list -A

# Output as JSON
kubectl kodama list -o json

# Output as YAML
kubectl kodama list -o yaml
```

**Output columns:**

- `NAME` - Session name
- `STATUS` - Pod status (Running, Pending, Failed, etc.)
- `NAMESPACE` - Kubernetes namespace
- `SYNC` - Sync status (Active, Inactive, Error)
- `AGE` - Time since session creation

### `kubectl kodama attach`

Attach to a running session with an interactive shell.

```bash
kubectl kodama attach <session-name> [flags]
```

**Flags:**

- `--command <cmd>` - Execute specific command instead of interactive shell
- `--namespace, -n <name>` - Kubernetes namespace

**Examples:**

```bash
# Open interactive shell in session
kubectl kodama attach my-work

# Execute single command
kubectl kodama attach my-work --command "ls -la /workspace"

# Run tests in session
kubectl kodama attach feature-work --command "npm test"

# Check git status
kubectl kodama attach hotfix --command "git status"
```

**Interactive shell:**

- Default working directory: `/workspace`
- Available editors: `helix` (hx), `vim`, `nano`
- Terminal multiplexer: `zellij` (pre-configured)
- Git installed and configured

### `kubectl kodama delete`

Delete a session and its resources.

```bash
kubectl kodama delete <session-name> [flags]
```

**Flags:**

- `--keep-config` - Keep session configuration file
- `--force, -f` - Skip confirmation prompt
- `--namespace, -n <name>` - Kubernetes namespace

**Examples:**

```bash
# Delete session with confirmation
kubectl kodama delete my-work

# Force delete without confirmation
kubectl kodama delete my-work --force

# Delete but keep config for later reference
kubectl kodama delete old-session --keep-config

# Delete from specific namespace
kubectl kodama delete experiment -n testing --force
```

**What gets deleted:**

- Kubernetes pod
- ConfigMap with editor configuration
- Session state file (unless `--keep-config`)
- Active file sync (if running)

**Note:** Persistent volumes (PVCs) are NOT automatically deleted to preserve data.

## Advanced Usage

### Git Authentication

**Recommended: Use dotenv files (unified with Claude auth):**

Add your GitHub token to `.env` file:

```bash
# .env file
GITHUB_TOKEN=ghp_your_token_here
```

Configure in `.kodama.yaml`:

```yaml
env:
  dotenvFiles:
    - .env
    - .env.local
```

Start session (automatically uses GitHub token from .env):

```bash
kubectl kodama start private-work --repo https://github.com/myorg/private-repo --sync .
```

**Alternative: Environment variable:**

```bash
export GITHUB_TOKEN=ghp_your_token_here
kubectl kodama start private-work --repo https://github.com/myorg/private-repo
```

**Alternative: SSH Keys (for SSH-based git):**

Configure a Kubernetes secret:

```yaml
# In ~/.kodama/config.yaml
git:
  secretName: git-ssh-key
```

Create the secret:

```bash
kubectl create secret generic git-ssh-key \
  --from-file=ssh-privatekey=$HOME/.ssh/id_rsa
```

See `examples/unified-credentials/` for complete unified authentication setup.

### File Synchronization

**Exclude Patterns:**

Create `.kodamaignore` in your project root:

```
# Exclude large directories
node_modules/
target/
dist/
build/

# Exclude cache files
*.cache
.cache/
__pycache__/

# Exclude IDE files
.idea/
.vscode/
*.swp
```

**Gitignore Integration:**

By default, Kodama respects `.gitignore` patterns. Disable in config:

```yaml
# ~/.kodama/config.yaml
sync:
  useGitignore: false
```

### Environment Variables

**Load environment variables from dotenv files:**

Kodama supports loading environment variables from `.env` files and injecting them into your session pod. This is useful for development secrets, API keys, and configuration values.

```bash
# Load from a single .env file
kubectl kodama start my-session --env-file .env --sync .

# Load from multiple files (last file wins for duplicate variables)
kubectl kodama start my-session \
  --env-file .env \
  --env-file .env.local \
  --sync .
```

**Security features:**

- System-critical variables (PATH, HOME, etc.) are automatically excluded
- Kubernetes service variables are never overridden
- Claude authentication variables are protected
- Warning displayed when loading .env files

**Exclude specific variables:**

```bash
# Exclude specific variables from injection
kubectl kodama start my-session \
  --env-file .env \
  --env-exclude VERBOSE \
  --env-exclude DEBUG_MODE \
  --sync .
```

**Template configuration:**

Add to `.kodama.yaml` in your repository:

```yaml
env:
  dotenvFiles:
    - .env
    - .env.local
  excludeVars:
    - CUSTOM_VAR_TO_EXCLUDE
```

**Global configuration:**

Add to `~/.kodama/config.yaml`:

```yaml
defaults:
  env:
    excludeVars:
      - SYSTEM_SPECIFIC_VAR
```

**Important notes:**

- Dotenv files are read from your **local machine** (not from git)
- Keep .env files out of version control (add to .gitignore)
- Variable names must match `^[A-Z_][A-Z0-9_]*$` pattern
- Total environment data must not exceed 1MB (Kubernetes limit)
- Environment secrets are automatically cleaned up when session is deleted

**Unified Credentials Management:**

Use dotenv files to manage **all credentials** (GitHub PAT, Claude Code auth, cloud credentials) in one place:

```bash
# .env file
GITHUB_TOKEN=ghp_your_github_token
CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_claude_token
AWS_ACCESS_KEY_ID=your_aws_key
DATABASE_URL=postgresql://...
```

Configure in `.kodama.yaml`:

```yaml
env:
  dotenvFiles:
    - .env
    - .env.local
```

Start session (all credentials automatically available):

```bash
kubectl kodama start dev --repo https://github.com/myorg/private-repo --sync .
```

Both git operations and Claude Code authentication work automatically! See `examples/unified-credentials/` for complete setup guide.

### Custom Editor Configuration

**Override default editor configs:**

1. Create `.kodama/configs/` in your repository root
2. Add custom configuration files:
   - `helix-config.toml` - Helix editor configuration
   - `helix-languages.toml` - Helix language server settings
   - `zellij-config.kdl` - Zellij terminal multiplexer config

**Example Helix config** (`.kodama/configs/helix-config.toml`):

```toml
theme = "onedark"

[editor]
line-number = "relative"
cursorline = true
auto-save = true

[editor.cursor-shape]
insert = "bar"
normal = "block"
```

### Coding Agent Integration

**Execute tasks via prompt:**

```bash
# Simple task
kubectl kodama start fix-bug \
  --repo https://github.com/myorg/app \
  --prompt "Fix the authentication timeout issue in src/auth.ts"

# Complex task from file
cat > task.txt <<EOF
Refactor the user service to use dependency injection:
1. Extract database operations to a repository layer
2. Add interfaces for all dependencies
3. Update tests to use mocks
EOF

kubectl kodama start refactor-task \
  --repo https://github.com/myorg/backend \
  --prompt-file task.txt
```

### Resource Management

**Custom resource limits per session:**

```bash
# High-memory session for data processing
kubectl kodama start data-work \
  --repo https://github.com/myorg/data-pipeline \
  --cpu 4 --memory 16Gi

# Lightweight session for documentation
kubectl kodama start docs \
  --repo https://github.com/myorg/docs \
  --cpu 0.5 --memory 1Gi
```

**Set defaults in global config:**

```yaml
# ~/.kodama/config.yaml
defaults:
  resources:
    cpu: "2"
    memory: "4Gi"
  storage:
    workspace: "20Gi"
    claudeHome: "2Gi"
```

## Common Workflows

### Working on a Feature Branch

```bash
# Start session with repo and specific branch
kubectl kodama start feature-work \
  --repo git@github.com:myorg/myapp.git \
  --branch feature/new-api

# Attach and develop
kubectl kodama attach feature-work

# Inside the session:
# - Edit code with helix
# - Run tests
# - Commit changes
# (git commands work normally)

# When done
kubectl kodama delete feature-work
```

### Quick Bug Fix

```bash
# Start without local sync for faster startup
kubectl kodama start hotfix \
  --repo https://github.com/myorg/app \
  --branch main \
  --no-sync

# Attach and make changes
kubectl kodama attach hotfix --command "hx src/buggy-file.ts"

# Commit and push from inside session
kubectl kodama attach hotfix --command "git commit -am 'fix: resolve issue' && git push"

# Clean up
kubectl kodama delete hotfix --force
```

### Data Science / Jupyter Notebook Work

```bash
# Start with high memory for data processing
kubectl kodama start data-analysis \
  --repo https://github.com/myorg/data-science \
  --cpu 8 --memory 32Gi \
  --sync ~/projects/data-science

# Files sync automatically as you work locally
# Attach to run notebooks or scripts
kubectl kodama attach data-analysis --command "python analyze.py"

# Keep session running for days
kubectl kodama list  # Check status anytime
```

### Multiple Parallel Sessions

```bash
# Frontend work
kubectl kodama start frontend \
  --repo https://github.com/myorg/web-app \
  --branch develop

# Backend API work
kubectl kodama start backend \
  --repo https://github.com/myorg/api \
  --branch develop

# Database migrations
kubectl kodama start db-migrations \
  --repo https://github.com/myorg/migrations \
  --cpu 1 --memory 1Gi

# List all active sessions
kubectl kodama list

# Work on each independently
kubectl kodama attach frontend
kubectl kodama attach backend
```

### Using Coding Agent for Automation

```bash
# Create prompt file
cat > refactor-task.txt <<EOF
Refactor the authentication service:
1. Extract token validation into a separate module
2. Add comprehensive error handling
3. Write unit tests with >80% coverage
4. Update documentation
EOF

# Start session with coding agent
kubectl kodama start auto-refactor \
  --repo https://github.com/myorg/auth-service \
  --prompt-file refactor-task.txt

# Agent executes task automatically
# Attach to review changes
kubectl kodama attach auto-refactor

# Review, test, and commit
```

### Team Collaboration

```bash
# Team member 1: Start session and share config
kubectl kodama start shared-debug \
  --repo https://github.com/myorg/app \
  --branch bug-investigation

# Share session name and namespace with team
kubectl kodama list -o yaml > shared-session.yaml

# Team member 2: Attach to same session (same cluster)
kubectl kodama attach shared-debug

# Both can work in same environment
# Useful for pair programming or debugging
```

### Local Development with Sync

```bash
# Start session with current directory sync
cd ~/projects/myapp
kubectl kodama start local-dev --repo . --sync .

# Edit files locally in your favorite IDE
# Changes automatically sync to pod

# Run tests in pod environment
kubectl kodama attach local-dev --command "npm test"

# Build in pod
kubectl kodama attach local-dev --command "npm run build"

# Interactive debugging
kubectl kodama attach local-dev
```

### Long-Running Background Tasks

```bash
# Start session for long-running process
kubectl kodama start ml-training \
  --repo https://github.com/myorg/ml-models \
  --cpu 16 --memory 64Gi

# Start training in background
kubectl kodama attach ml-training --command \
  "nohup python train.py --epochs 1000 > training.log 2>&1 &"

# Detach (session keeps running)
# Check progress later
kubectl kodama attach ml-training --command "tail -f training.log"

# Session runs until you delete it
kubectl kodama delete ml-training
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

### ✅ Phase 1: Foundation (Complete)

- [x] Go module initialization
- [x] CLI framework (cobra)
- [x] Configuration management
- [x] Kubernetes client setup
- [x] Build infrastructure
- [x] Session state persistence

### ✅ Phase 2: Core Session Management (Complete)

- [x] `start` command - Create new sessions with git clone
- [x] `list` command - List all sessions with status
- [x] `attach` command - Interactive shell access
- [x] `delete` command - Remove sessions and resources
- [x] File synchronization (initial + continuous)
- [x] Editor configuration (Helix + Zellij)
- [x] Git integration with branch management
- [x] Coding agent interface framework

### 🚧 Phase 3: Advanced Features (In Progress)

- [x] Basic file sync with fsnotify
- [ ] Enhanced sync with mutagen integration
- [ ] `stop` / `resume` commands for session lifecycle
- [ ] Coding agent execution with Claude Code CLI
- [ ] Agent task status tracking
- [ ] Session templates
- [ ] Multi-container sessions
- [ ] Volume snapshot/restore

### 📅 Phase 4: Production Readiness

- [ ] Comprehensive end-to-end testing
- [ ] Performance optimization
- [ ] Advanced monitoring and logging
- [ ] Web UI for session management
- [ ] Team collaboration features
- [ ] Session sharing and cloning

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
  cpu: "1"      # CPU limit (can use decimals like "0.5" or "2")
  memory: "2Gi" # Memory limit (use Mi for megabytes, Gi for gigabytes)
```

### Storage Configuration

Default storage sizes:

```yaml
storage:
  workspace: "10Gi"   # Workspace PVC size (where your code lives)
  claudeHome: "1Gi"   # Claude home directory size (.claude config)
```

### Complete Configuration Example

```yaml
# ~/.kodama/config.yaml
defaults:
  namespace: dev-sessions
  image: ghcr.io/illumination-k/kodama-claude:latest

  resources:
    cpu: "2"
    memory: "4Gi"

  storage:
    workspace: "20Gi"
    claudeHome: "2Gi"

  branchPrefix: "kodama/"

sync:
  useGitignore: true       # Respect .gitignore patterns (default: true)
  excludePatterns:         # Additional patterns to exclude from sync
    - "*.log"
    - "tmp/"
    - ".DS_Store"
```

### Environment Variables

Kodama supports the following environment variables:

- `GITHUB_TOKEN` - GitHub personal access token for private repo access
- `KUBECONFIG` - Path to kubeconfig file (default: `~/.kube/config`)
- `KODAMA_CONFIG_DIR` - Config directory (default: `~/.kodama`)

## Troubleshooting

### Session Won't Start

**Check pod status:**

```bash
kubectl kodama list
kubectl get pod kodama-<session-name> -n <namespace>
kubectl describe pod kodama-<session-name> -n <namespace>
```

**Common issues:**

- Insufficient cluster resources (CPU/Memory)
- Image pull errors (check image name and registry access)
- PVC creation failures (check storage class availability)

### File Sync Not Working

**Verify sync status:**

```bash
kubectl kodama list  # Check SYNC column
```

**Common issues:**

- Large files or directories being synced (add to `.kodamaignore`)
- Permission issues on local directory
- Pod not in Running state

**Manual sync verification:**

```bash
# Check files in pod
kubectl kodama attach my-session --command "ls -la /workspace"

# Compare with local
ls -la /path/to/local/project
```

### Git Authentication Failures

**For HTTPS (private repos):**

```bash
# Verify token is set
echo $GITHUB_TOKEN

# Test token manually
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

**For SSH:**

```bash
# Verify secret exists
kubectl get secret git-ssh-key

# Check secret content
kubectl get secret git-ssh-key -o yaml
```

### Session Stuck in Pending

**Check pod events:**

```bash
kubectl get events --field-selector involvedObject.name=kodama-<session-name>
```

**Common causes:**

- Node resources exhausted
- Storage provisioning delay
- Image pull backoff

### Delete Session Manually

If `kubectl kodama delete` fails:

```bash
# Delete pod directly
kubectl delete pod kodama-<session-name>

# Delete ConfigMap
kubectl delete configmap kodama-editor-config-<session-name>

# Remove session config
rm ~/.kodama/sessions/<session-name>.yaml

# Optional: Delete PVCs (WARNING: destroys data)
kubectl delete pvc kodama-workspace-<session-name>
kubectl delete pvc kodama-claude-home-<session-name>
```

## Frequently Asked Questions

### Why use Kodama instead of local Claude Code?

- **Consistent environment**: Same OS, tools, and dependencies across team
- **Resource isolation**: Don't consume local CPU/memory for heavy tasks
- **Persistent state**: Sessions survive local machine restarts
- **Team collaboration**: Multiple developers can access same environment
- **Cloud resources**: Leverage Kubernetes cluster compute power

### Does Kodama require cluster admin privileges?

No. Kodama only requires permissions to:

- Create/delete pods in specific namespaces
- Create/delete ConfigMaps
- Execute commands in pods

Standard developer access to a namespace is sufficient.

### What happens to my data when I delete a session?

By default, PVCs (persistent volume claims) are **not** automatically deleted. Your workspace and Claude home directory data persist even after deleting the session. To permanently delete data, manually delete the PVCs:

```bash
kubectl delete pvc kodama-workspace-<session-name>
kubectl delete pvc kodama-claude-home-<session-name>
```

### Can I use Kodama with private git repositories?

Yes. Use either:

1. **GitHub token**: Set `GITHUB_TOKEN` environment variable
2. **SSH keys**: Configure a Kubernetes secret with your SSH private key

See [Git Authentication](#git-authentication) for details.

### How does file synchronization work?

Kodama uses a two-phase sync approach:

1. **Initial sync**: Tar-based bulk transfer when session starts
2. **Continuous sync**: File watcher (fsnotify) detects local changes and copies to pod

Files matching `.gitignore` and `.kodamaignore` patterns are automatically excluded.

### Can I use my own Docker image?

Yes. Specify a custom image in your global config:

```yaml
# ~/.kodama/config.yaml
defaults:
  image: my-registry.com/my-claude-image:latest
```

Your image should include:

- Claude Code CLI
- Git
- Any editors/tools you need

### How do I share a session with teammates?

Share the session name and namespace:

```bash
kubectl kodama list -o yaml > session-info.yaml
# Share session-info.yaml with teammate
```

Teammates with access to the same Kubernetes cluster can attach:

```bash
kubectl kodama attach <session-name> -n <namespace>
```

### What editors are available in sessions?

By default:

- **Helix** (hx) - Modern terminal editor with LSP support
- **Vim** - Classic editor
- **Nano** - Simple editor

You can customize editor configurations via `.kodama/configs/` in your repo.

### Can I run GUI applications?

No. Kodama is designed for terminal-based development. For GUI applications, consider:

- Port forwarding with `kubectl port-forward`
- Using web-based IDEs
- Remote desktop solutions like X11 forwarding

### How much does it cost to run Kodama?

Cost depends on your Kubernetes cluster provider and resource usage:

- **Cloud providers**: Pay for pod CPU/memory/storage (typically $0.04-0.10/hour for basic session)
- **Self-hosted**: No additional cost beyond infrastructure
- **Idle sessions**: Still consume resources (consider deleting when not in use)

### Does Kodama support Windows?

Yes. The Kodama CLI works on Windows, macOS, and Linux. However:

- File sync paths use OS-specific conventions
- Git authentication may require different setup on Windows
- Sessions run in Linux containers regardless of host OS

### Can I run multiple commands in the background?

Yes. Use `kubectl kodama attach` with `--command` and shell job control:

```bash
# Start background process
kubectl kodama attach my-session --command "nohup ./long-task.sh > output.log 2>&1 &"

# Check on it later
kubectl kodama attach my-session --command "tail -f output.log"
```

### How do I update Kodama?

Rebuild and reinstall from source:

```bash
cd kodama
git pull
mise run build
mise run dev-install
```

Or with Go directly:

```bash
go build -o kubectl-kodama ./cmd/kubectl-kodama
cp kubectl-kodama ~/.local/bin/
```

## Contributing

Contributions are welcome! Here's how you can help:

### Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include: Kodama version, Kubernetes version, error messages
- Provide minimal reproduction steps

### Development Setup

```bash
# Fork and clone the repository
git clone https://github.com/yourusername/kodama.git
cd kodama

# Install dependencies
mise install

# Build and test
mise run build
mise run test
mise run lint
```

### Pull Request Guidelines

1. Create a feature branch: `git checkout -b feature/my-feature`
2. Write tests for new functionality
3. Ensure all tests pass: `mise run test`
4. Run linter: `mise run lint`
5. Format code: `mise run fmt`
6. Submit PR with clear description

### Areas for Contribution

- Additional editor integrations (VS Code, Emacs)
- Enhanced file sync (mutagen integration)
- Session templates and presets
- Web UI for session management
- Documentation improvements
- Bug fixes and performance optimizations

## License

MIT

## Acknowledgments

Inspired by [DevPod](https://github.com/loft-sh/devpod) and [Devspace](https://github.com/devspace-sh/devspace).
