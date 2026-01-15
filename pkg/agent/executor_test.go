package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCodingAgentExecutor_TaskStart_Default(t *testing.T) {
	mock := NewMockCodingAgentExecutor()
	ctx := context.Background()

	taskID, err := mock.TaskStart(ctx, "test-ns", "test-pod", "test prompt")

	require.NoError(t, err)
	assert.Equal(t, "task-1", taskID)

	calls := mock.GetTaskStartCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test-ns", calls[0].Namespace)
	assert.Equal(t, "test-pod", calls[0].PodName)
	assert.Equal(t, "test prompt", calls[0].Prompt)
}

func TestMockCodingAgentExecutor_TaskStart_SequentialIDs(t *testing.T) {
	mock := NewMockCodingAgentExecutor()
	ctx := context.Background()

	taskID1, err := mock.TaskStart(ctx, "ns1", "pod1", "prompt1")
	require.NoError(t, err)
	assert.Equal(t, "task-1", taskID1)

	taskID2, err := mock.TaskStart(ctx, "ns2", "pod2", "prompt2")
	require.NoError(t, err)
	assert.Equal(t, "task-2", taskID2)

	calls := mock.GetTaskStartCalls()
	require.Len(t, calls, 2)
}

func TestMockCodingAgentExecutor_TaskStart_CustomFunc(t *testing.T) {
	mock := NewMockCodingAgentExecutor()
	mock.TaskStartFunc = func(ctx context.Context, namespace, podName, prompt string) (string, error) {
		return "custom-task-id", nil
	}

	ctx := context.Background()
	taskID, err := mock.TaskStart(ctx, "ns", "pod", "prompt")

	require.NoError(t, err)
	assert.Equal(t, "custom-task-id", taskID)
}

func TestMockCodingAgentExecutor_TaskStart_Error(t *testing.T) {
	mock := NewMockCodingAgentExecutor()
	mock.TaskStartFunc = func(ctx context.Context, namespace, podName, prompt string) (string, error) {
		return "", fmt.Errorf("simulated error")
	}

	ctx := context.Background()
	taskID, err := mock.TaskStart(ctx, "ns", "pod", "prompt")

	assert.Error(t, err)
	assert.Empty(t, taskID)
	assert.Contains(t, err.Error(), "simulated error")
}

func TestMockCodingAgentExecutor_Reset(t *testing.T) {
	mock := NewMockCodingAgentExecutor()

	_, _ = mock.TaskStart(context.Background(), "ns1", "pod1", "prompt1")
	_, _ = mock.TaskStart(context.Background(), "ns2", "pod2", "prompt2")

	require.Len(t, mock.GetTaskStartCalls(), 2)

	mock.Reset()

	assert.Len(t, mock.GetTaskStartCalls(), 0)
	assert.Equal(t, 1, mock.NextTaskID)
}

func TestMockCodingAgentExecutor_RecordsAllCalls(t *testing.T) {
	mock := NewMockCodingAgentExecutor()
	ctx := context.Background()

	_, _ = mock.TaskStart(ctx, "ns1", "pod1", "prompt1")
	_, _ = mock.TaskStart(ctx, "ns2", "pod2", "prompt2")
	_, _ = mock.TaskStart(ctx, "ns3", "pod3", "prompt3")

	calls := mock.GetTaskStartCalls()
	require.Len(t, calls, 3)

	assert.Equal(t, "ns1", calls[0].Namespace)
	assert.Equal(t, "pod1", calls[0].PodName)
	assert.Equal(t, "prompt1", calls[0].Prompt)

	assert.Equal(t, "ns2", calls[1].Namespace)
	assert.Equal(t, "pod2", calls[1].PodName)
	assert.Equal(t, "prompt2", calls[1].Prompt)

	assert.Equal(t, "ns3", calls[2].Namespace)
	assert.Equal(t, "pod3", calls[2].PodName)
	assert.Equal(t, "prompt3", calls[2].Prompt)
}
