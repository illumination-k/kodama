package commands

import (
	"context"

	"github.com/illumination-k/kodama/pkg/usecase"
	"github.com/spf13/cobra"
)

// NewAttachCommand creates a new attach command
func NewAttachCommand() *cobra.Command {
	var (
		command   string
		ttyMode   bool
		localPort int
		noBrowser bool
	)

	cmd := &cobra.Command{
		Use:   "attach <name>",
		Short: "Attach to a session",
		Long: `Attach to a running session and start Claude Code.

By default, uses ttyd (web-based terminal) if enabled in the session.
Opens port-forward and launches browser automatically.

Examples:
  kubectl kodama attach my-work                 # Use ttyd (open browser)
  kubectl kodama attach my-work --no-browser    # Use ttyd (no browser)
  kubectl kodama attach my-work --tty           # Force TTY mode
  kubectl kodama attach my-work --port 8080     # Custom local port
  kubectl kodama attach my-work --command "claude --help"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")

			opts := usecase.AttachSessionOptions{
				Name:           args[0],
				Command:        command,
				KubeconfigPath: kubeconfigPath,
				TtyMode:        ttyMode,
				LocalPort:      localPort,
				NoBrowser:      noBrowser,
			}

			return usecase.AttachSession(context.Background(), opts)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Command to run in pod (default: interactive shell)")
	cmd.Flags().BoolVar(&ttyMode, "tty", false, "Force TTY mode (disable ttyd)")
	cmd.Flags().IntVar(&localPort, "port", 0, "Local port for port-forward (default: same as pod port)")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")

	return cmd
}
