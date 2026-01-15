package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/illumination-k/kodama/pkg/git"
)

// realCodingAgentExecutor implements CodingAgentExecutor using kubectl exec
type realCodingAgentExecutor struct {
	commandExecutor git.CommandExecutor
}

// NewCodingAgentExecutor creates a new real coding agent executor
func NewCodingAgentExecutor() CodingAgentExecutor {
	return &realCodingAgentExecutor{
		commandExecutor: git.NewKubectlExecutor(),
	}
}

// NewCodingAgentExecutorWithCommandExecutor creates executor with custom command executor
// This is useful for testing and dependency injection
func NewCodingAgentExecutorWithCommandExecutor(cmdExec git.CommandExecutor) CodingAgentExecutor {
	return &realCodingAgentExecutor{
		commandExecutor: cmdExec,
	}
}

// TaskStart initiates a coding task in the pod
func (r *realCodingAgentExecutor) TaskStart(ctx context.Context, namespace, podName, prompt string) (string, error) {
	// For now, this is a placeholder that echoes the prompt
	// Future implementation will invoke actual claude-code agent

	// Escape single quotes in prompt for shell safety
	escapedPrompt := strings.ReplaceAll(prompt, "'", "'\\''")

	command := []string{
		"sh", "-c",
		fmt.Sprintf("echo 'Task started with prompt: %s' && echo 'task-placeholder-id'", escapedPrompt),
	}

	stdout, stderr, err := r.commandExecutor.ExecInPod(ctx, namespace, podName, command)
	if err != nil {
		return "", fmt.Errorf("failed to start task: %s: %w", stderr, err)
	}

	// Parse task ID from stdout (in real implementation)
	// For now, return placeholder from last line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) > 0 {
		taskID := strings.TrimSpace(lines[len(lines)-1])
		return taskID, nil
	}

	return "task-placeholder-id", nil
}
