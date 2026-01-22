package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/illumination-k/kodama/pkg/usecase"
)

// NewStartCommand creates a new start command
func NewStartCommand() *cobra.Command {
	var (
		repo            string
		syncPath        string
		namespace       string
		cpu             string
		memory          string
		customResources []string
		branch          string
		prompt          string
		promptFile      string
		image           string
		command         string
		gitSecret       string
		cloneDepth      int
		singleBranch    bool
		gitCloneArgs    string
		configFile      string
		ttydEnabled     bool
		ttydPort        int
		ttydOptions     string
		ttydReadonly    bool
		envFiles        []string
		envExclude      []string
	)

	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a new Claude Code session",
		Long: `Start a new Claude Code session in Kubernetes.

Creates a pod running claude-code and syncs files from your local machine.

Examples:
  kubectl kodama start my-work --sync ~/projects/myrepo
  kubectl kodama start my-work --repo https://github.com/user/repo --branch main
  kubectl kodama start my-work --namespace dev --cpu 2 --memory 4Gi`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate mutual exclusivity of prompt flags
			if prompt != "" && promptFile != "" {
				return fmt.Errorf("cannot specify both --prompt and --prompt-file")
			}

			// Parse custom resources
			customResourcesMap := make(map[string]string)
			for _, res := range customResources {
				parts := strings.Split(res, "=")
				if len(parts) != 2 {
					return fmt.Errorf("invalid resource format: %s (expected format: resourceName=quantity, e.g., nvidia.com/gpu=1)", res)
				}
				customResourcesMap[parts[0]] = parts[1]
			}

			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")

			opts := usecase.StartSessionOptions{
				Name:            args[0],
				Repo:            repo,
				SyncPath:        syncPath,
				Namespace:       namespace,
				CPU:             cpu,
				Memory:          memory,
				CustomResources: customResourcesMap,
				Branch:          branch,
				KubeconfigPath:  kubeconfigPath,
				Prompt:          prompt,
				PromptFile:      promptFile,
				Image:           image,
				Command:         command,
				GitSecret:       gitSecret,
				CloneDepth:      cloneDepth,
				SingleBranch:    singleBranch,
				GitCloneArgs:    gitCloneArgs,
				ConfigFile:      configFile,
				TtydEnabled:     cmd.Flags().Changed("ttyd"),
				TtydEnabledVal:  ttydEnabled,
				TtydPort:        ttydPort,
				TtydOptions:     ttydOptions,
				TtydReadonly:    ttydReadonly,
				TtydReadonlySet: cmd.Flags().Changed("ttyd-readonly"),
				EnvFiles:        envFiles,
				EnvExclude:      envExclude,
			}

			session, err := usecase.StartSession(context.Background(), opts)
			if err != nil {
				return err
			}

			// Print success message
			fmt.Printf("\n✨ Session '%s' is ready!\n", session.Name)

			isTtydEnabled := session.Ttyd.Enabled != nil && *session.Ttyd.Enabled
			if isTtydEnabled {
				fmt.Printf("\n🌐 Web-based terminal (ttyd) is enabled\n")
				fmt.Printf("   The session will be accessible via browser\n")
			}

			fmt.Printf("\nNext steps:\n")
			if isTtydEnabled {
				fmt.Printf("  kubectl kodama attach %s           # Open in browser (ttyd)\n", session.Name)
				fmt.Printf("  kubectl kodama attach %s --tty     # Use traditional TTY mode\n", session.Name)
			} else {
				fmt.Printf("  kubectl kodama attach %s           # Attach to session\n", session.Name)
			}
			fmt.Printf("  kubectl kodama list                # List all sessions\n")
			fmt.Printf("  kubectl kodama delete %s           # Delete session\n", session.Name)

			if session.Sync.Enabled {
				fmt.Printf("\n📁 Files are syncing between %s and pod\n", session.Sync.LocalPath)
				fmt.Println("   Tip: Use 'kubectl kodama attach --sync' for live sync during development")
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVar(&repo, "repo", "", "Git repository URL to clone (mutually exclusive with --sync)")
	cmd.Flags().StringVar(&syncPath, "sync", "", "Local path to sync (default: current directory, mutually exclusive with --repo)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&cpu, "cpu", "", "CPU limit (e.g., '1', '2')")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory limit (e.g., '2Gi', '4Gi')")
	cmd.Flags().StringSliceVar(&customResources, "resource", []string{}, "Custom resource (can be specified multiple times, e.g., --resource nvidia.com/gpu=1 --resource amd.com/gpu=2)")
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
	cmd.Flags().BoolVar(&ttydEnabled, "ttyd", true, "Enable ttyd (web-based terminal)")
	cmd.Flags().IntVar(&ttydPort, "ttyd-port", 0, "Ttyd port (default: 7681)")
	cmd.Flags().StringVar(&ttydOptions, "ttyd-options", "", "Additional ttyd options")
	cmd.Flags().BoolVar(&ttydReadonly, "ttyd-readonly", false, "Enable read-only mode for ttyd (disables terminal input)")
	cmd.Flags().StringSliceVar(&envFiles, "env-file", []string{}, "Dotenv file(s) to load (can be specified multiple times)")
	cmd.Flags().StringSliceVar(&envExclude, "env-exclude", []string{}, "Environment variable names to exclude from injection (can be specified multiple times)")

	return cmd
}
