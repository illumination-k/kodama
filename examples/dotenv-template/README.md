# Dotenv Template Example

This example demonstrates how to use session templates (`.kodama.yaml`) with environment variable support.

## Setup

1. **Copy the example files to your repository:**

```bash
# Copy template to your repo root
cp .kodama.yaml /path/to/your/repo/

# Copy and customize dotenv files
cp .env.example /path/to/your/repo/.env
cp .env.local.example /path/to/your/repo/.env.local
```

2. **Edit the dotenv files with your actual values:**

```bash
cd /path/to/your/repo
vim .env        # Add your environment variables
vim .env.local  # Add local overrides
```

3. **Ensure .env files are in .gitignore:**

```bash
# Add to .gitignore
echo ".env" >> .gitignore
echo ".env.local" >> .gitignore
```

## Usage

### Start a session (dotenv files loaded automatically)

When you start a Kodama session in a repository with `.kodama.yaml`, the environment files are automatically loaded:

```bash
cd /path/to/your/repo
kubectl kodama start my-session --sync .
```

The session will:

1. Read `.env` from your local machine
2. Read `.env.local` from your local machine (overrides duplicate vars from `.env`)
3. Exclude `VERBOSE` and `DEBUG_MODE` (configured in `.kodama.yaml`)
4. Create a Kubernetes secret with the variables
5. Inject all variables into the pod via `envFrom`

### Override dotenv files via CLI

You can override the template configuration with CLI flags:

```bash
# Use different dotenv files
kubectl kodama start my-session \
  --env-file .env.production \
  --sync .

# Exclude additional variables
kubectl kodama start my-session \
  --env-exclude EXTRA_VAR \
  --sync .
```

### Verify environment variables in the pod

```bash
# Attach to the session
kubectl kodama attach my-session

# Check loaded variables
env | grep DATABASE_URL
env | grep API_KEY
```

## How It Works

### Configuration Priority

Environment configuration follows Kodama's 3-tier priority:

1. **CLI flags** (highest priority)
   - `--env-file` overrides template dotenv files
   - `--env-exclude` adds to template exclusions

2. **Template config** (`.kodama.yaml`)
   - `env.dotenvFiles` specifies which files to load
   - `env.excludeVars` specifies variables to exclude

3. **Global config** (`~/.kodama/config.yaml`)
   - `defaults.env.excludeVars` adds global exclusions

### Variable Merging

When multiple dotenv files are specified, variables are merged with **last-wins** precedence:

```yaml
env:
  dotenvFiles:
    - .env          # Loaded first
    - .env.local    # Loaded second (overrides .env)
```

If both files define `DATABASE_URL`, the value from `.env.local` is used.

### Automatic Exclusions

System-critical variables are **always excluded** (cannot be overridden):

- System: `PATH`, `HOME`, `USER`, `SHELL`, `TERM`, `PWD`
- Kubernetes: `KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT`, etc.
- Claude: `CLAUDE_CODE_AUTH_TOKEN`, `CLAUDE_AUTH_FILE`, `ANTHROPIC_API_KEY`

### Security Features

- Dotenv files are read from your **local machine** (not from git)
- Warning displayed when loading `.env` files
- Variables must match pattern: `^[A-Z_][A-Z0-9_]*$`
- Total size limited to 1MB (Kubernetes secret limit)
- Secrets automatically cleaned up on session deletion

## Template Configuration Reference

```yaml
env:
  # Dotenv files to load (relative to repo root or absolute paths)
  dotenvFiles:
    - .env              # Base configuration
    - .env.local        # Local overrides
    - .env.production   # Production settings (example)

  # Additional variables to exclude (beyond system defaults)
  excludeVars:
    - VERBOSE           # Example: exclude debug flags
    - DEBUG_MODE        # Example: exclude local debugging
    - CI                # Example: exclude CI-specific vars
```

## Common Patterns

### Development vs Production

**Development** (`.kodama.yaml`):

```yaml
env:
  dotenvFiles:
    - .env
    - .env.local
  excludeVars:
    - CI
```

**Production** (`.kodama.production.yaml`):

```yaml
env:
  dotenvFiles:
    - .env.production
  excludeVars:
    - DEBUG_MODE
    - VERBOSE
```

Start with specific template:

```bash
kubectl kodama start prod-session \
  --config .kodama.production.yaml \
  --sync .
```

### Shared Team Configuration

**Global config** (`~/.kodama/config.yaml`):

```yaml
defaults:
  env:
    excludeVars:
      - LOCAL_CACHE_DIR  # Never inject local paths
```

**Project template** (`.kodama.yaml`):

```yaml
env:
  dotenvFiles:
    - .env
    - .env.${USER}  # Per-user overrides (e.g., .env.alice)
```

## Best Practices

1. **Never commit .env files**
   - Always add `.env*` to `.gitignore` (except `.env.example`)

2. **Use .env.example as documentation**
   - Commit `.env.example` with placeholder values
   - Team members copy to `.env` and fill in actual values

3. **Layer your environment files**
   - `.env` - Base/shared configuration
   - `.env.local` - Personal overrides
   - `.env.production` - Production settings

4. **Document excluded variables**
   - Add comments in `.kodama.yaml` explaining why variables are excluded

5. **Validate before committing**
   - Ensure `.env` files are excluded from git
   - Test with example files to verify configuration

## Troubleshooting

### Variables not appearing in pod

Check if variables are being excluded:

```bash
# Check default exclusions
kubectl kodama start test --env-file .env --sync . 2>&1 | grep "excluded"

# Verify variable names are valid (uppercase, alphanumeric, underscores)
# Invalid: my-var, 123VAR, var.name
# Valid: MY_VAR, VAR_123, _PRIVATE_VAR
```

### Secret size limit exceeded

If you have many large variables:

```bash
# Check .env file size
du -h .env

# Split into multiple smaller files
# Or exclude large variables and use ConfigMaps instead
```

### Dotenv file not found

Ensure paths are relative to repository root:

```yaml
env:
  dotenvFiles:
    - .env              # ✓ Correct: relative to repo root
    # - /abs/path/.env  # ✓ Also works: absolute path
    # - ../outside/.env # ✗ Avoid: paths outside repo
```
