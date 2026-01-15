package git

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectToken(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		token    string
		expected string
	}{
		{
			name:     "HTTPS URL with token",
			repoURL:  "https://github.com/user/repo.git",
			token:    "ghp_test123",
			expected: "https://ghp_test123@github.com/user/repo.git",
		},
		{
			name:     "HTTPS URL without token",
			repoURL:  "https://github.com/user/repo.git",
			token:    "",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "SSH URL with token (ignored)",
			repoURL:  "git@github.com:user/repo.git",
			token:    "ghp_test123",
			expected: "git@github.com:user/repo.git",
		},
		{
			name:     "HTTP URL with token",
			repoURL:  "http://github.com/user/repo.git",
			token:    "ghp_test123",
			expected: "http://github.com/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectToken(tt.repoURL, tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		token    string
		expected string
	}{
		{
			name:     "error with token",
			errMsg:   "fatal: could not read from remote repository using token ghp_test123",
			token:    "ghp_test123",
			expected: "fatal: could not read from remote repository using token ***",
		},
		{
			name:     "error without token",
			errMsg:   "fatal: repository not found",
			token:    "ghp_test123",
			expected: "fatal: repository not found",
		},
		{
			name:     "empty token",
			errMsg:   "fatal: could not read from remote repository using token ghp_test123",
			token:    "",
			expected: "fatal: could not read from remote repository using token ghp_test123",
		},
		{
			name:     "multiple token occurrences",
			errMsg:   "token ghp_test123 is invalid, please check ghp_test123",
			token:    "ghp_test123",
			expected: "token *** is invalid, please check ***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.errMsg, tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitManager_Clone_Success(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	namespace := "test-ns"
	podName := "test-pod"
	repoURL := "https://github.com/test/repo.git"
	branch := ""
	token := ""

	// Configure mock responses
	mockExec.SetResponse("which git", "", "", nil) // git is installed
	mockExec.SetResponse("git clone", "", "", nil)

	// Execute
	err := gitMgr.Clone(ctx, namespace, podName, repoURL, branch, token)

	// Assertions
	require.NoError(t, err)

	commands := mockExec.GetCommands()
	require.Len(t, commands, 2) // which git + git clone
	assert.Equal(t, namespace, commands[1].Namespace)
	assert.Equal(t, podName, commands[1].PodName)

	cmdStr := strings.Join(commands[1].Command, " ")
	assert.Contains(t, cmdStr, "git clone")
	assert.Contains(t, cmdStr, repoURL)
	assert.Contains(t, cmdStr, "/workspace")
}

func TestGitManager_Clone_WithBranch(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branch := "develop"

	mockExec.SetResponse("which git", "", "", nil)
	mockExec.SetResponse("git clone", "", "", nil)

	err := gitMgr.Clone(ctx, "ns", "pod", "https://github.com/test/repo.git", branch, "")

	require.NoError(t, err)

	commands := mockExec.GetCommands()
	cmdStr := strings.Join(commands[1].Command, " ")
	assert.Contains(t, cmdStr, "--branch develop")
}

func TestGitManager_Clone_WithToken(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	token := "ghp_test123"
	repoURL := "https://github.com/test/repo.git"

	mockExec.SetResponse("which git", "", "", nil)
	mockExec.SetResponse("git clone", "", "", nil)

	err := gitMgr.Clone(ctx, "ns", "pod", repoURL, "", token)

	require.NoError(t, err)

	commands := mockExec.GetCommands()
	cmdStr := strings.Join(commands[1].Command, " ")
	// Token should be injected in URL
	assert.Contains(t, cmdStr, "https://ghp_test123@github.com/test/repo.git")
}

func TestGitManager_Clone_WithSSH(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	token := "ghp_test123"
	sshURL := "git@github.com:test/repo.git"

	mockExec.SetResponse("which git", "", "", nil)
	mockExec.SetResponse("git clone", "", "", nil)

	err := gitMgr.Clone(ctx, "ns", "pod", sshURL, "", token)

	require.NoError(t, err)

	commands := mockExec.GetCommands()
	cmdStr := strings.Join(commands[1].Command, " ")
	// SSH URL should be unchanged (token not injected)
	assert.Contains(t, cmdStr, sshURL)
	assert.NotContains(t, cmdStr, token)
}

func TestGitManager_Clone_Failure(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()

	mockExec.SetResponse("git clone", "", "fatal: repository not found", fmt.Errorf("exit code 128"))

	err := gitMgr.Clone(ctx, "ns", "pod", "https://github.com/invalid/repo.git", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
	assert.Contains(t, err.Error(), "repository not found")
}

func TestGitManager_Clone_FailureWithTokenSanitization(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	token := "ghp_secret123"

	mockExec.SetResponse("git clone", "", "fatal: authentication failed for 'https://ghp_secret123@github.com/test/repo.git'", fmt.Errorf("exit code 128"))

	err := gitMgr.Clone(ctx, "ns", "pod", "https://github.com/test/repo.git", "", token)

	assert.Error(t, err)
	// Token should be sanitized in error message
	assert.NotContains(t, err.Error(), token)
	assert.Contains(t, err.Error(), "***")
}

func TestGitManager_GetCurrentBranch(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()

	mockExec.SetResponse("git -C /workspace branch --show-current", "main\n", "", nil)

	branch, err := gitMgr.GetCurrentBranch(ctx, "ns", "pod")

	require.NoError(t, err)
	assert.Equal(t, "main", branch)

	commands := mockExec.GetCommands()
	cmdStr := strings.Join(commands[0].Command, " ")
	assert.Contains(t, cmdStr, "git -C /workspace branch --show-current")
}

func TestGitManager_GetCurrentCommit(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	commitHash := "abc123def456"

	mockExec.SetResponse("git -C /workspace rev-parse HEAD", commitHash+"\n", "", nil)

	commit, err := gitMgr.GetCurrentCommit(ctx, "ns", "pod")

	require.NoError(t, err)
	assert.Equal(t, commitHash, commit)

	commands := mockExec.GetCommands()
	cmdStr := strings.Join(commands[0].Command, " ")
	assert.Contains(t, cmdStr, "git -C /workspace rev-parse HEAD")
}

func TestGitManager_GetCurrentBranch_Failure(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()

	mockExec.SetResponse("git -C /workspace branch --show-current", "", "fatal: not a git repository", fmt.Errorf("exit code 128"))

	branch, err := gitMgr.GetCurrentBranch(ctx, "ns", "pod")

	assert.Error(t, err)
	assert.Empty(t, branch)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGitManager_GetCurrentCommit_Failure(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()

	mockExec.SetResponse("git -C /workspace rev-parse HEAD", "", "fatal: not a git repository", fmt.Errorf("exit code 128"))

	commit, err := gitMgr.GetCurrentCommit(ctx, "ns", "pod")

	assert.Error(t, err)
	assert.Empty(t, commit)
	assert.Contains(t, err.Error(), "not a git repository")
}
