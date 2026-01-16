package git

import "context"

// CloneOptions contains options for git clone operation
type CloneOptions struct {
	Branch       string // Branch to clone
	Depth        int    // Shallow clone depth (0 = full clone)
	SingleBranch bool   // Clone only single branch
	ExtraArgs    string // Additional arguments to git clone
}

// GitManager provides interface for git repository operations in pods
type GitManager interface {
	// Clone clones a repository into the pod's workspace directory (/workspace)
	// token is optional and used for HTTPS authentication with GitHub
	// For HTTPS URLs, token is injected as: https://<token>@github.com/user/repo.git
	// For SSH URLs, token is ignored and URL is passed through unchanged
	Clone(ctx context.Context, namespace, podName, repoURL, branch, token string) error

	// CloneWithOptions clones a repository with advanced options
	CloneWithOptions(ctx context.Context, namespace, podName, repoURL, token string, opts *CloneOptions) error

	// GetCurrentBranch returns the current git branch in the pod's workspace
	GetCurrentBranch(ctx context.Context, namespace, podName string) (string, error)

	// GetCurrentCommit returns the current commit hash in the pod's workspace
	GetCurrentCommit(ctx context.Context, namespace, podName string) (string, error)

	// BranchExists checks if a branch exists locally and/or remotely
	// Returns (localExists bool, remoteExists bool, error)
	BranchExists(ctx context.Context, namespace, podName, branchName string) (bool, bool, error)

	// CreateBranch creates a new local branch from current HEAD
	CreateBranch(ctx context.Context, namespace, podName, branchName string) error

	// CheckoutBranch checks out an existing branch (local or remote)
	CheckoutBranch(ctx context.Context, namespace, podName, branchName string) error
}
