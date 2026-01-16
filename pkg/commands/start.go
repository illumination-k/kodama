package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/illumination-k/kodama/pkg/agent"
	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/git"
	"github.com/illumination-k/kodama/pkg/kubernetes"
	"github.com/illumination-k/kodama/pkg/sync"
	"github.com/illumination-k/kodama/pkg/sync/exclude"
	"github.com/spf13/cobra"
)

// NewStartCommand creates a new start command
func NewStartCommand() *cobra.Command {
	var (
		repo         string
		syncPath     string
		namespace    string
		cpu          string
		memory       string
		branch       string
		noSync       bool
		prompt       string
		promptFile   string
		image        string
		gitSecret    string
		cloneDepth   int
		singleBranch bool
		gitCloneArgs string
		configFile   string
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

			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
			return runStart(args[0], repo, syncPath, namespace, cpu, memory, branch, noSync, kubeconfigPath, prompt, promptFile, image, gitSecret, cloneDepth, singleBranch, gitCloneArgs, configFile)
		},
	}

	// Flags
	cmd.Flags().StringVar(&repo, "repo", "", "Git repository URL to clone (use with --no-sync)")
	cmd.Flags().StringVar(&syncPath, "sync", "", "Local path to sync (default: current directory)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&cpu, "cpu", "", "CPU limit (e.g., '1', '2')")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory limit (e.g., '2Gi', '4Gi')")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to clone (default: repository default branch)")
	cmd.Flags().BoolVar(&noSync, "no-sync", false, "Disable file synchronization")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Prompt for coding agent")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "File containing prompt for coding agent")
	cmd.Flags().StringVar(&image, "image", "", "Container image to use (overrides global default)")
	cmd.Flags().StringVar(&gitSecret, "git-secret", "", "Kubernetes secret name for git credentials (overrides global default)")
	cmd.Flags().IntVar(&cloneDepth, "clone-depth", 0, "Create a shallow clone with specified depth (0 = full clone)")
	cmd.Flags().BoolVar(&singleBranch, "single-branch", false, "Clone only the specified branch (or default branch)")
	cmd.Flags().StringVar(&gitCloneArgs, "git-clone-args", "", "Additional arguments to pass to git clone (advanced)")
	cmd.Flags().StringVar(&configFile, "config", "", "Path to session template config file")

	return cmd
}

