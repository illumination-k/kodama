# Unified Credentials Management with Dotenv Files

This example demonstrates how to manage **all credentials** (GitHub PAT, Claude Code authentication, cloud provider credentials, etc.) through dotenv files for a unified, simple approach.

## Overview

Instead of managing credentials through multiple mechanisms:

- ❌ Kubernetes secrets for git authentication
- ❌ Kubernetes secrets for Claude auth
- ❌ File-based Claude authentication
- ❌ Environment variables set separately

Use a **single unified approach**:

- ✅ All credentials in `.env` files
- ✅ Automatically loaded and injected into pods
- ✅ Works seamlessly with session templates (`.kodama.yaml`)

## Quick Start

### 1. Setup

```bash
# Copy example files to your repository
cd /path/to/your/repo
cp .kodama.yaml .
cp .env.example .env

# Edit .env with your actual credentials
vim .env
```

### 2. Add to .gitignore

```bash
# CRITICAL: Never commit credentials to git!
echo ".env" >> .gitignore
echo ".env.*" >> .gitignore
echo "!.env.example" >> .gitignore
```

### 3. Start Session

```bash
# Credentials are automatically loaded from .env
kubectl kodama start dev --sync .
```

That's it! Both GitHub and Claude Code authentication now work automatically.

## How It Works

### Session Start Flow

1. **Load dotenv files** (from your local machine)
   ```bash
   # .env file contains:
   GITHUB_TOKEN=ghp_xxx
   CLAUDE_CODE_AUTH_TOKEN=sk-ant-xxx
   ```

2. **Create Kubernetes secret**
   - Secret name: `kodama-env-{session-name}`
   - Contains all environment variables as key-value pairs
   - Labeled with `app: kodama`, `session: {name}`

3. **Inject into pod**
   - Pod spec includes `envFrom` referencing the secret
   - All variables available as environment variables

4. **Authentication works automatically**
   - Git operations use `GITHUB_TOKEN`
   - Claude Code uses `CLAUDE_CODE_AUTH_TOKEN`
   - No additional configuration needed!

5. **Cleanup on deletion**
   - Secret automatically deleted when session ends
   - Also cleaned up if session start fails

## Configuration

### `.kodama.yaml` Template

```yaml
env:
  dotenvFiles:
    - .env              # Base credentials
    - .env.credentials  # Additional credentials
    - .env.local        # Local overrides

  # Optional: Exclude specific variables
  # excludeVars:
  #   - DEBUG_MODE
```

### `.env` File Structure

```bash
# Git Authentication
GITHUB_TOKEN=ghp_your_github_token

# Claude Code Authentication
CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_claude_token

# Application Variables
DATABASE_URL=postgresql://...
API_KEY=your_api_key
```

## Supported Credentials

### Git Authentication

Kodama automatically uses these variables for git operations:

- `GITHUB_TOKEN` - GitHub Personal Access Token
- `GH_TOKEN` - Alternative GitHub token variable

**Generate GitHub PAT:**

1. Visit https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Select scopes: `repo` (for private repos)
4. Copy token to your `.env` file

### Claude Code Authentication

Kodama automatically uses these variables for Claude authentication:

- `CLAUDE_CODE_AUTH_TOKEN` - Claude API key
- `ANTHROPIC_API_KEY` - Alternative Anthropic API key variable

**Obtain Claude API Key:**

1. Visit https://console.anthropic.com/settings/keys
2. Create new key
3. Copy to your `.env` file

### Cloud Provider Credentials

Also supported (examples):

```bash
# AWS
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Google Cloud
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Azure
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret
```

## Migration from Other Methods

### From Kubernetes Secrets (Git)

**Before:**

```bash
# Create K8s secret manually
kubectl create secret generic git-secret \
  --from-literal=token=ghp_xxx

# Reference in config
# ~/.kodama/config.yaml
git:
  secretName: git-secret
```

**After:**

```bash
# Simply add to .env
GITHUB_TOKEN=ghp_xxx

# No config changes needed!
```

### From Kubernetes Secrets (Claude Auth)

**Before:**

```bash
# Create K8s secret
kubectl create secret generic claude-token \
  --from-literal=token=sk-ant-xxx

# Reference in config
# ~/.kodama/config.yaml
claude:
  authType: token
  token:
    secretName: claude-token
```

**After:**

```bash
# Simply add to .env
CLAUDE_CODE_AUTH_TOKEN=sk-ant-xxx

# No config changes needed!
```

### From File-based Authentication

**Before:**

```bash
# Create auth file
mkdir -p ~/.kodama
cat > ~/.kodama/claude-auth.json <<EOF
{
  "default": {
    "api_key": "sk-ant-xxx"
  }
}
EOF

# Configure
# ~/.kodama/config.yaml
claude:
  authType: file
  file:
    path: ~/.kodama/claude-auth.json
```

**After:**

```bash
# Simply add to .env
CLAUDE_CODE_AUTH_TOKEN=sk-ant-xxx

# Delete old files
rm ~/.kodama/claude-auth.json
```

## CLI Override

Override template configuration with flags:

