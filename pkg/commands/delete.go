package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/kubernetes"
	"github.com/illumination-k/kodama/pkg/sync"
)

// NewDeleteCommand creates a new delete command
func NewDeleteCommand() *cobra.Command {
	var keepConfig bool
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a session",
		Long: `Delete a session by removing pod and optionally config.

Steps:
  1. Stop mutagen sync (if active)
  2. Delete Kubernetes pod
  3. Remove session config (unless --keep-config)

Examples:
  kubectl kodama delete my-work
  kubectl kodama delete my-work --keep-config
  kubectl kodama delete my-work --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
			return runDelete(args[0], keepConfig, force, kubeconfigPath)
		},
	}

	cmd.Flags().BoolVar(&keepConfig, "keep-config", false, "Keep session config file")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(name string, keepConfig, force bool, kubeconfigPath string) error {
	ctx := context.Background()

	// 1. Load session
	store, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to initialize config store: %w", err)
	}

	session, err := store.LoadSession(name)
	if err != nil {
		if errors.Is(err, config.ErrSessionNotFound) {
			return fmt.Errorf("session '%s' not found\n\nAvailable sessions:\n  kubectl kodama list", name)
		}
		return fmt.Errorf("failed to load session: %w", err)
	}

	// 2. Confirm deletion (unless --force)
	if !force {
		fmt.Printf("Delete session '%s'", name)
		if session.Sync.Enabled {
			fmt.Printf(" (sync: %s)", session.Sync.LocalPath)
		}
		fmt.Printf("? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, readErr := reader.ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("failed to read confirmation: %w", readErr)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Canceled")
			return nil
		}
	}

	// 3. Stop file sync
	if session.Sync.Enabled && session.Sync.MutagenSession != "" {
		fmt.Println("⏳ Stopping file sync...")
		syncMgr := sync.NewSyncManager()
		if syncErr := syncMgr.Stop(ctx, session.Sync.MutagenSession); syncErr != nil {
			fmt.Printf("⚠️  Warning: Failed to stop sync: %v\n", syncErr)
		} else {
			fmt.Println("✓ Sync stopped")
		}
	}

	// 4. Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(kubeconfigPath)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to create kubernetes client: %v\n", err)
	} else {
		// 4a. Delete environment secret if exists
		if session.Env.SecretCreated && session.Env.SecretName != "" {
			fmt.Println("🗑️  Deleting environment secret...")
			if err := k8sClient.DeleteSecret(ctx, session.Env.SecretName, session.Namespace); err != nil {
				fmt.Printf("⚠️  Warning: Failed to delete secret: %v\n", err)
			} else {
				fmt.Println("✓ Secret deleted")
			}
		}

		// 4b. Delete pod
		fmt.Println("⏳ Deleting pod...")
		if err := k8sClient.DeletePod(ctx, session.PodName, session.Namespace); err != nil {
			fmt.Printf("⚠️  Warning: Failed to delete pod: %v\n", err)
		} else {
			fmt.Println("✓ Pod deletion initiated")

			// Wait for pod to be fully deleted
			fmt.Println("⏳ Waiting for pod termination...")
			waitTimeout := 2 * time.Minute
			if err := k8sClient.WaitForPodDeleted(ctx, session.PodName, session.Namespace, waitTimeout); err != nil {
				fmt.Printf("⚠️  Warning: Failed to confirm pod deletion: %v\n", err)
			} else {
				fmt.Println("✓ Pod fully terminated and removed")
			}
		}
	}

	// 5. Delete session config (unless --keep-config)
	if !keepConfig {
		if err := store.DeleteSession(name); err != nil {
			return fmt.Errorf("failed to delete session config: %w", err)
		}
		fmt.Println("✓ Session config deleted")
	} else {
		session.UpdateStatus(config.StatusStopped)
		if err := store.SaveSession(session); err != nil {
			return fmt.Errorf("failed to update session status: %w", err)
		}
		fmt.Println("✓ Session config kept (status: Stopped)")
	}

	fmt.Printf("\n✨ Session '%s' deleted\n", name)

	return nil
}
