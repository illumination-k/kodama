package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/illumination-k/kodama/pkg/agent"
)

// StartAgent initiates a coding agent task for this session
func (s *SessionConfig) StartAgent(ctx context.Context, executor agent.CodingAgentExecutor, prompt string) error {
	// Validation
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	if !s.IsRunning() {
		return fmt.Errorf("session must be running to start agent")
	}

	// Create execution record
	execution := AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     prompt,
		Status:     "running",
	}

	// Start task
	taskID, err := executor.TaskStart(ctx, s.Namespace, s.PodName, prompt)
	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		s.RecordAgentExecution(execution)
		return fmt.Errorf("failed to start agent task: %w", err)
	}

	execution.TaskID = taskID
	execution.Status = "completed" // For now, mark as completed immediately
	s.RecordAgentExecution(execution)

	return nil
}

// ReadPromptFromFile reads prompt content from a file
func ReadPromptFromFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	content, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file '%s': %w", filePath, err)
	}

	if len(content) == 0 {
		return "", fmt.Errorf("prompt file '%s' is empty", filePath)
	}

	return string(content), nil
}

// truncatePrompt truncates a prompt for display purposes
func truncatePrompt(prompt string, maxLen int) string {
	if len(prompt) <= maxLen {
		return prompt
	}
	return prompt[:maxLen] + "..."
}