```bash
# Use different dotenv file
kubectl kodama start prod \
  --env-file .env.production \
  --sync .

# Multiple files (last wins)
kubectl kodama start dev \
  --env-file .env \
  --env-file .env.local \
  --sync .

# Exclude specific variables
kubectl kodama start dev \
  --env-exclude VERBOSE \
  --sync .
```

## Environment-Specific Credentials

Manage different environments with separate files:

```bash
# Development
.env.development:
  GITHUB_TOKEN=ghp_dev_token
  CLAUDE_CODE_AUTH_TOKEN=sk-ant-dev-token
  DATABASE_URL=postgresql://localhost/dev

# Production
.env.production:
  GITHUB_TOKEN=ghp_prod_token
  CLAUDE_CODE_AUTH_TOKEN=sk-ant-prod-token
  DATABASE_URL=postgresql://prod-db/prod
```

Start with specific environment:

```bash
# Development
kubectl kodama start dev --env-file .env.development --sync .

# Production
kubectl kodama start prod --env-file .env.production --sync .
```

## Security Best Practices

### 1. Never Commit Credentials

```bash
# .gitignore
.env
.env.*
!.env.example
```

### 2. Use .env.example for Documentation

```bash
# .env.example (committed to git)
GITHUB_TOKEN=ghp_your_token_here
CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_token_here
DATABASE_URL=postgresql://...

# .env (NOT committed - actual values)
GITHUB_TOKEN=ghp_actual_secret_token
CLAUDE_CODE_AUTH_TOKEN=sk-ant-actual_secret_key
DATABASE_URL=postgresql://user:pass@host/db
```

### 3. Validate .gitignore

```bash
# Check that .env is ignored
git status --ignored

# Should show .env in ignored files
# If not, add to .gitignore immediately!
```

### 4. Rotate Credentials Regularly

```bash
# Update .env with new tokens
vim .env

# Restart session to use new credentials
kubectl kodama delete dev
kubectl kodama start dev --sync .
```

### 5. Use Environment-Specific Files

```bash
# Developer-specific overrides
.env.${USER}  # e.g., .env.alice, .env.bob

# Template configuration
env:
  dotenvFiles:
    - .env
    - .env.${USER}
```

## Protected Variables

These system-critical variables **cannot be overridden** (always excluded):

- System: `PATH`, `HOME`, `USER`, `SHELL`, `TERM`, `PWD`
- Kubernetes: `KUBERNETES_SERVICE_*`, `KUBERNETES_PORT_*`

All other variables (including credentials) are allowed in `.env` files.

## Troubleshooting

### Credentials Not Working

1. **Check variable names**
   ```bash
   # Must be uppercase with underscores
   GITHUB_TOKEN=...     # ✓ Correct
   github_token=...     # ✗ Wrong
   GitHub-Token=...     # ✗ Wrong
   ```

2. **Verify .env file is loaded**
   ```bash
   # Start with verbose output
   kubectl kodama start dev --sync . 2>&1 | grep "Loading dotenv"
   # Should show: "📝 Loading dotenv files..."
   ```

3. **Check in pod**
   ```bash
   kubectl kodama attach dev
   env | grep GITHUB_TOKEN
   env | grep CLAUDE_CODE_AUTH_TOKEN
   ```

### Secret Size Limit

If total environment data exceeds 1MB:

```bash
# Error: environment variables exceed Kubernetes secret size limit

# Solutions:
# 1. Split into multiple files with selective loading
# 2. Store large values in ConfigMaps instead
# 3. Use file-based configs for large data
```

### Variable Excluded

If a variable isn't appearing:

```bash
# Check if it's being excluded
# In .kodama.yaml:
env:
  excludeVars:
    - MY_VAR  # This will be excluded

# Remove from exclusions or use different name
```

## Benefits

✅ **Simplicity** - Single unified approach for all credentials
✅ **Portability** - Works on any machine with local .env files
✅ **Flexibility** - Easy to switch between environments
✅ **Security** - Automatic cleanup, no persistent K8s secrets
✅ **Standard** - Uses industry-standard .env file format
✅ **Developer-Friendly** - Familiar workflow, minimal configuration

## Complete Example

```bash
# 1. Setup repository
cd /path/to/project

# 2. Create .kodama.yaml
cat > .kodama.yaml <<EOF
env:
  dotenvFiles:
    - .env
    - .env.local
EOF

# 3. Create .env file
cat > .env <<EOF
GITHUB_TOKEN=ghp_your_github_token
CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_claude_token
DATABASE_URL=postgresql://localhost:5432/mydb
API_KEY=your_api_key
EOF

# 4. Add to .gitignore
cat >> .gitignore <<EOF
.env
.env.*
!.env.example
EOF

# 5. Start session (auto-loads credentials)
kubectl kodama start dev --sync .

# 6. Verify in pod
kubectl kodama attach dev
env | grep -E "GITHUB_TOKEN|CLAUDE_CODE_AUTH_TOKEN"

# 7. Cleanup
kubectl kodama delete dev  # Automatically deletes secret
```

🎉 **All credentials are now managed through .env files!**
