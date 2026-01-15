package kubernetes

import (
	"context"
	"strings"
)

// MockExecutor is a mock implementation of CommandExecutor for testing
type MockExecutor struct {
	Responses map[string]MockResponse
	Commands  []MockCommand
}

// MockCommand represents a recorded command execution
type MockCommand struct {
	Namespace string
	PodName   string
	Command   []string
}

// MockResponse represents a mock response for a command
type MockResponse struct {
	Error  error
	Stdout string
	Stderr string
}

// NewMockExecutor creates a new MockExecutor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Commands:  []MockCommand{},
		Responses: make(map[string]MockResponse),
	}
}

// ExecInPod records the command and returns a pre-configured response
func (m *MockExecutor) ExecInPod(ctx context.Context, namespace, podName string, command []string) (string, string, error) {
	// Record the command
	m.Commands = append(m.Commands, MockCommand{
		Namespace: namespace,
		PodName:   podName,
		Command:   command,
	})

	// Find matching response
	cmdStr := strings.Join(command, " ")
	for prefix, response := range m.Responses {
		if strings.HasPrefix(cmdStr, prefix) {
			return response.Stdout, response.Stderr, response.Error
		}
	}

	// Default success
	return "", "", nil
}

// SetResponse configures a mock response for commands starting with prefix
func (m *MockExecutor) SetResponse(prefix, stdout, stderr string, err error) {
	m.Responses[prefix] = MockResponse{
		Stdout: stdout,
		Stderr: stderr,
		Error:  err,
	}
}

// GetCommands returns all executed commands (for assertions)
func (m *MockExecutor) GetCommands() []MockCommand {
	return m.Commands
}

// Reset clears all recorded commands and responses
func (m *MockExecutor) Reset() {
	m.Commands = []MockCommand{}
	m.Responses = make(map[string]MockResponse)
}
