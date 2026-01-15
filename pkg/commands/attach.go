package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/kubernetes"
	"github.com/spf13/cobra"
)

// NewAttachCommand creates a new attach command
func NewAttachCommand() *cobra.Command {
	var command string

	cmd := &cobra.Command{
		Use:   "attach <name>",
		Short: "Attach to a session",
		Long: `Attach to a running session and start Claude Code.

Opens an interactive shell in the pod and launches Claude Code.

Examples:
  kubectl kodama attach my-work
  kubectl kodama attach my-work --command "claude --help"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
			return runAttach(args[0], command, kubeconfigPath)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Command to run in pod (default: interactive shell)")

	return cmd
}

func runAttach(name, command, kubeconfigPath string) error {
	ctx := context.Background()

	// 1. Load session config
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize config store: %w", err)
	}

	session, err := store.LoadSession(name)
	if err != nil {
		if errors.Is(err, config.ErrSessionNotFound) {
			return fmt.Errorf("session '%s' not found\n\nAvailable sessions:\n  kubectl kodama list", name)
		}
		return fmt.Errorf("failed to load session: %w", err)
	}

	// 2. Verify pod is running
	k8sClient, err := kubernetes.NewClient(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	podStatus, err := k8sClient.GetPod(ctx, session.PodName, session.Namespace)
	if err != nil {
		return fmt.Errorf("pod not found: %w\n\nStart the session with:\n  kubectl kodama start %s", err, name)
	}

	if !podStatus.Ready {
		return fmt.Errorf("pod is not ready (status: %s)\n\nCheck pod status:\n  kubectl get pod %s -n %s\n  kubectl describe pod %s -n %s",
			podStatus.Phase, session.PodName, session.Namespace, session.PodName, session.Namespace)
	}

	// 3. Execute kubectl exec with TTY
	fmt.Printf("Attaching to session '%s'...\n", name)

	var execCmd *exec.Cmd

	if command != "" {
		// Run specific command
		//#nosec G204 -- kubectl exec with user command is the intended functionality
		execCmd = exec.CommandContext(ctx, "kubectl", "exec", "-it",
			"-n", session.Namespace,
			session.PodName,
			"--",
			"/bin/bash", "-c", fmt.Sprintf("cd /workspace && %s", command),
		)
	} else {
		// Open interactive shell
		//#nosec G204 -- kubectl exec with session data from config store
		execCmd = exec.CommandContext(ctx, "kubectl", "exec", "-it",
			"-n", session.Namespace,
			session.PodName,
			"--",
			"/bin/bash", "-c", "cd /workspace && exec bash",
		)
	}

	// Connect stdin/stdout/stderr
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}
