package task

import (
	"context"
	"fmt"
	"os"

	"github.com/illumination-k/kodama/pkg/agent"
)

// Start initiates a coding task with the given prompt
func Start(ctx context.Context, executor agent.CodingAgentExecutor, namespace, podName, prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	fmt.Printf("Starting coding agent task...\n")
	fmt.Printf("Prompt: %s\n", truncatePrompt(prompt, 100))

	taskID, err := executor.TaskStart(ctx, namespace, podName, prompt)
	if err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	fmt.Printf("Task started: %s\n", taskID)
	return nil
}

// ReadPromptFromFile reads prompt content from a file
func ReadPromptFromFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	content, err := os.ReadFile(filePath)
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
