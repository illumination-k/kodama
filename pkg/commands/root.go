package commands

import (
	"fmt"

	"github.com/illumination-k/kodama/internal/version"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for kubectl-kodama
func NewRootCommand() *cobra.Command {
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

	// Add subcommands
	cmd.AddCommand(NewStartCommand())
	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(NewAttachCommand())
	cmd.AddCommand(NewDeleteCommand())
	cmd.AddCommand(NewDevCommand())
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
