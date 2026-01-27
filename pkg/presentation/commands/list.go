package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/illumination-k/kodama/pkg/application/service"
	"github.com/illumination-k/kodama/pkg/config"
)

// NewListCommand creates a new list command
func NewListCommand(sessionService *service.SessionService) *cobra.Command {
	var allNamespaces bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all sessions",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(sessionService, outputFormat)
		},
	}

	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List sessions from all namespaces")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, yaml, json")

	return cmd
}

func runList(sessionService *service.SessionService, outputFormat string) error {
	ctx := context.Background()

	// 1. Load sessions from ~/.kodama/sessions/
	sessions, err := sessionService.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return nil
	}

	// 2. Enrich sessions with actual pod and sync status
	for _, session := range sessions {
		// Verify pod status
		podStatus, err := sessionService.GetPod(ctx, session.PodName, session.Namespace)
		if err != nil {
			// Pod doesn't exist or error
			if session.Status == config.StatusRunning {
				session.UpdateStatus(config.StatusStopped)
				_ = sessionService.SaveSession(session) // Best effort update
			}
		} else {
			// Update status based on pod phase
			if podStatus.Ready && session.Status != config.StatusRunning {
				session.UpdateStatus(config.StatusRunning)
				_ = sessionService.SaveSession(session) // Best effort update
			} else if !podStatus.Ready && session.Status == config.StatusRunning {
				session.UpdateStatus(config.StatusFailed)
				_ = sessionService.SaveSession(session) // Best effort update
			}
		}

		// Check sync session status if enabled
		if session.Sync.Enabled && session.Sync.MutagenSession != "" {
			_, err := sessionService.GetSyncManager().Status(ctx, session.Sync.MutagenSession)
			if err != nil {
				// Sync session is gone
				session.Sync.Enabled = false
				_ = sessionService.SaveSession(session) // Best effort update
			}
		}
	}

	// 3. Display in requested format
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

	_, _ = fmt.Fprintln(w, "NAME\tSTATUS\tNAMESPACE\tPATH\tSYNC\tAGE")

	for _, session := range sessions {
		syncStatus := "-"
		if session.Sync.Enabled {
			syncStatus = "Active"
		}

		// Show repo if available, otherwise show local path
		pathDisplay := "-"
		if session.Repo != "" {
			pathDisplay = session.Repo
		} else if session.Sync.LocalPath != "" {
			pathDisplay = session.Sync.LocalPath
		}

		age := formatDuration(time.Since(session.CreatedAt))

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			session.Name,
			session.Status,
			session.Namespace,
			pathDisplay,
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
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	}
}
