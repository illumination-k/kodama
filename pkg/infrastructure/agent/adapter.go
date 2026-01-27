package agent

import (
	"context"

	"github.com/illumination-k/kodama/pkg/agent"
	"github.com/illumination-k/kodama/pkg/application/port"
)

// Adapter implements port.AgentExecutor using the existing agent.CodingAgentExecutor
type Adapter struct {
	executor agent.CodingAgentExecutor
}

// NewAdapter creates a new agent adapter
func NewAdapter() port.AgentExecutor {
	return &Adapter{
		executor: agent.NewCodingAgentExecutor(),
	}
}

// NewAdapterWithExecutor creates an adapter with a custom executor (for testing)
func NewAdapterWithExecutor(executor agent.CodingAgentExecutor) port.AgentExecutor {
	return &Adapter{
		executor: executor,
	}
}

// TaskStart initiates a new coding task with the given prompt
func (a *Adapter) TaskStart(ctx context.Context, namespace, podName, prompt string) (taskID string, err error) {
	return a.executor.TaskStart(ctx, namespace, podName, prompt)
}
