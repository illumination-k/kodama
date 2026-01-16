package commands

import (
	"context"

	"github.com/illumination-k/kodama/pkg/usecase"
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

			opts := usecase.AttachSessionOptions{
				Name:           args[0],
				Command:        command,
				KubeconfigPath: kubeconfigPath,
			}

			return usecase.AttachSession(context.Background(), opts)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Command to run in pod (default: interactive shell)")

	return cmd
}
