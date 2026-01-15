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

func TestIsMainBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{
			name:   "main branch",
			branch: "main",
			want:   true,
		},
		{
			name:   "master branch",
			branch: "master",
			want:   true,
		},
		{
			name:   "trunk branch",
			branch: "trunk",
			want:   true,
		},
		{
			name:   "development branch",
			branch: "development",
			want:   true,
		},
		{
			name:   "main with uppercase",
			branch: "Main",
			want:   true,
		},
		{
			name:   "master with uppercase",
			branch: "MASTER",
			want:   true,
		},
		{
			name:   "main with whitespace",
			branch: "  main  ",
			want:   true,
		},
		{
			name:   "feature branch",
			branch: "feature/test",
			want:   false,
		},
		{
			name:   "kodama branch",
			branch: "kodama/test-123",
			want:   false,
		},
		{
			name:   "develop branch",
			branch: "develop",
			want:   false,
		},
		{
			name:   "empty string",
			branch: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMainBranch(tt.branch)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		sessionName string
		wantPrefix  string
		wantSuffix  bool
	}{
		{
			name:        "standard case with prefix",
			prefix:      "kodama/",
			sessionName: "my-work",
			wantPrefix:  "kodama/my-work-",
			wantSuffix:  true,
		},
		{
			name:        "prefix without trailing slash",
			prefix:      "kodama",
			sessionName: "test",
			wantPrefix:  "kodama/test-",
			wantSuffix:  true,
		},
		{
			name:        "session name with spaces",
			prefix:      "kodama/",
			sessionName: "my work session",
			wantPrefix:  "kodama/my-work-session-",
			wantSuffix:  true,
		},
		{
			name:        "session name with underscores",
			prefix:      "kodama/",
			sessionName: "my_work_session",
			wantPrefix:  "kodama/my-work-session-",
			wantSuffix:  true,
		},
		{
			name:        "session name with uppercase",
			prefix:      "kodama/",
			sessionName: "MyWork",
			wantPrefix:  "kodama/mywork-",
			wantSuffix:  true,
		},
		{
			name:        "empty prefix",
			prefix:      "",
			sessionName: "test",
			wantPrefix:  "test-",
			wantSuffix:  true,
		},
		{
			name:        "session name with special chars",
			prefix:      "kodama/",
			sessionName: "test@123!",
			wantPrefix:  "kodama/test123-",
			wantSuffix:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateBranchName(tt.prefix, tt.sessionName)

			// Check prefix
			assert.True(t, strings.HasPrefix(got, tt.wantPrefix), "expected prefix '%s', got '%s'", tt.wantPrefix, got)

			// Check timestamp suffix (14 digits)
			if tt.wantSuffix {
				parts := strings.Split(got, "-")
				lastPart := parts[len(parts)-1]
				assert.Len(t, lastPart, 14, "timestamp should be 14 digits")
				assert.Regexp(t, `^\d{14}$`, lastPart, "timestamp should be all digits")
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "test",
			expected: "test",
		},
		{
			name:     "name with spaces",
			input:    "my work",
			expected: "my-work",
		},
		{
			name:     "name with underscores",
			input:    "my_work",
			expected: "my-work",
		},
		{
			name:     "name with uppercase",
			input:    "MyWork",
			expected: "mywork",
		},
		{
			name:     "name with invalid git chars",
			input:    "test~^:?*[",
			expected: "test",
		},
		{
			name:     "name with special chars",
			input:    "test@123!",
			expected: "test123",
		},
		{
			name:     "name with leading/trailing hyphens",
			input:    "-test-",
			expected: "test",
		},
		{
			name:     "mixed case with spaces and special chars",
			input:    "My Test Work!",
			expected: "my-test-work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBranchName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGitManager_BranchExists_BothExist(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/test"

	// Configure mock: branch exists both locally and remotely
	mockExec.SetResponse("git -C /workspace branch --list", "  feature/test\n", "", nil)
	mockExec.SetResponse("git -C /workspace ls-remote", "abc123def456 refs/heads/feature/test\n", "", nil)

	localExists, remoteExists, err := gitMgr.BranchExists(ctx, "ns", "pod", branchName)

	require.NoError(t, err)
	assert.True(t, localExists)
	assert.True(t, remoteExists)
}

func TestGitManager_BranchExists_OnlyLocal(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/local"

	// Branch exists locally but not remotely
	mockExec.SetResponse("git -C /workspace branch --list", "  feature/local\n", "", nil)
	mockExec.SetResponse("git -C /workspace ls-remote", "", "", nil)

	localExists, remoteExists, err := gitMgr.BranchExists(ctx, "ns", "pod", branchName)

	require.NoError(t, err)
	assert.True(t, localExists)
	assert.False(t, remoteExists)
}

func TestGitManager_BranchExists_OnlyRemote(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/remote"

	// Branch exists remotely but not locally
	mockExec.SetResponse("git -C /workspace branch --list", "", "", nil)
	mockExec.SetResponse("git -C /workspace ls-remote", "abc123 refs/heads/feature/remote\n", "", nil)

	localExists, remoteExists, err := gitMgr.BranchExists(ctx, "ns", "pod", branchName)

	require.NoError(t, err)
	assert.False(t, localExists)
	assert.True(t, remoteExists)
}

func TestGitManager_BranchExists_Neither(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/nonexistent"

	// Branch doesn't exist anywhere
	mockExec.SetResponse("git -C /workspace branch --list", "", "", nil)
	mockExec.SetResponse("git -C /workspace ls-remote", "", "", nil)

	localExists, remoteExists, err := gitMgr.BranchExists(ctx, "ns", "pod", branchName)

	require.NoError(t, err)
	assert.False(t, localExists)
	assert.False(t, remoteExists)
}

func TestGitManager_BranchExists_NetworkError(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/test"

	// Local check succeeds, remote check fails (network error)
	mockExec.SetResponse("git -C /workspace branch --list", "  feature/test\n", "", nil)
	mockExec.SetResponse("git -C /workspace ls-remote", "", "fatal: unable to access remote", fmt.Errorf("network error"))

	localExists, remoteExists, err := gitMgr.BranchExists(ctx, "ns", "pod", branchName)

	// Network errors are non-fatal
	require.NoError(t, err)
	assert.True(t, localExists)
	assert.False(t, remoteExists)
}

func TestGitManager_CreateBranch_Success(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/new-branch"

	mockExec.SetResponse("git -C /workspace checkout -b", "", "", nil)

	err := gitMgr.CreateBranch(ctx, "ns", "pod", branchName)

	require.NoError(t, err)

	commands := mockExec.GetCommands()
	require.Len(t, commands, 1)
	cmdStr := strings.Join(commands[0].Command, " ")
	assert.Contains(t, cmdStr, "git -C /workspace checkout -b")
	assert.Contains(t, cmdStr, branchName)
}

func TestGitManager_CreateBranch_Failure(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/existing"

	mockExec.SetResponse("git -C /workspace checkout -b", "", "fatal: A branch named 'feature/existing' already exists", fmt.Errorf("exit code 128"))

	err := gitMgr.CreateBranch(ctx, "ns", "pod", branchName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create branch")
	assert.Contains(t, err.Error(), "already exists")
}

func TestGitManager_CheckoutBranch_Success(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/existing"

	mockExec.SetResponse("git -C /workspace checkout", "", "", nil)

	err := gitMgr.CheckoutBranch(ctx, "ns", "pod", branchName)

	require.NoError(t, err)

	commands := mockExec.GetCommands()
	require.Len(t, commands, 1)
	cmdStr := strings.Join(commands[0].Command, " ")
	assert.Contains(t, cmdStr, "git -C /workspace checkout")
	assert.Contains(t, cmdStr, branchName)
}

func TestGitManager_CheckoutBranch_Failure(t *testing.T) {
	mockExec := NewMockExecutor()
	gitMgr := NewGitManagerWithExecutor(mockExec)

	ctx := context.Background()
	branchName := "feature/nonexistent"

	mockExec.SetResponse("git -C /workspace checkout", "", "error: pathspec 'feature/nonexistent' did not match any file(s) known to git", fmt.Errorf("exit code 1"))

	err := gitMgr.CheckoutBranch(ctx, "ns", "pod", branchName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to checkout branch")
	assert.Contains(t, err.Error(), "did not match")
}
