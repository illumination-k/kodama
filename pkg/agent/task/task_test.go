package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/illumination-k/kodama/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart_Success(t *testing.T) {
	mock := agent.NewMockCodingAgentExecutor()
	ctx := context.Background()

	err := Start(ctx, mock, "test-ns", "test-pod", "test prompt")

	require.NoError(t, err)

	calls := mock.GetTaskStartCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test-ns", calls[0].Namespace)
	assert.Equal(t, "test-pod", calls[0].PodName)
	assert.Equal(t, "test prompt", calls[0].Prompt)
}

func TestStart_EmptyPrompt(t *testing.T) {
	mock := agent.NewMockCodingAgentExecutor()
	ctx := context.Background()

	err := Start(ctx, mock, "test-ns", "test-pod", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt cannot be empty")
}

func TestStart_ExecutorError(t *testing.T) {
	mock := agent.NewMockCodingAgentExecutor()
	mock.TaskStartFunc = func(ctx context.Context, namespace, podName, prompt string) (string, error) {
		return "", fmt.Errorf("executor failed")
	}

	ctx := context.Background()
	err := Start(ctx, mock, "test-ns", "test-pod", "test prompt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start task")
	assert.Contains(t, err.Error(), "executor failed")
}

func TestReadPromptFromFile_Success(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "prompt.txt")
	content := "This is a test prompt from file"

	err := os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(t, err)

	// Read prompt
	prompt, err := ReadPromptFromFile(filePath)

	require.NoError(t, err)
	assert.Equal(t, content, prompt)
}

func TestReadPromptFromFile_FileNotFound(t *testing.T) {
	prompt, err := ReadPromptFromFile("/nonexistent/file.txt")

	assert.Error(t, err)
	assert.Empty(t, prompt)
	assert.Contains(t, err.Error(), "failed to read prompt file")
}

func TestReadPromptFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(filePath, []byte(""), 0o644)
	require.NoError(t, err)

	prompt, err := ReadPromptFromFile(filePath)

	assert.Error(t, err)
	assert.Empty(t, prompt)
	assert.Contains(t, err.Error(), "is empty")
}

func TestReadPromptFromFile_EmptyPath(t *testing.T) {
	prompt, err := ReadPromptFromFile("")

	assert.Error(t, err)
	assert.Empty(t, prompt)
	assert.Contains(t, err.Error(), "file path cannot be empty")
}

func TestTruncatePrompt(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		maxLen   int
		expected string
	}{
		{
			name:     "short prompt",
			prompt:   "short",
			maxLen:   100,
			expected: "short",
		},
		{
			name:     "exact length",
			prompt:   "exactly100",
			maxLen:   10,
			expected: "exactly100",
		},
		{
			name:     "long prompt",
			prompt:   "This is a very long prompt that exceeds the maximum length",
			maxLen:   20,
			expected: "This is a very long ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncatePrompt(tt.prompt, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadPromptFromFile_MultilineContent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "multiline.txt")
	content := "Line 1\nLine 2\nLine 3"

	err := os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(t, err)

	prompt, err := ReadPromptFromFile(filePath)

	require.NoError(t, err)
	assert.Equal(t, content, prompt)
	assert.Contains(t, prompt, "\n")
}
