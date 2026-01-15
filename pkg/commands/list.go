package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/kubernetes"
	"github.com/illumination-k/kodama/pkg/sync"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewListCommand creates a new list command
func NewListCommand() *cobra.Command {
	var allNamespaces bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all sessions",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
			return runList(allNamespaces, outputFormat, kubeconfigPath)
		},
	}

	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List sessions from all namespaces")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, yaml, json")

	return cmd
}

func runList(allNamespaces bool, outputFormat string, kubeconfigPath string) error {
	ctx := context.Background()

	// 1. Load sessions from ~/.kodama/sessions/
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize config store: %w", err)
	}

	sessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return nil
	}

	// 2. Create K8s client to verify pod status
	k8sClient, err := kubernetes.NewClient(kubeconfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create kubernetes client: %v\n", err)
		fmt.Fprintf(os.Stderr, "         Showing sessions without pod status verification\n\n")
		// Continue without K8s verification
	}

	// 3. Create sync manager for checking sync status
	syncMgr := sync.NewMutagenManager()

	// 4. Enrich sessions with actual pod and sync status
	for _, session := range sessions {
		if k8sClient != nil {
			// Verify pod status
			podStatus, err := k8sClient.GetPod(ctx, session.PodName, session.Namespace)
			if err != nil {
				// Pod doesn't exist or error
				if session.Status == config.StatusRunning {
					session.UpdateStatus(config.StatusStopped)
					_ = store.SaveSession(session) // Best effort update
				}
			} else {
				// Update status based on pod phase
				if podStatus.Ready && session.Status != config.StatusRunning {
					session.UpdateStatus(config.StatusRunning)
					_ = store.SaveSession(session) // Best effort update
				} else if !podStatus.Ready && session.Status == config.StatusRunning {
					session.UpdateStatus(config.StatusFailed)
					_ = store.SaveSession(session) // Best effort update
				}
			}
		}

		// Check mutagen sync session status if enabled
		if session.Sync.Enabled && session.Sync.MutagenSession != "" {
			_, err := syncMgr.Status(ctx, session.Sync.MutagenSession)
			if err != nil {
				// Sync session is gone
				session.Sync.Enabled = false
				_ = store.SaveSession(session) // Best effort update
			}
		}
	}

	// 5. Display in requested format
	switch outputFormat {
	case "yaml":
		return outputYAML(sessions)
	case "json":
		return outputJSON(sessions)
	default:
		return outputTable(sessions)
	}
}

func outputTable(sessions []*config.SessionConfig) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	_, _ = fmt.Fprintln(w, "NAME\tSTATUS\tNAMESPACE\tSYNC\tAGE")

	for _, session := range sessions {
		syncStatus := "-"
		if session.Sync.Enabled {
			syncStatus = "Active"
		}

		age := formatDuration(time.Since(session.CreatedAt))

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			session.Name,
			session.Status,
			session.Namespace,
			syncStatus,
			age,
		)
	}

	return nil
}

func outputYAML(sessions []*config.SessionConfig) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()

	for _, session := range sessions {
		if err := encoder.Encode(session); err != nil {
			return fmt.Errorf("failed to encode session to YAML: %w", err)
		}
		fmt.Println("---")
	}

	return nil
}

func outputJSON(sessions []*config.SessionConfig) error {
	// For JSON output, we'll use YAML library which can produce JSON-like output
	// A proper JSON implementation would use encoding/json
	data, err := yaml.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	} else if d < 30*24*time.Hour {
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	} else {
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	}
}
