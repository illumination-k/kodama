package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/illumination-k/kodama/internal/version"
	"github.com/illumination-k/kodama/pkg/application"
	"github.com/illumination-k/kodama/pkg/commands"
)

// NewRootCommand creates the root command for kubectl-kodama with dependency injection
func NewRootCommand(app *application.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl-kodama",
		Short: "Manage Claude Code sessions in Kubernetes",
		Long: `kubectl-kodama is a kubectl plugin for managing Claude Code development sessions.
It provides a simple interface to start, stop, and manage containerized development
environments in your Kubernetes cluster.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	cmd.PersistentFlags().StringP("namespace", "n", "", "Kubernetes namespace")
	cmd.PersistentFlags().String("kubeconfig", "", "Path to kubeconfig file")

	// Add subcommands with dependency injection
	cmd.AddCommand(commands.NewStartCommand())           // Keep using old start command for now
	cmd.AddCommand(NewListCommand(app.SessionService))   // New refactored command
	cmd.AddCommand(commands.NewAttachCommand())          // Keep using old attach command for now
	cmd.AddCommand(NewDeleteCommand(app.SessionService)) // New refactored command
	cmd.AddCommand(commands.NewDebugCommand())           // Debug command for manifest generation
	cmd.AddCommand(commands.NewDevCommand())             // Keep using old dev command for now
	cmd.AddCommand(newVersionCommand())

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kubectl-kodama version %s\n", version.Version)
		},
	}
}
