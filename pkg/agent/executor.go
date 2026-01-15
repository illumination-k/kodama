package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/illumination-k/kodama/pkg/agent/auth"
	"github.com/illumination-k/kodama/pkg/git"
)

// realCodingAgentExecutor implements CodingAgentExecutor using kubectl exec
type realCodingAgentExecutor struct {
	commandExecutor git.CommandExecutor
	authProvider    auth.AuthProvider
	sanitizer       *auth.Sanitizer
}

// NewCodingAgentExecutor creates a new real coding agent executor
// Uses default authentication provider from environment
func NewCodingAgentExecutor() CodingAgentExecutor {
	authProvider, _ := auth.GetDefaultAuthProvider() // Ignore error, auth is optional
	return &realCodingAgentExecutor{
		commandExecutor: git.NewKubectlExecutor(),
		authProvider:    authProvider,
		sanitizer:       auth.NewSanitizer(),
	}
}

// NewCodingAgentExecutorWithAuth creates executor with specified auth provider
func NewCodingAgentExecutorWithAuth(authProvider auth.AuthProvider) CodingAgentExecutor {
	return &realCodingAgentExecutor{
		commandExecutor: git.NewKubectlExecutor(),
		authProvider:    authProvider,
		sanitizer:       auth.NewSanitizer(),
	}
}

// NewCodingAgentExecutorWithCommandExecutor creates executor with custom command executor
// This is useful for testing and dependency injection
func NewCodingAgentExecutorWithCommandExecutor(cmdExec git.CommandExecutor) CodingAgentExecutor {
	authProvider, _ := auth.GetDefaultAuthProvider() // Ignore error, auth is optional
	return &realCodingAgentExecutor{
		commandExecutor: cmdExec,
		authProvider:    authProvider,
		sanitizer:       auth.NewSanitizer(),
	}
}

// TaskStart initiates a coding task in the pod
func (r *realCodingAgentExecutor) TaskStart(ctx context.Context, namespace, podName, prompt string) (string, error) {
	// Get authentication credentials if auth provider is available
	var token string
	if r.authProvider != nil {
		// Check if token needs refresh
		if r.authProvider.NeedsRefresh() {
			if err := r.authProvider.Refresh(ctx); err != nil {
				return "", r.sanitizer.SanitizeError(fmt.Errorf("failed to refresh credentials: %w", err))
			}
		}

		// Get credentials
		creds, err := r.authProvider.GetCredentials(ctx)
		if err != nil {
			return "", r.sanitizer.SanitizeError(fmt.Errorf("failed to get credentials: %w", err))
		}

		token = creds.Token
		r.sanitizer.AddToken(token)
	}

	// For now, this is a placeholder that echoes the prompt
	// Future implementation will invoke actual claude-code agent
	// Example: claude-code agent --token "$TOKEN" --prompt "$PROMPT"

	// Escape single quotes in prompt for shell safety
	escapedPrompt := strings.ReplaceAll(prompt, "'", "'\\''")

	var command []string
	if token != "" {
		// If we have a token, we could pass it to claude-code
		// For now, just echo that we have authentication
		command = []string{
			"sh", "-c",
			fmt.Sprintf("echo 'Task started with prompt: %s (authenticated)' && echo 'task-placeholder-id'", escapedPrompt),
		}
	} else {
		command = []string{
			"sh", "-c",
			fmt.Sprintf("echo 'Task started with prompt: %s' && echo 'task-placeholder-id'", escapedPrompt),
		}
	}

	stdout, stderr, err := r.commandExecutor.ExecInPod(ctx, namespace, podName, command)
	if err != nil {
		return "", r.sanitizer.SanitizeError(fmt.Errorf("failed to start task: %s: %w", stderr, err))
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
