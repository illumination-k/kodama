package port

import (
	"context"
)

// AgentExecutor abstracts coding agent operations for testing
type AgentExecutor interface {
	// TaskStart initiates a new coding task with the given prompt
	// Returns task ID and error
	TaskStart(ctx context.Context, namespace, podName, prompt string) (taskID string, err error)

	// Additional methods for future expansion:
	// TaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)
	// TaskStop(ctx context.Context, taskID string) error
}

// TaskStatus represents the status of a coding agent task
type TaskStatus struct {
	TaskID   string
	Status   string // "pending", "running", "completed", "failed"
	Progress string
	Error    string
}
