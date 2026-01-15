package agent

import (
	"context"
	"fmt"
)

// MockCodingAgentExecutor is a mock implementation for testing
type MockCodingAgentExecutor struct {
	TaskStartFunc  func(ctx context.Context, namespace, podName, prompt string) (string, error)
	TaskStartCalls []TaskStartCall
	NextTaskID     int
}

// TaskStartCall records a call to TaskStart
type TaskStartCall struct {
	Namespace string
	PodName   string
	Prompt    string
}

// NewMockCodingAgentExecutor creates a new mock executor
func NewMockCodingAgentExecutor() *MockCodingAgentExecutor {
	return &MockCodingAgentExecutor{
		TaskStartCalls: []TaskStartCall{},
		NextTaskID:     1,
	}
}

// TaskStart records the call and returns a mock task ID
func (m *MockCodingAgentExecutor) TaskStart(ctx context.Context, namespace, podName, prompt string) (string, error) {
	// Record the call
	m.TaskStartCalls = append(m.TaskStartCalls, TaskStartCall{
		Namespace: namespace,
		PodName:   podName,
		Prompt:    prompt,
	})

	// Use custom function if provided
	if m.TaskStartFunc != nil {
		return m.TaskStartFunc(ctx, namespace, podName, prompt)
	}

	// Default behavior: return sequential task IDs
	taskID := fmt.Sprintf("task-%d", m.NextTaskID)
	m.NextTaskID++
	return taskID, nil
}

// GetTaskStartCalls returns all recorded calls (for test assertions)
func (m *MockCodingAgentExecutor) GetTaskStartCalls() []TaskStartCall {
	return m.TaskStartCalls
}

// Reset clears all recorded calls
func (m *MockCodingAgentExecutor) Reset() {
	m.TaskStartCalls = []TaskStartCall{}
	m.NextTaskID = 1
	m.TaskStartFunc = nil
}
