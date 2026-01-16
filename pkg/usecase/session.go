package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/illumination-k/kodama/pkg/agent"
	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/gitcmd"
	"github.com/illumination-k/kodama/pkg/kubernetes"
	"github.com/illumination-k/kodama/pkg/sync"
	"github.com/illumination-k/kodama/pkg/sync/exclude"
)

// StartSessionOptions contains all options for starting a session
type StartSessionOptions struct {
	Name            string
	Repo            string
	SyncPath        string
	Namespace       string
	CPU             string
	Memory          string
	Branch          string
	KubeconfigPath  string
	Prompt          string
	PromptFile      string
	Image           string
	Command         string
	GitSecret       string
	CloneDepth      int
	SingleBranch    bool
	GitCloneArgs    string
	ConfigFile      string
	TtydEnabled     bool
	TtydEnabledVal  bool
	TtydPort        int
	TtydOptions     string
	TtydReadonly    bool
	TtydReadonlySet bool
	DiffViewer      bool
	DiffViewerPort  int
}

// AttachSessionOptions contains all options for attaching to a session
type AttachSessionOptions struct {
	Name           string
	Command        string
	KubeconfigPath string
	TtyMode        bool
	LocalPort      int
	NoBrowser      bool
}

// StartSession starts a new Claude Code session and returns the session config
func StartSession(ctx context.Context, opts StartSessionOptions) (*config.SessionConfig, error) {
	// 1. Load global config for defaults
	store, err := config.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config store: %w", err)
	}

	globalConfig, err := store.LoadGlobalConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// 1.5 Load session template config if specified or found in current directory
	var templateConfig *config.SessionConfig
	configFile := opts.ConfigFile

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
			return nil, fmt.Errorf("failed to load session template: %w", err)
		}
		templateConfig = loadedTemplate
		fmt.Println("✓ Template loaded")
	}

	// 2. Check if session already exists
	existingSessions, err := store.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list existing sessions: %w", err)
	}

	for _, s := range existingSessions {
		if s.Name == opts.Name {
			return nil, fmt.Errorf("session '%s' already exists. Use 'kubectl kodama delete %s' to remove it first", opts.Name, opts.Name)
		}
	}

	// 3. Apply defaults with 4-tier priority merge
	// Priority: CLI flags > Template config > Global config > Hardcoded defaults
	namespace := opts.Namespace
	cpu := opts.CPU
	memory := opts.Memory
	image := opts.Image
	gitSecret := opts.GitSecret
	branch := opts.Branch
	cloneDepth := opts.CloneDepth
	singleBranch := opts.SingleBranch
	gitCloneArgs := opts.GitCloneArgs
	repo := opts.Repo
	command := opts.Command
	// Default ttyd to true if not explicitly set in global config
	ttydEnabled := true
	if globalConfig.Defaults.Ttyd.Enabled != nil {
		ttydEnabled = *globalConfig.Defaults.Ttyd.Enabled
	}
	ttydPort := globalConfig.Defaults.Ttyd.Port
	ttydOptions := globalConfig.Defaults.Ttyd.Options
	// Default ttyd writable to true if not explicitly set in global config
	ttydWritable := true
	if globalConfig.Defaults.Ttyd.Writable != nil {
		ttydWritable = *globalConfig.Defaults.Ttyd.Writable
	}

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
		// Apply template ttyd config
		if templateConfig.Ttyd.Enabled != nil {
			ttydEnabled = *templateConfig.Ttyd.Enabled
		}
		if templateConfig.Ttyd.Port != 0 {
			ttydPort = templateConfig.Ttyd.Port
		}
		if templateConfig.Ttyd.Options != "" {
			ttydOptions = templateConfig.Ttyd.Options
		}
		if templateConfig.Ttyd.Writable != nil {
			ttydWritable = *templateConfig.Ttyd.Writable
		}
	}

	// Layer 3: CLI flags (already set, highest priority - no override needed)
	// Apply CLI flag ttyd overrides
	if opts.TtydEnabled {
		ttydEnabled = opts.TtydEnabledVal
	}
	if opts.TtydPort != 0 {
		ttydPort = opts.TtydPort
	}
	if opts.TtydOptions != "" {
		ttydOptions = opts.TtydOptions
	}
	if opts.TtydReadonlySet {
		ttydWritable = !opts.TtydReadonly
	}

	// Validate required fields after merge
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required. Specify via --namespace flag, template config, or set default in ~/.kodama/config.yaml")
	}

	// 4. Validate mutual exclusivity between --repo and --sync
	if opts.SyncPath != "" && repo != "" {
		return nil, fmt.Errorf("cannot use both --sync and --repo. Choose one mode per session")
	}

	// 5. Determine sync path (only when repo is not specified)
	var syncEnabled bool
	var resolvedSyncPath string
	if repo == "" {
		if opts.SyncPath != "" {
			resolvedSyncPath = opts.SyncPath
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
		return nil, fmt.Errorf("clone depth must be non-negative (got %d)", cloneDepth)
	}

	if gitCloneArgs != "" {
		if validateErr := gitcmd.ValidateCloneArgs(gitCloneArgs); validateErr != nil {
			return nil, fmt.Errorf("invalid git clone arguments: %w", validateErr)
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
		Name:      opts.Name,
		Namespace: namespace,
		Repo:      repo,
		PodName:   fmt.Sprintf("kodama-%s", opts.Name),
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
		Ttyd: config.TtydConfig{
			Enabled:  &ttydEnabled,
			Port:     ttydPort,
			Options:  ttydOptions,
			Writable: &ttydWritable,
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
		if templateConfig.DiffViewer != nil {
			session.DiffViewer = templateConfig.DiffViewer
		}
	}

	// Apply DiffViewer settings from CLI flags (highest priority)
	if opts.DiffViewer {
		if session.DiffViewer == nil {
			session.DiffViewer = &config.DiffViewerConfig{}
		}
		session.DiffViewer.Enabled = true
	}
	if opts.DiffViewerPort > 0 && opts.DiffViewerPort <= math.MaxInt32 && session.DiffViewer != nil {
		// #nosec G115 -- Port numbers are limited to 0-65535, well within int32 range
		session.DiffViewer.Port = int32(opts.DiffViewerPort)
	}

	// Apply global DiffViewer defaults if not set
	if session.DiffViewer == nil && globalConfig.DiffViewer.Enabled {
		session.DiffViewer = &config.DiffViewerConfig{
			Enabled: true,
			Image:   globalConfig.DiffViewer.Image,
			Port:    globalConfig.DiffViewer.Port,
		}
	}

	// Validate session
	if validateErr := session.Validate(); validateErr != nil {
		return nil, fmt.Errorf("invalid session configuration: %w", validateErr)
	}

	// 6. Save initial session config
	if saveErr := store.SaveSession(session); saveErr != nil {
		return nil, fmt.Errorf("failed to save session config: %w", saveErr)
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
	k8sClient, err = kubernetes.NewClient(opts.KubeconfigPath)
	if err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 8. Update status to Starting
	session.UpdateStatus(config.StatusStarting)
	if err := store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to update session status: %w", err)
	}

	// Progress indicator
	fmt.Printf("Creating session '%s'...\n", opts.Name)

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
		return nil, fmt.Errorf("container image is required. Specify via --image flag or set default in ~/.kodama/config.yaml")
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
		effectiveBranch = fmt.Sprintf("kodama/%s", opts.Name)
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

		// Ttyd configuration
		TtydEnabled:  ttydEnabled,
		TtydPort:     ttydPort,
		TtydOptions:  ttydOptions,
		TtydWritable: ttydWritable,
	}

	// Add DiffViewer sidecar if enabled
	if session.DiffViewer != nil && session.DiffViewer.Enabled {
		podSpec.DiffViewer = &kubernetes.DiffViewerSpec{
			Enabled: true,
			Image:   session.DiffViewer.Image,
			Port:    session.DiffViewer.Port,
		}
	}

	if err := k8sClient.CreatePod(ctx, podSpec); err != nil {
		session.UpdateStatus(config.StatusFailed)
		_ = store.SaveSession(session) // Best effort update
		return nil, fmt.Errorf("failed to create pod: %w", err)
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
		return nil, fmt.Errorf("pod failed to start: %w\n\nTroubleshooting:\n  kubectl logs %s -c claude-installer -n %s\n  kubectl logs %s -c workspace-initializer -n %s\n  kubectl describe pod %s -n %s",
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
		return nil, fmt.Errorf("failed to save final session state: %w", err)
	}

	// 13. Execute coding agent task if prompt provided
	if opts.Prompt != "" || opts.PromptFile != "" {
		var finalPrompt string
		var promptErr error

		if opts.PromptFile != "" {
			fmt.Printf("\n⏳ Reading prompt from file: %s\n", opts.PromptFile)
			finalPrompt, promptErr = config.ReadPromptFromFile(opts.PromptFile)
			if promptErr != nil {
				fmt.Printf("⚠️  Warning: Failed to read prompt file: %v\n", promptErr)
				fmt.Println("   Session is running. You can manually invoke the agent later.")
			} else {
				fmt.Println("✓ Prompt loaded")
			}
		} else {
			finalPrompt = opts.Prompt
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

	// Mark start as successful to skip cleanup
	startSucceeded = true
	return session, nil
}

// AttachSession attaches to an existing session
func AttachSession(ctx context.Context, opts AttachSessionOptions) error {
	// 1. Load session config
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize config store: %w", err)
	}

	session, err := store.LoadSession(opts.Name)
	if err != nil {
		if errors.Is(err, config.ErrSessionNotFound) {
			return fmt.Errorf("session '%s' not found\n\nAvailable sessions:\n  kubectl kodama list", opts.Name)
		}
		return fmt.Errorf("failed to load session: %w", err)
	}

	// 2. Determine attachment mode
	// Use ttyd mode if: ttyd is enabled in session AND --tty flag is not set
	ttydEnabled := session.Ttyd.Enabled != nil && *session.Ttyd.Enabled
	if ttydEnabled && !opts.TtyMode {
		return attachViaTtyd(ctx, session, opts)
	}

	// Fall back to traditional TTY mode
	return AttachToSession(ctx, session, opts.Command, opts.KubeconfigPath)
}

// AttachToSession attaches to a session using the provided session config
func AttachToSession(ctx context.Context, session *config.SessionConfig, command, kubeconfigPath string) error {
	// 1. Verify pod is running
	k8sClient, err := kubernetes.NewClient(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	podStatus, err := k8sClient.GetPod(ctx, session.PodName, session.Namespace)
	if err != nil {
		return fmt.Errorf("pod not found: %w\n\nStart the session with:\n  kubectl kodama start %s", err, session.Name)
	}

	if !podStatus.Ready {
		return fmt.Errorf("pod is not ready (status: %s)\n\nCheck pod status:\n  kubectl get pod %s -n %s\n  kubectl describe pod %s -n %s",
			podStatus.Phase, session.PodName, session.Namespace, session.PodName, session.Namespace)
	}

	// 2. Execute kubectl exec with TTY
	fmt.Printf("Attaching to session '%s'...\n", session.Name)

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

// cleanupFailedStart removes Kubernetes resources created during a failed start attempt
func cleanupFailedStart(ctx context.Context, k8sClient *kubernetes.Client, namespace, podName string, podCreated bool) {
	fmt.Println("\n⚠️  Start command failed. Cleaning up created resources...")

	if podCreated {
		fmt.Println("⏳ Deleting pod...")
		if err := k8sClient.DeletePod(ctx, podName, namespace); err != nil {
			fmt.Printf("⚠️  Warning: Failed to delete pod: %v\n", err)
			fmt.Printf("   Manual cleanup: kubectl delete pod %s -n %s\n", podName, namespace)
		} else {
			fmt.Println("✓ Pod deleted")
		}
	}

	fmt.Println("✓ Cleanup completed")
}

// determineCustomDirs returns the custom directories to sync
// Session-level custom dirs completely override global custom dirs
func determineCustomDirs(globalCfg *config.GlobalConfig, sessionCfg *config.SessionConfig) []config.CustomDirSync {
	// Session custom dirs override global custom dirs
	if len(sessionCfg.Sync.CustomDirs) > 0 {
		return sessionCfg.Sync.CustomDirs
	}
	return globalCfg.Sync.CustomDirs
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

// attachViaTtyd attaches to a session using ttyd (web-based terminal)
func attachViaTtyd(ctx context.Context, session *config.SessionConfig, opts AttachSessionOptions) error {
	// 1. Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(opts.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 2. Verify pod is running
	podStatus, err := k8sClient.GetPod(ctx, session.PodName, session.Namespace)
	if err != nil {
		return fmt.Errorf("pod not found: %w\n\nStart the session with:\n  kubectl kodama start %s", err, session.Name)
	}

	if !podStatus.Ready {
		return fmt.Errorf("pod is not ready (status: %s)\n\nCheck pod status:\n  kubectl get pod %s -n %s\n  kubectl describe pod %s -n %s",
			podStatus.Phase, session.PodName, session.Namespace, session.PodName, session.Namespace)
	}

	// 3. Determine ports
	remotePort := session.Ttyd.Port
	if remotePort == 0 {
		remotePort = 7681 // default ttyd port
	}

	localPort := opts.LocalPort
	if localPort == 0 {
		localPort = remotePort // use same port locally by default
	}

	// 4. Start port-forward
	fmt.Printf("Starting port-forward: localhost:%d -> %s:%d...\n", localPort, session.PodName, remotePort)

	portForwardCmd, err := k8sClient.StartPortForward(ctx, session.PodName, localPort, remotePort)
	if err != nil {
		return fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Ensure port-forward is cleaned up on exit
	defer func() {
		if portForwardCmd.Process != nil {
			_ = portForwardCmd.Process.Kill()
		}
	}()

	fmt.Println("✓ Port-forward established")

	// 5. Open browser if requested
	url := fmt.Sprintf("http://localhost:%d", localPort)
	if !opts.NoBrowser {
		fmt.Printf("Opening browser: %s\n", url)
		if err := openBrowser(url); err != nil {
			fmt.Printf("⚠️  Failed to open browser: %v\n", err)
			fmt.Printf("   Please open manually: %s\n", url)
		}
	} else {
		fmt.Printf("Access the terminal at: %s\n", url)
	}

	// 6. Wait for port-forward process to exit (Ctrl+C or process termination)
	fmt.Println("\nPress Ctrl+C to stop port-forward and exit")
	return portForwardCmd.Wait()
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	ctx := context.Background()

	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}
