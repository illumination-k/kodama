package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/illumination-k/kodama/pkg/kubernetes"
)

// simpleGitManager implements GitManager using kubectl exec
type simpleGitManager struct {
	executor kubernetes.CommandExecutor
}

// NewGitManager creates a GitManager instance with kubectl executor
func NewGitManager() GitManager {
	return &simpleGitManager{
		executor: kubernetes.NewKubectlExecutor(),
	}
}

// NewGitManagerWithExecutor creates a GitManager with custom executor (for testing)
func NewGitManagerWithExecutor(executor kubernetes.CommandExecutor) GitManager {
	return &simpleGitManager{
		executor: executor,
	}
}

// Clone clones a git repository into the pod's /workspace directory
func (s *simpleGitManager) Clone(ctx context.Context, namespace, podName, repoURL, branch, token string) error {
	// Use CloneWithOptions with basic options
	opts := &CloneOptions{
		Branch: branch,
	}
	return s.CloneWithOptions(ctx, namespace, podName, repoURL, token, opts)
}

// CloneWithOptions clones a git repository with advanced options
func (s *simpleGitManager) CloneWithOptions(ctx context.Context, namespace, podName, repoURL, token string, opts *CloneOptions) error {
	// Ensure git is installed
	if err := s.ensureGitInstalled(ctx, namespace, podName); err != nil {
		return fmt.Errorf("failed to ensure git is installed: %w", err)
	}

	// Inject token into URL if HTTPS
	cloneURL := injectToken(repoURL, token)

	// Build git clone command
	args := []string{"git", "clone"}

	// Add depth option for shallow clone
	if opts != nil && opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}

	// Add single-branch option
	if opts != nil && opts.SingleBranch {
		args = append(args, "--single-branch")
	}

	// Add branch option
	if opts != nil && opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	// Add extra arguments (parse from string)
	if opts != nil && opts.ExtraArgs != "" {
		// Validate args for safety
		if err := ValidateCloneArgs(opts.ExtraArgs); err != nil {
			return fmt.Errorf("invalid git clone arguments: %w", err)
		}

		// Parse extra args (simple split by space)
		extraArgs := strings.Fields(opts.ExtraArgs)
		args = append(args, extraArgs...)
	}

	// Add repository URL and target directory
	args = append(args, cloneURL, "/workspace")

	// Execute in pod
	stdout, stderr, err := s.executor.ExecInPod(ctx, namespace, podName, args)
	if err != nil {
		// Sanitize token from error message
		sanitizedErr := sanitizeError(stderr, token)
		if sanitizedErr == "" {
			sanitizedErr = sanitizeError(stdout, token)
		}
		return fmt.Errorf("git clone failed: %s: %w", sanitizedErr, err)
	}

	return nil
}

// ValidateCloneArgs performs basic validation on extra git clone arguments
// to prevent command injection or dangerous options
func ValidateCloneArgs(args string) error {
	if args == "" {
		return nil
	}

	// Disallow dangerous patterns
	dangerousPatterns := []string{
		"|", "&&", "||", ";", "`", "$(", // Command injection
		"--upload-pack", "--config", // Potentially dangerous git options
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(args, pattern) {
			return fmt.Errorf("git clone args contain disallowed pattern: %s", pattern)
		}
	}

	return nil
}

// ensureGitInstalled checks if git is installed and installs it if necessary
func (s *simpleGitManager) ensureGitInstalled(ctx context.Context, namespace, podName string) error {
	// Check if git is already installed
	_, _, err := s.executor.ExecInPod(ctx, namespace, podName, []string{"which", "git"})
	if err == nil {
		// git is already installed
		return nil
	}

	// Try to install git using apt-get (for Debian/Ubuntu-based images)
	_, _, err = s.executor.ExecInPod(ctx, namespace, podName, []string{"sh", "-c", "apt-get update -qq && apt-get install -y -qq git > /dev/null 2>&1"})
	if err == nil {
		return nil
	}

	// Try to install git using apk (for Alpine-based images)
	_, _, err = s.executor.ExecInPod(ctx, namespace, podName, []string{"apk", "add", "--no-cache", "git"})
	if err == nil {
		return nil
	}

	// Try to install git using yum (for RHEL/CentOS-based images)
	_, _, err = s.executor.ExecInPod(ctx, namespace, podName, []string{"yum", "install", "-y", "git"})
	if err == nil {
		return nil
	}

	return fmt.Errorf("git is not installed and could not be installed automatically. Please use a container image with git pre-installed")
}

// GetCurrentBranch returns the current git branch in the pod's workspace
func (s *simpleGitManager) GetCurrentBranch(ctx context.Context, namespace, podName string) (string, error) {
	args := []string{"git", "-C", "/workspace", "branch", "--show-current"}

	stdout, stderr, err := s.executor.ExecInPod(ctx, namespace, podName, args)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %s: %w", stderr, err)
	}

	return strings.TrimSpace(stdout), nil
}

