package commands

import (
	"context"
	"fmt"

	"github.com/illumination-k/kodama/pkg/usecase"
	"github.com/spf13/cobra"
)

// NewDevCommand creates a new dev command that combines start and attach
func NewDevCommand() *cobra.Command {
	var (
		repo         string
		syncPath     string
		namespace    string
		cpu          string
		memory       string
		branch       string
		prompt       string
		promptFile   string
		image        string
		command      string
		gitSecret    string
		cloneDepth   int
		singleBranch bool
		gitCloneArgs string
		configFile   string
		attachCmd    string
	)

	cmd := &cobra.Command{
		Use:   "dev <name>",
		Short: "Start a new session and attach to it",
		Long: `Start a new Claude Code session and immediately attach to it.

This command combines 'start' and 'attach' into a single workflow.

Examples:
  kubectl kodama dev my-work --sync ~/projects/myrepo
  kubectl kodama dev my-work --repo https://github.com/user/repo --branch main
  kubectl kodama dev my-work --namespace dev --cpu 2 --memory 4Gi`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate mutual exclusivity of prompt flags
			if prompt != "" && promptFile != "" {
				return fmt.Errorf("cannot specify both --prompt and --prompt-file")
			}

			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
			ctx := context.Background()

			// 1. Start the session
			startOpts := usecase.StartSessionOptions{
				Name:           args[0],
				Repo:           repo,
				SyncPath:       syncPath,
				Namespace:      namespace,
				CPU:            cpu,
				Memory:         memory,
				Branch:         branch,
				KubeconfigPath: kubeconfigPath,
				Prompt:         prompt,
				PromptFile:     promptFile,
				Image:          image,
				Command:        command,
				GitSecret:      gitSecret,
				CloneDepth:     cloneDepth,
				SingleBranch:   singleBranch,
				GitCloneArgs:   gitCloneArgs,
				ConfigFile:     configFile,
			}

			session, err := usecase.StartSession(ctx, startOpts)
			if err != nil {
				return err
			}

			// Print success message
			fmt.Printf("\n✨ Session '%s' is ready!\n", session.Name)

			if session.Sync.Enabled {
				fmt.Printf("📁 Files synced from %s\n", session.Sync.LocalPath)
			}

			// 2. Attach to the session
			fmt.Printf("\n🔗 Attaching to session '%s'...\n", session.Name)

			return usecase.AttachToSession(ctx, session, attachCmd, kubeconfigPath)
		},
	}

	// Start flags
	cmd.Flags().StringVar(&repo, "repo", "", "Git repository URL to clone (mutually exclusive with --sync)")
	cmd.Flags().StringVar(&syncPath, "sync", "", "Local path to sync (default: current directory, mutually exclusive with --repo)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&cpu, "cpu", "", "CPU limit (e.g., '1', '2')")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory limit (e.g., '2Gi', '4Gi')")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to clone (default: repository default branch)")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Prompt for coding agent")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "File containing prompt for coding agent")
	cmd.Flags().StringVar(&image, "image", "", "Container image to use (overrides global default)")
	cmd.Flags().StringVar(&command, "cmd", "", "Pod command override (space-separated, e.g., 'sh -c echo hello')")
	cmd.Flags().StringVar(&gitSecret, "git-secret", "", "Kubernetes secret name for git credentials (overrides global default)")
	cmd.Flags().IntVar(&cloneDepth, "clone-depth", 0, "Create a shallow clone with specified depth (0 = full clone)")
	cmd.Flags().BoolVar(&singleBranch, "single-branch", false, "Clone only the specified branch (or default branch)")
	cmd.Flags().StringVar(&gitCloneArgs, "git-clone-args", "", "Additional arguments to pass to git clone (advanced)")
	cmd.Flags().StringVar(&configFile, "config", "", "Path to session template config file")

	// Attach flags
	cmd.Flags().StringVar(&attachCmd, "attach-command", "", "Command to run when attaching (default: interactive shell)")

	return cmd
}
