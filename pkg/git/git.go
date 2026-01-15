package git

import "context"

// GitManager provides interface for git repository operations in pods
type GitManager interface {
	// Clone clones a repository into the pod's workspace directory (/workspace)
	// token is optional and used for HTTPS authentication with GitHub
	// For HTTPS URLs, token is injected as: https://<token>@github.com/user/repo.git
	// For SSH URLs, token is ignored and URL is passed through unchanged
	Clone(ctx context.Context, namespace, podName, repoURL, branch, token string) error

	// GetCurrentBranch returns the current git branch in the pod's workspace
	GetCurrentBranch(ctx context.Context, namespace, podName string) (string, error)

	// GetCurrentCommit returns the current commit hash in the pod's workspace
	GetCurrentCommit(ctx context.Context, namespace, podName string) (string, error)
}
