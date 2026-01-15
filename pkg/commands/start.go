package commands

import (
	"context"
	"fmt"
	"os"
	"time"

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
		repo      string
		syncPath  string
		namespace string
		cpu       string
		memory    string
		branch    string
		noSync    bool
	)

	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a new Claude Code session",
		Long: `Start a new Claude Code session in Kubernetes.

Creates a pod running claude-code and syncs files from your local machine.

Examples:
  kubectl kodama start my-work --sync ~/projects/myrepo
  kubectl kodama start my-work --namespace dev --cpu 2 --memory 4Gi
  kubectl kodama start my-work --no-sync`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
			return runStart(args[0], repo, syncPath, namespace, cpu, memory, branch, noSync, kubeconfigPath)
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

	return cmd
}

func runStart(name, repo, syncPath, namespace, cpu, memory, branch string, noSync bool, kubeconfigPath string) error {
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

	// 3. Apply defaults from global config and flags
	if namespace == "" {
		namespace = globalConfig.Defaults.Namespace
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required. Specify via --namespace flag or set default in ~/.kodama/config.yaml")
	}

	if cpu == "" {
		cpu = globalConfig.Defaults.Resources.CPU
	}
	if memory == "" {
		memory = globalConfig.Defaults.Resources.Memory
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

	// 5. Create session config
	now := time.Now()
	session := &config.SessionConfig{
		Name:      name,
		Namespace: namespace,
		Repo:      repo,
		PodName:   fmt.Sprintf("kodama-%s", name),
		Status:    config.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		Resources: config.ResourceConfig{
			CPU:    cpu,
			Memory: memory,
		},
		Sync: config.SyncConfig{
			Enabled:   syncEnabled,
			LocalPath: resolvedSyncPath,
		},
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

	// 9. Create pod
	fmt.Println("⏳ Creating pod...")
	podSpec := &kubernetes.PodSpec{
		Name:          session.PodName,
		Namespace:     namespace,
		Image:         globalConfig.Defaults.Image,
		CPULimit:      cpu,
		MemoryLimit:   memory,
		GitSecretName: globalConfig.Git.SecretName,
		Command:       []string{"sleep", "infinity"},
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

		if err := gitMgr.Clone(ctx, namespace, session.PodName, repo, branch, token); err != nil {
			session.UpdateStatus(config.StatusFailed)
			_ = store.SaveSession(session)
			return fmt.Errorf("failed to clone repository: %w\n\nTroubleshooting:\n  - Verify repository URL is correct\n  - Check authentication if private repo (use GITHUB_TOKEN env var or configure git.secretName in ~/.kodama/config.yaml)\n  - Ensure pod has network access\n  - View logs: kubectl logs %s -n %s\n\nCleanup: kubectl kodama delete %s",
				err, session.PodName, namespace, name)
		}

		fmt.Println("✓ Repository cloned")

		// Store git metadata in session
		session.Repo = repo
		session.Branch = branch
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

	// 13. Print success message
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
	patterns := []string{}

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