// GetCurrentCommit returns the current commit hash in the pod's workspace
func (s *simpleGitManager) GetCurrentCommit(ctx context.Context, namespace, podName string) (string, error) {
	args := []string{"git", "-C", "/workspace", "rev-parse", "HEAD"}

	stdout, stderr, err := s.executor.ExecInPod(ctx, namespace, podName, args)
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %s: %w", stderr, err)
	}

	return strings.TrimSpace(stdout), nil
}

// BranchExists checks if a branch exists locally and/or remotely
// Returns (localExists bool, remoteExists bool, error)
func (s *simpleGitManager) BranchExists(ctx context.Context, namespace, podName, branchName string) (bool, bool, error) {
	// Check local branch
	localArgs := []string{"git", "-C", "/workspace", "branch", "--list", branchName}
	localStdout, _, localErr := s.executor.ExecInPod(ctx, namespace, podName, localArgs)

	localExists := localErr == nil && strings.TrimSpace(localStdout) != ""

	// Check remote branch
	// Note: Network errors are non-fatal, we just return false for remote
	remoteArgs := []string{"git", "-C", "/workspace", "ls-remote", "--heads", "origin", branchName}
	remoteStdout, _, remoteErr := s.executor.ExecInPod(ctx, namespace, podName, remoteArgs)

	remoteExists := false
	if remoteErr == nil && strings.TrimSpace(remoteStdout) != "" {
		// Parse output: "abc123def456 refs/heads/branchName"
		// If output contains refs/heads/, the branch exists
		if strings.Contains(remoteStdout, "refs/heads/") {
			remoteExists = true
		}
	}

	return localExists, remoteExists, nil
}

// CreateBranch creates a new local branch from current HEAD
func (s *simpleGitManager) CreateBranch(ctx context.Context, namespace, podName, branchName string) error {
	args := []string{"git", "-C", "/workspace", "checkout", "-b", branchName}

	_, stderr, err := s.executor.ExecInPod(ctx, namespace, podName, args)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s': %s: %w", branchName, stderr, err)
	}

	return nil
}

// CheckoutBranch checks out an existing branch (local or remote)
func (s *simpleGitManager) CheckoutBranch(ctx context.Context, namespace, podName, branchName string) error {
	// First try to checkout the branch directly
	args := []string{"git", "-C", "/workspace", "checkout", branchName}

	_, stderr, err := s.executor.ExecInPod(ctx, namespace, podName, args)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %s: %w", branchName, stderr, err)
	}

	return nil
}

// injectToken adds GitHub token to HTTPS URL
// For HTTPS URLs: https://github.com/user/repo.git -> https://<token>@github.com/user/repo.git
// For SSH URLs: git@github.com:user/repo.git -> unchanged
func injectToken(repoURL, token string) string {
	if token == "" || !strings.HasPrefix(repoURL, "https://") {
		return repoURL
	}
	return strings.Replace(repoURL, "https://", "https://"+token+"@", 1)
}

// sanitizeError removes token from error messages to prevent leaks
func sanitizeError(errMsg, token string) string {
	if token == "" {
		return errMsg
	}
	return strings.ReplaceAll(errMsg, token, "***")
}

// IsMainBranch checks if the given branch name is a protected main branch
// Protected branches: main, master, trunk, development (case-insensitive)
func IsMainBranch(branchName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(branchName))
	mainBranches := []string{"main", "master", "trunk", "development"}

	for _, main := range mainBranches {
		if normalized == main {
			return true
		}
	}
	return false
}

// GenerateBranchName creates a timestamped branch name
// Format: {prefix}{sanitized-session-name}-{YYYYMMDDHHmmss}
// Example: kodama/my-work-20250115143000
func GenerateBranchName(prefix, sessionName string) string {
	// Sanitize session name for git branch compatibility
	sanitized := sanitizeBranchName(sessionName)

	// Generate timestamp
	timestamp := time.Now().Format("20060102150405")

	// Combine prefix, session name, and timestamp
	if prefix == "" {
		return fmt.Sprintf("%s-%s", sanitized, timestamp)
	}

	// Ensure prefix ends with / if it doesn't already
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return fmt.Sprintf("%s%s-%s", prefix, sanitized, timestamp)
}

// sanitizeBranchName sanitizes a session name for use in git branch names
// - Converts to lowercase
// - Replaces spaces and underscores with hyphens
// - Removes invalid git characters
func sanitizeBranchName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remove invalid git characters
	// Git branch names cannot contain: ~, ^, :, ?, *, [, \, .., @{
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\", "..", "@{", "@"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "")
	}

	// Remove any remaining problematic characters using regex
	// Keep only alphanumeric, hyphens, and forward slashes
	reg := regexp.MustCompile(`[^a-z0-9\-/]+`)
	name = reg.ReplaceAllString(name, "")

	// Remove leading/trailing hyphens and slashes
	name = strings.Trim(name, "-/")

	return name
}
