package git

import (
	"context"
	"fmt"
	"strings"
)

// simpleGitManager implements GitManager using kubectl exec
type simpleGitManager struct {
	executor CommandExecutor
}

// NewGitManager creates a GitManager instance with kubectl executor
func NewGitManager() GitManager {
	return &simpleGitManager{
		executor: NewKubectlExecutor(),
	}
}

// NewGitManagerWithExecutor creates a GitManager with custom executor (for testing)
func NewGitManagerWithExecutor(executor CommandExecutor) GitManager {
	return &simpleGitManager{
		executor: executor,
	}
}

// Clone clones a git repository into the pod's /workspace directory
func (s *simpleGitManager) Clone(ctx context.Context, namespace, podName, repoURL, branch, token string) error {
	// Ensure git is installed
	if err := s.ensureGitInstalled(ctx, namespace, podName); err != nil {
		return fmt.Errorf("failed to ensure git is installed: %w", err)
	}

	// Inject token into URL if HTTPS
	cloneURL := injectToken(repoURL, token)

	// Build git clone command
	args := []string{"git", "clone"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
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
