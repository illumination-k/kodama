package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/usecase"
)

// NewDebugCommand creates a new debug command
func NewDebugCommand() *cobra.Command {
	var (
		// Configuration source
		fromConfig bool

		// Flags from start command (only used when fromConfig=false)
		repo            string
		syncPath        string
		namespace       string
		cpu             string
		memory          string
		customResources []string
		branch          string
		image           string
		command         string
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
		secretFiles     []string

		// Output options
		outputFormat string
		showSecrets  bool
	)

	cmd := &cobra.Command{
		Use:   "debug <name>",
		Short: "Generate Kubernetes manifests without creating resources",
		Long: `Debug command shows what manifests would be created by the start command
without actually creating any resources in the cluster.

This is useful for:
- Testing manifest generation logic
- Validating configuration
- Debugging issues before deployment
- CI/CD validation

By default, secret values are redacted. Use --show-secrets to reveal them.

Examples:
  # Generate manifests from flags
  kubectl kodama debug my-session --namespace dev --image myimage:latest

  # Generate from existing session config
  kubectl kodama debug my-session --from-config

  # Show actual secret values
  kubectl kodama debug my-session --namespace dev --env-file .env --show-secrets

  # Output as JSON
  kubectl kodama debug my-session --namespace dev -o json

  # Save to file
  kubectl kodama debug my-session --namespace dev > manifests.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]
			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")

			var opts usecase.StartSessionOptions

			if fromConfig {
				// Load from existing session config
				store, err := config.NewStore()
				if err != nil {
					return fmt.Errorf("failed to initialize config store: %w", err)
				}

				session, err := store.LoadSession(sessionName)
				if err != nil {
					return fmt.Errorf("failed to load session config: %w", err)
				}

				// Convert session config back to options
				opts = sessionConfigToOptions(session, kubeconfigPath)
			} else {
				// Build options from flags (same as start command)

				// Parse custom resources
				customResourcesMap := make(map[string]string)
				for _, res := range customResources {
					parts := strings.Split(res, "=")
					if len(parts) != 2 {
						return fmt.Errorf("invalid resource format: %s (expected format: resourceName=quantity, e.g., nvidia.com/gpu=1)", res)
					}
					customResourcesMap[parts[0]] = parts[1]
				}

				// Parse secret files
				secretFileMappings := make([]usecase.SecretFileMapping, 0, len(secretFiles))
				for _, mapping := range secretFiles {
					parts := strings.SplitN(mapping, ":", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid secret file format: %s (expected format: source:destination)", mapping)
					}
					secretFileMappings = append(secretFileMappings, usecase.SecretFileMapping{
						Source:      parts[0],
						Destination: parts[1],
					})
				}

				opts = usecase.StartSessionOptions{
					Name:            sessionName,
					Repo:            repo,
					SyncPath:        syncPath,
					Namespace:       namespace,
					CPU:             cpu,
					Memory:          memory,
					CustomResources: customResourcesMap,
					Branch:          branch,
					KubeconfigPath:  kubeconfigPath,
					Image:           image,
					Command:         command,
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
					SecretFiles:     secretFileMappings,
				}
			}

			// Enable dry-run mode
			opts.DryRun = true

			// Call StartSession with dry-run enabled
			session, err := usecase.StartSession(context.Background(), opts)
			if err != nil {
				return fmt.Errorf("failed to generate manifests: %w", err)
			}

			// Get manifests from session (populated by StartSession in dry-run mode)
			if session.ManifestsGenerated == nil {
				return fmt.Errorf("no manifests generated")
			}

			manifests, ok := session.ManifestsGenerated.(*usecase.ManifestCollection)
			if !ok {
				return fmt.Errorf("invalid manifests type")
			}

			// Apply secret redaction if not showing secrets
			if !showSecrets {
				manifests = usecase.RedactSecrets(manifests)
			}

			// Output manifests in requested format
			switch outputFormat {
			case "yaml":
				return usecase.WriteManifestsYAML(manifests, os.Stdout)
			case "json":
				return usecase.WriteManifestsJSON(manifests, os.Stdout)
			default:
				return fmt.Errorf("unsupported output format: %s (supported: yaml, json)", outputFormat)
			}
		},
	}

	// Configuration source flags
	cmd.Flags().BoolVar(&fromConfig, "from-config", false, "Load configuration from existing session instead of flags")

	// Output flags
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "yaml", "Output format (yaml|json)")
	cmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "Show actual secret values instead of redacting them")

	// Start command flags (only used when --from-config is not set)
	cmd.Flags().StringVar(&repo, "repo", "", "Git repository URL to clone")
	cmd.Flags().StringVar(&syncPath, "sync", "", "Local path to sync")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&cpu, "cpu", "", "CPU limit (e.g., '1', '2')")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory limit (e.g., '2Gi', '4Gi')")
	cmd.Flags().StringSliceVar(&customResources, "resource", []string{}, "Custom resource (e.g., --resource nvidia.com/gpu=1)")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to clone")
	cmd.Flags().StringVar(&image, "image", "", "Container image to use")
	cmd.Flags().StringVar(&command, "cmd", "", "Pod command override")
	cmd.Flags().IntVar(&cloneDepth, "clone-depth", 0, "Shallow clone depth (0 = full clone)")
	cmd.Flags().BoolVar(&singleBranch, "single-branch", false, "Clone only specified branch")
	cmd.Flags().StringVar(&gitCloneArgs, "git-clone-args", "", "Additional git clone arguments")
	cmd.Flags().StringVar(&configFile, "config", "", "Session template config file")
	cmd.Flags().BoolVar(&ttydEnabled, "ttyd", true, "Enable ttyd (web-based terminal)")
	cmd.Flags().IntVar(&ttydPort, "ttyd-port", 0, "Ttyd port (default: 7681)")
	cmd.Flags().StringVar(&ttydOptions, "ttyd-options", "", "Additional ttyd options")
	cmd.Flags().BoolVar(&ttydReadonly, "ttyd-readonly", false, "Enable read-only mode for ttyd")
	cmd.Flags().StringSliceVar(&envFiles, "env-file", []string{}, "Dotenv file(s) to load")
	cmd.Flags().StringSliceVar(&envExclude, "env-exclude", []string{}, "Environment variables to exclude")
	cmd.Flags().StringSliceVar(&secretFiles, "secret-file", []string{}, "Inject file as secret (format: source:destination)")

	return cmd
}

// sessionConfigToOptions converts a SessionConfig back to StartSessionOptions for dry-run
func sessionConfigToOptions(session *config.SessionConfig, kubeconfigPath string) usecase.StartSessionOptions {
	// Convert secret file mappings
	secretFileMappings := make([]usecase.SecretFileMapping, len(session.SecretFile.Files))
	for i, mapping := range session.SecretFile.Files {
		secretFileMappings[i] = usecase.SecretFileMapping{
			Source:      mapping.Source,
			Destination: mapping.Destination,
		}
	}

	return usecase.StartSessionOptions{
		Name:            session.Name,
		Repo:            session.Repo,
		SyncPath:        session.Sync.LocalPath,
		Namespace:       session.Namespace,
		CPU:             session.Resources.CPU,
		Memory:          session.Resources.Memory,
		CustomResources: session.Resources.CustomResources,
		Branch:          session.Branch,
		KubeconfigPath:  kubeconfigPath,
		Image:           session.Image,
		Command:         strings.Join(session.Command, " "),
		CloneDepth:      session.GitClone.Depth,
		SingleBranch:    session.GitClone.SingleBranch,
		GitCloneArgs:    session.GitClone.ExtraArgs,
		TtydEnabled:     session.Ttyd.Enabled != nil,
		TtydEnabledVal:  session.Ttyd.Enabled != nil && *session.Ttyd.Enabled,
		TtydPort:        session.Ttyd.Port,
		TtydOptions:     session.Ttyd.Options,
		TtydReadonly:    session.Ttyd.Writable != nil && !*session.Ttyd.Writable,
		TtydReadonlySet: session.Ttyd.Writable != nil,
		EnvFiles:        session.Env.DotenvFiles,
		EnvExclude:      session.Env.ExcludeVars,
		SecretFiles:     secretFileMappings,
	}
}