func runStart(name, repo, syncPath, namespace, cpu, memory, branch string, noSync bool, kubeconfigPath, prompt, promptFile, image, gitSecret string, cloneDepth int, singleBranch bool, gitCloneArgs, configFile string) error {
	ctx := context.Background()

	// 1. Load global config for defaults
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize config store: %w", err)
	}

	globalConfig, err := store.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// 1.5 Load session template config if specified
	var templateConfig *config.SessionConfig
	if configFile != "" {
		fmt.Printf("Loading session template from: %s\n", configFile)
		var loadedTemplate *config.SessionConfig
		loadedTemplate, err = store.LoadSessionTemplate(configFile)
		if err != nil {
			return fmt.Errorf("failed to load session template: %w", err)
		}
		templateConfig = loadedTemplate
		fmt.Println("✓ Template loaded")
	}

	// 2. Check if session already exists
	existingSessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list existing sessions: %w", err)
	}

	for _, s := range existingSessions {
		if s.Name == name {
			return fmt.Errorf("session '%s' already exists. Use 'kubectl kodama delete %s' to remove it first", name, name)
		}
	}

	// 3. Apply defaults with 4-tier priority merge
	// Priority: CLI flags > Template config > Global config > Hardcoded defaults

	// Layer 1: Apply global config defaults (if CLI flag is empty)
	if namespace == "" {
		namespace = globalConfig.Defaults.Namespace
	}
	if cpu == "" {
		cpu = globalConfig.Defaults.Resources.CPU
	}
	if memory == "" {
		memory = globalConfig.Defaults.Resources.Memory
	}
	if image == "" {
		image = globalConfig.Defaults.Image
	}
	if gitSecret == "" {
		gitSecret = globalConfig.Git.SecretName
	}

	// Layer 2: Apply template config (if --config specified and CLI flag is empty)
	if templateConfig != nil {
		if namespace == "" && templateConfig.Namespace != "" {
			namespace = templateConfig.Namespace
		}
		if cpu == "" && templateConfig.Resources.CPU != "" {
			cpu = templateConfig.Resources.CPU
		}
		if memory == "" && templateConfig.Resources.Memory != "" {
			memory = templateConfig.Resources.Memory
		}
		if image == "" && templateConfig.Image != "" {
			image = templateConfig.Image
		}
		if gitSecret == "" && templateConfig.GitSecret != "" {
			gitSecret = templateConfig.GitSecret
		}
		if branch == "" && templateConfig.Branch != "" {
			branch = templateConfig.Branch
		}
		if cloneDepth == 0 && templateConfig.GitClone.Depth > 0 {
			cloneDepth = templateConfig.GitClone.Depth
		}
		if !singleBranch && templateConfig.GitClone.SingleBranch {
			singleBranch = templateConfig.GitClone.SingleBranch
		}
		if gitCloneArgs == "" && templateConfig.GitClone.ExtraArgs != "" {
			gitCloneArgs = templateConfig.GitClone.ExtraArgs
		}
		if repo == "" && templateConfig.Repo != "" {
			repo = templateConfig.Repo
		}
	}

	// Layer 3: CLI flags (already set, highest priority - no override needed)

	// Validate required fields after merge
	if namespace == "" {
		return fmt.Errorf("namespace is required. Specify via --namespace flag, template config, or set default in ~/.kodama/config.yaml")
	}

	// 4. Determine sync path
	var syncEnabled bool
	var resolvedSyncPath string
	if !noSync {
		if syncPath != "" {
			resolvedSyncPath = syncPath
			syncEnabled = true
		} else {
			// Default to current directory
			cwd, cwdErr := os.Getwd()
			if cwdErr == nil {
				resolvedSyncPath = cwd
				syncEnabled = true
			}
		}
	}

	// 5. Validate that either sync or repo is provided
	if !syncEnabled && repo == "" {
		return fmt.Errorf("either --sync or --repo must be specified. Use --sync to sync local files, or --repo to clone a git repository")
	}

	// 6. Validate clone options
	if cloneDepth < 0 {
		return fmt.Errorf("clone depth must be non-negative (got %d)", cloneDepth)
	}

	if gitCloneArgs != "" {
		if validateErr := git.ValidateCloneArgs(gitCloneArgs); validateErr != nil {
			return fmt.Errorf("invalid git clone arguments: %w", validateErr)
		}
	}

	// 7. Create session config
	now := time.Now()
	session := &config.SessionConfig{
		Name:      name,
		Namespace: namespace,
		Repo:      repo,
		PodName:   fmt.Sprintf("kodama-%s", name),
		Image:     image,
		GitSecret: gitSecret,
		GitClone: config.GitCloneConfig{
			Depth:        cloneDepth,
			SingleBranch: singleBranch,
			ExtraArgs:    gitCloneArgs,
		},
		Status:     config.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		AutoBranch: true, // Enable automatic branch management by default
		Resources: config.ResourceConfig{
			CPU:    cpu,
			Memory: memory,
		},
		Sync: config.SyncConfig{
			Enabled:   syncEnabled,
			LocalPath: resolvedSyncPath,
		},
	}

	// Merge any additional template config fields not handled by CLI flags
	if templateConfig != nil {
		if len(templateConfig.Sync.Exclude) > 0 {
			session.Sync.Exclude = templateConfig.Sync.Exclude
		}
		if templateConfig.Sync.UseGitignore != nil {
			session.Sync.UseGitignore = templateConfig.Sync.UseGitignore
		}
		if templateConfig.ClaudeAuth != nil {
			session.ClaudeAuth = templateConfig.ClaudeAuth
		}
	}

	// Validate session
	if validateErr := session.Validate(); validateErr != nil {
		return fmt.Errorf("invalid session configuration: %w", validateErr)
	}

	// 6. Save initial session config
	if saveErr := store.SaveSession(session); saveErr != nil {
		return fmt.Errorf("failed to save session config: %w", saveErr)
	}

	// 7. Create K8s client
	k8sClient, err := kubernetes.NewClient(kubeconfigPath)
	if err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 8. Update status to Starting
	session.UpdateStatus(config.StatusStarting)
	if err := store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	// Progress indicator
	fmt.Printf("Creating session '%s'...\n", name)

	// 9. Create editor configuration ConfigMap
	fmt.Println("⏳ Creating editor configuration...")
	configMapName := fmt.Sprintf("kodama-editor-config-%s", name)
	configPath := ""
	if syncEnabled && resolvedSyncPath != "" {
		configPath = resolvedSyncPath
	}

	if err := k8sClient.CreateEditorConfigMap(ctx, namespace, configMapName, configPath); err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session)
		return fmt.Errorf("failed to create editor configuration: %w\n\nCleanup: kubectl kodama delete %s", err, name)
	}
	fmt.Println("✓ Editor configuration created")

	// 10. Create pod
	fmt.Println("⏳ Creating pod...")

	// Determine which image to use: session config > global default
	effectiveImage := globalConfig.Defaults.Image
	if session.Image != "" {
		effectiveImage = session.Image
	}

	// Validate image
	if effectiveImage == "" {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session)
		return fmt.Errorf("container image is required. Specify via --image flag or set default in ~/.kodama/config.yaml")
	}

	// Determine which git secret to use: session config > global default
	effectiveGitSecret := globalConfig.Git.SecretName
	if session.GitSecret != "" {
		effectiveGitSecret = session.GitSecret
	}

	podSpec := &kubernetes.PodSpec{
		Name:                session.PodName,
		Namespace:           namespace,
		Image:               effectiveImage,
		CPULimit:            cpu,
		MemoryLimit:         memory,
		GitSecretName:       effectiveGitSecret,
		EditorConfigMapName: configMapName,
		Command:             []string{"sleep", "infinity"},
	}

	if err := k8sClient.CreatePod(ctx, podSpec); err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return fmt.Errorf("failed to create pod: %w\n\nCleanup: kubectl kodama delete %s", err, name)
	}
	fmt.Println("✓ Pod created")

	// 10. Wait for pod ready
	fmt.Println("⏳ Waiting for pod to be ready...")
	if err := k8sClient.WaitForPodReady(ctx, session.PodName, namespace, 5*time.Minute); err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return fmt.Errorf("pod failed to start: %w\n\nTroubleshooting:\n  kubectl logs %s -n %s\n  kubectl describe pod %s -n %s\n\nCleanup: kubectl kodama delete %s",
			err, session.PodName, namespace, session.PodName, namespace, name)
	}
	fmt.Println("✓ Pod is ready")

	// 11. Clone git repository (if repo is specified and sync is disabled)
	if !syncEnabled && repo != "" {
		fmt.Printf("⏳ Cloning repository: %s...\n", repo)

		gitMgr := git.NewGitManager()

		// Get GitHub token from environment or global config
		token := os.Getenv("GITHUB_TOKEN")
		// Note: if globalConfig.Git.SecretName is set, token is available in pod via GH_TOKEN env var
		// The git clone command will use the token injected in the URL

		// Build clone options from session config
		cloneOpts := &git.CloneOptions{
			Branch:       branch,
			Depth:        session.GitClone.Depth,
			SingleBranch: session.GitClone.SingleBranch,
			ExtraArgs:    session.GitClone.ExtraArgs,
		}

		if err := gitMgr.CloneWithOptions(ctx, namespace, session.PodName, repo, token, cloneOpts); err != nil {
			session.UpdateStatus(config.StatusFailed)
			_ = store.SaveSession(session)
			return fmt.Errorf("failed to clone repository: %w\n\nTroubleshooting:\n  - Verify repository URL is correct\n  - Check authentication if private repo (use GITHUB_TOKEN env var or configure git.secretName in ~/.kodama/config.yaml)\n  - Ensure pod has network access\n  - View logs: kubectl logs %s -n %s\n\nCleanup: kubectl kodama delete %s",
				err, session.PodName, namespace, name)
		}

		fmt.Println("✓ Repository cloned")

		// 11.1 Handle automatic branch creation/checkout
		currentBranch, branchErr := gitMgr.GetCurrentBranch(ctx, namespace, session.PodName)
		if branchErr != nil {
			// Log warning but don't fail - branch management is optional
			fmt.Printf("⚠️  Warning: Could not determine current branch: %v\n", branchErr)
			fmt.Println("   Continuing with current state.")
		} else {
			// Determine target branch
			var targetBranch string
			var needsNewBranch bool

			switch {
			case branch != "" && git.IsMainBranch(branch):
				// User specified a main branch - auto-create new branch instead
				fmt.Printf("⚠️  Main branch '%s' detected - creating feature branch instead\n", branch)
				targetBranch = git.GenerateBranchName(globalConfig.Defaults.BranchPrefix, name)
				needsNewBranch = true
			case branch == "" && git.IsMainBranch(currentBranch):
				// No branch specified and cloned default is main - create new branch
				fmt.Printf("⚠️  Repository default branch '%s' is protected - creating feature branch\n", currentBranch)
				targetBranch = git.GenerateBranchName(globalConfig.Defaults.BranchPrefix, name)
				needsNewBranch = true
			case branch != "" && branch != currentBranch:
				// User specified a non-main branch - try to check it out
				targetBranch = branch
				needsNewBranch = false
			default:
				// Already on the correct branch (user-specified non-main)
				targetBranch = currentBranch
				needsNewBranch = false
			}

			// Handle branch creation or checkout if needed
			if targetBranch != currentBranch {
				fmt.Printf("⏳ Setting up branch: %s...\n", targetBranch)

				// Check if branch exists
				localExists, remoteExists, checkErr := gitMgr.BranchExists(ctx, namespace, session.PodName, targetBranch)
				switch {
				case checkErr != nil:
					fmt.Printf("⚠️  Warning: Could not check branch existence: %v\n", checkErr)
					fmt.Println("   Continuing with current branch.")
				case remoteExists:
					// Checkout existing remote branch
					if checkoutErr := gitMgr.CheckoutBranch(ctx, namespace, session.PodName, targetBranch); checkoutErr != nil {
						fmt.Printf("⚠️  Warning: Could not checkout remote branch '%s': %v\n", targetBranch, checkoutErr)
						fmt.Println("   Continuing with current branch.")
					} else {
						fmt.Printf("✓ Checked out existing remote branch: %s\n", targetBranch)
						currentBranch = targetBranch
					}
				case localExists:
					// Checkout existing local branch
					if checkoutErr := gitMgr.CheckoutBranch(ctx, namespace, session.PodName, targetBranch); checkoutErr != nil {
						fmt.Printf("⚠️  Warning: Could not checkout branch '%s': %v\n", targetBranch, checkoutErr)
						fmt.Println("   Continuing with current branch.")
					} else {
						fmt.Printf("✓ Checked out existing branch: %s\n", targetBranch)
						currentBranch = targetBranch
					}
				case needsNewBranch || !localExists:
					// Create new branch (either for main protection or user-specified branch doesn't exist)
					if createErr := gitMgr.CreateBranch(ctx, namespace, session.PodName, targetBranch); createErr != nil {
						fmt.Printf("⚠️  Warning: Could not create branch '%s': %v\n", targetBranch, createErr)
						fmt.Println("   Continuing with current branch.")
					} else {
						fmt.Printf("✓ Created new branch: %s\n", targetBranch)
						currentBranch = targetBranch
					}
				}
			}
		}

		// Store git metadata in session
		session.Repo = repo
		session.Branch = currentBranch // Use actual checked-out branch, not requested branch
		currentCommit, commitErr := gitMgr.GetCurrentCommit(ctx, namespace, session.PodName)
		if commitErr == nil {
			session.CommitHash = currentCommit
		}
	}

	// 12. Perform initial sync (if enabled)
	if syncEnabled {
		fmt.Printf("⏳ Performing initial sync: %s → pod...\n", resolvedSyncPath)

		syncMgr := sync.NewSyncManager()

		// Build exclude config
		excludeCfg := buildExcludeConfig(resolvedSyncPath, globalConfig, session)

		// Perform one-time sync
		if err := syncMgr.InitialSync(ctx, resolvedSyncPath, namespace, session.PodName, excludeCfg); err != nil {
			fmt.Printf("⚠️  Warning: Failed to sync: %v\n", err)
			fmt.Println("   Continuing without sync.")
			session.Sync.Enabled = false
		} else {
			fmt.Println("✓ Initial sync completed")
			fmt.Println("   Tip: Use 'kubectl kodama attach --sync' for live sync during development")
		}
	}

	// 12. Update status to Running and save
	session.UpdateStatus(config.StatusRunning)
	session.UpdatedAt = time.Now()
	if err := store.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save final session state: %w", err)
	}

	// 13. Execute coding agent task if prompt provided
	if prompt != "" || promptFile != "" {
		var finalPrompt string
		var promptErr error

		if promptFile != "" {
			fmt.Printf("\n⏳ Reading prompt from file: %s\n", promptFile)
			finalPrompt, promptErr = config.ReadPromptFromFile(promptFile)
			if promptErr != nil {
				fmt.Printf("⚠️  Warning: Failed to read prompt file: %v\n", promptErr)
				fmt.Println("   Session is running. You can manually invoke the agent later.")
			} else {
				fmt.Println("✓ Prompt loaded")
			}
		} else {
			finalPrompt = prompt
		}

		// Only proceed with agent execution if we have a valid prompt
		if promptErr == nil && finalPrompt != "" {
			// Create agent executor
			agentExecutor := agent.NewCodingAgentExecutor()

			// Start the agent through session
			fmt.Println("\n🤖 Initiating coding agent...")
			if agentErr := session.StartAgent(ctx, agentExecutor, finalPrompt); agentErr != nil {
				// Don't fail the entire start command if agent fails
				// The session is already created and running
				fmt.Printf("⚠️  Warning: Failed to start coding agent: %v\n", agentErr)
				fmt.Println("   Session is running. You can manually invoke the agent later.")
			} else {
				fmt.Println("✓ Agent task started")
			}

			// Save updated session with agent execution record
			if err := store.SaveSession(session); err != nil {
				fmt.Printf("⚠️  Warning: Failed to save agent execution record: %v\n", err)
			}
		}
	}

	// 14. Print success message
	fmt.Printf("\n✨ Session '%s' is ready!\n", name)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  kubectl kodama attach %s    # Attach to session\n", name)
	fmt.Printf("  kubectl kodama list         # List all sessions\n")
	fmt.Printf("  kubectl kodama delete %s    # Delete session\n", name)

	if syncEnabled && session.Sync.Enabled {
		fmt.Printf("\n📁 Files are syncing between %s and pod\n", resolvedSyncPath)
	}

	return nil
}

// buildExcludeConfig creates exclude.Config from global and session configs
func buildExcludeConfig(localPath string, globalCfg *config.GlobalConfig, sessionCfg *config.SessionConfig) *exclude.Config {
	// Determine if gitignore should be used
	useGitignore := true // default

	// Global config override
	if globalCfg.Sync.UseGitignore != nil {
		useGitignore = *globalCfg.Sync.UseGitignore
	}

	// Session config override (highest priority)
	if sessionCfg.Sync.UseGitignore != nil {
		useGitignore = *sessionCfg.Sync.UseGitignore
	}

	// Merge patterns: session overrides global
	var patterns []string

	if len(sessionCfg.Sync.Exclude) > 0 {
		// Session patterns completely replace global patterns
		patterns = sessionCfg.Sync.Exclude
	} else {
		// Use global patterns
		patterns = globalCfg.Sync.Exclude
	}

	return &exclude.Config{
		BasePath:     localPath,
		Patterns:     patterns,
		UseGitignore: useGitignore,
	}
}
