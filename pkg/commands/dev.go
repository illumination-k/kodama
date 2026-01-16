package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/illumination-k/kodama/pkg/agent"
	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/gitcmd"
	"github.com/illumination-k/kodama/pkg/kubernetes"
	"github.com/illumination-k/kodama/pkg/sync"
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
			return runDev(args[0], repo, syncPath, namespace, cpu, memory, branch, kubeconfigPath, prompt, promptFile, image, command, gitSecret, cloneDepth, singleBranch, gitCloneArgs, configFile, attachCmd)
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

func runDev(name, repo, syncPath, namespace, cpu, memory, branch, kubeconfigPath, prompt, promptFile, image, command, gitSecret string, cloneDepth int, singleBranch bool, gitCloneArgs, configFile, attachCmd string) error {
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

	// 1.5 Load session template config if specified or found in current directory
	var templateConfig *config.SessionConfig

	// Auto-detect .kodama.yaml in current directory if --config not specified
	if configFile == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr == nil {
			candidatePath := fmt.Sprintf("%s/.kodama.yaml", cwd)
			if _, statErr := os.Stat(candidatePath); statErr == nil {
				configFile = candidatePath
				fmt.Printf("📄 Found .kodama.yaml in current directory\n")
			}
		}
	}

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
		if command == "" && len(templateConfig.Command) > 0 {
			command = strings.Join(templateConfig.Command, " ")
		}
	}

	// Layer 3: CLI flags (already set, highest priority - no override needed)

	// Validate required fields after merge
	if namespace == "" {
		return fmt.Errorf("namespace is required. Specify via --namespace flag, template config, or set default in ~/.kodama/config.yaml")
	}

	// 4. Validate mutual exclusivity between --repo and --sync
	if syncPath != "" && repo != "" {
		return fmt.Errorf("cannot use both --sync and --repo. Choose one mode per session")
	}

	// 5. Determine sync path (only when repo is not specified)
	var syncEnabled bool
	var resolvedSyncPath string
	if repo == "" {
		if syncPath != "" {
			resolvedSyncPath = syncPath
			syncEnabled = true
		} else {
			// Default to current directory when neither --repo nor --sync specified
			cwd, cwdErr := os.Getwd()
			if cwdErr == nil {
				resolvedSyncPath = cwd
				syncEnabled = true
			}
		}
	}

	// 6. Validate clone options
	if cloneDepth < 0 {
		return fmt.Errorf("clone depth must be non-negative (got %d)", cloneDepth)
	}

	if gitCloneArgs != "" {
		if validateErr := gitcmd.ValidateCloneArgs(gitCloneArgs); validateErr != nil {
			return fmt.Errorf("invalid git clone arguments: %w", validateErr)
		}
	}

	// Parse command string into slice
	var cmdSlice []string
	if command != "" {
		cmdSlice = strings.Fields(command)
	}

	// 7. Create session config
	now := time.Now()
	session := &config.SessionConfig{
		Name:      name,
		Namespace: namespace,
		Repo:      repo,
		PodName:   fmt.Sprintf("kodama-%s", name),
		Image:     image,
		Command:   cmdSlice,
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
		if len(templateConfig.Sync.CustomDirs) > 0 {
			session.Sync.CustomDirs = templateConfig.Sync.CustomDirs
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

	// Track which Kubernetes resources are created for cleanup on failure
	var (
		k8sClient      *kubernetes.Client
		podCreated     bool
		startSucceeded bool // Set to true at the very end to skip cleanup
	)

	// Setup cleanup on error - will only run if startSucceeded is false
	defer func() {
		if !startSucceeded && k8sClient != nil {
			cleanupFailedStart(ctx, k8sClient, namespace, session.PodName, podCreated)
		}
	}()

	// 7. Create K8s client
	k8sClient, err = kubernetes.NewClient(kubeconfigPath)
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

	// 9. Create pod
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

	// Determine branch name for init container (if repo mode)
	effectiveBranch := branch
	if repo != "" && effectiveBranch == "" {
		// Generate default branch name if not specified
		effectiveBranch = fmt.Sprintf("kodama/%s", name)
	}

	// Determine command to run in pod
	effectiveCommand := session.Command
	if len(effectiveCommand) == 0 {
		effectiveCommand = []string{"sleep", "infinity"}
	}

	podSpec := &kubernetes.PodSpec{
		Name:          session.PodName,
		Namespace:     namespace,
		Image:         effectiveImage,
		CPULimit:      cpu,
		MemoryLimit:   memory,
		GitSecretName: effectiveGitSecret,
		Command:       effectiveCommand,

		// Git configuration for workspace-initializer init container
		GitRepo:         repo,
		GitBranch:       effectiveBranch,
		GitCloneDepth:   cloneDepth,
		GitSingleBranch: singleBranch,
		GitCloneArgs:    gitCloneArgs,
	}

	if err := k8sClient.CreatePod(ctx, podSpec); err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return fmt.Errorf("failed to create pod: %w", err)
	}
	podCreated = true
	fmt.Println("✓ Pod created")

	// 10. Wait for pod ready (including init containers)
	if repo != "" {
		fmt.Printf("⏳ Waiting for init containers (installing Claude Code and cloning repository: %s)...\n", repo)
	} else {
		fmt.Println("⏳ Waiting for init containers (installing Claude Code)...")
	}
	if err := k8sClient.WaitForPodReady(ctx, session.PodName, namespace, 5*time.Minute); err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return fmt.Errorf("pod failed to start: %w\n\nTroubleshooting:\n  kubectl logs %s -c claude-installer -n %s\n  kubectl logs %s -c workspace-initializer -n %s\n  kubectl describe pod %s -n %s",
			err, session.PodName, namespace, session.PodName, namespace, session.PodName, namespace)
	}
	fmt.Println("✓ Init containers completed")

	// Store git metadata in session if repo mode
	if repo != "" {
		session.Repo = repo
		session.Branch = effectiveBranch
		// Note: Commit hash will be populated if needed via git operations in the pod later
	}

	// 11. Perform initial sync (if enabled) - runs AFTER init containers complete
	if syncEnabled {
		fmt.Printf("⏳ Syncing local files: %s → pod...\n", resolvedSyncPath)

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
		}

		// Sync custom directories (dotfiles, configs, etc.)
		customDirs := determineCustomDirs(globalConfig, session)
		if len(customDirs) > 0 {
			customSyncMgr := sync.NewCustomDirSyncManager(syncMgr)
			if err := customSyncMgr.SyncCustomDirs(ctx, customDirs, namespace, session.PodName, globalConfig); err != nil {
				fmt.Printf("⚠️  Warning: Failed to sync custom directories: %v\n", err)
			}
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

	if syncEnabled && session.Sync.Enabled {
		fmt.Printf("📁 Files synced from %s\n", resolvedSyncPath)
	}

	// Mark start as successful to skip cleanup
	startSucceeded = true

	// 15. Attach to session
	fmt.Printf("\n🔗 Attaching to session '%s'...\n", name)

	var execCmd *exec.Cmd

	if attachCmd != "" {
		// Run specific command
		//#nosec G204 -- kubectl exec with user command is the intended functionality
		execCmd = exec.CommandContext(ctx, "kubectl", "exec", "-it",
			"-n", session.Namespace,
			session.PodName,
			"--",
			"/bin/bash", "-c", fmt.Sprintf("cd /workspace && %s", attachCmd),
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
