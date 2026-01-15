package sync

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// simpleSyncManager implements SyncManager interface using fsnotify + kubectl cp
type simpleSyncManager struct {
	watchers map[string]*fsnotify.Watcher
	stopChan map[string]chan struct{}
}

// Compile-time check that simpleSyncManager implements SyncManager
var _ SyncManager = (*simpleSyncManager)(nil)

// NewSimpleSyncManager creates a new SyncManager instance using simple sync
func NewSimpleSyncManager() SyncManager {
	return &simpleSyncManager{
		watchers: make(map[string]*fsnotify.Watcher),
		stopChan: make(map[string]chan struct{}),
	}
}

// InitialSync performs one-time sync from local to pod
func (s *simpleSyncManager) InitialSync(ctx context.Context, localPath, namespace, podName string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Verify directory exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("local path does not exist: %w", err)
	}

	return s.initialSync(ctx, absPath, namespace, podName)
}

// Start creates a new sync session using kubectl cp and fsnotify
func (s *simpleSyncManager) Start(ctx context.Context, sessionName, localPath, namespace, podName string) error {
	// Check if session already exists
	if _, exists := s.watchers[sessionName]; exists {
		return fmt.Errorf("sync session '%s' already exists", sessionName)
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Verify directory exists
	if _, statErr := os.Stat(absPath); statErr != nil {
		return fmt.Errorf("local path does not exist: %w", statErr)
	}

	// Initial sync: copy all files to pod
	fmt.Println("🔄 Performing initial sync...")
	if syncErr := s.initialSync(ctx, absPath, namespace, podName); syncErr != nil {
		return fmt.Errorf("initial sync failed: %w", syncErr)
	}
	fmt.Println("✓ Initial sync completed")

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Add directory to watcher (recursively)
	if err := s.addDirRecursive(watcher, absPath); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Create stop channel
	stopChan := make(chan struct{})

	// Store watcher and stop channel
	s.watchers[sessionName] = watcher
	s.stopChan[sessionName] = stopChan

	// Start watching in background
	go s.watchFiles(ctx, absPath, namespace, podName, watcher, stopChan)

	return nil
}

// initialSync performs initial sync of all files
func (s *simpleSyncManager) initialSync(ctx context.Context, localPath, namespace, podName string) error {
	// Use tar + kubectl exec for efficient initial sync
	remotePath := "/workspace"

	// Create tar archive excluding common ignore patterns
	tarCmd := exec.CommandContext(ctx, "tar", "czf", "-",
		"--exclude=.git",
		"--exclude=node_modules",
		"--exclude=.kodama",
		"--exclude=*.log",
		"-C", localPath,
		".",
	)

	// Pipe to kubectl exec to extract in pod
	untarCmd := exec.CommandContext(ctx, "kubectl", "exec", "-i",
		"-n", namespace,
		podName,
		"--",
		"tar", "xzf", "-", "-C", remotePath,
	)

	// Connect pipes
	pipe, err := tarCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	untarCmd.Stdin = pipe

	// Start both commands
	if err := tarCmd.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}

	if err := untarCmd.Start(); err != nil {
		_ = tarCmd.Process.Kill()
		return fmt.Errorf("failed to start kubectl exec: %w", err)
	}

	// Wait for completion
	if err := tarCmd.Wait(); err != nil {
		return fmt.Errorf("tar command failed: %w", err)
	}

	if err := untarCmd.Wait(); err != nil {
		return fmt.Errorf("kubectl exec failed: %w", err)
	}

	return nil
}

// addDirRecursive adds directory and subdirectories to watcher
func (s *simpleSyncManager) addDirRecursive(watcher *fsnotify.Watcher, path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if info.IsDir() {
			base := filepath.Base(walkPath)
			if base == ".git" || base == "node_modules" || base == ".kodama" {
				return filepath.SkipDir
			}

			return watcher.Add(walkPath)
		}

		return nil
	})
}

// watchFiles monitors file changes and syncs to pod
func (s *simpleSyncManager) watchFiles(ctx context.Context, localPath, namespace, podName string, watcher *fsnotify.Watcher, stopChan chan struct{}) {
	// Debounce timer to batch rapid changes
	var timer *time.Timer
	pendingFiles := make(map[string]bool)

	syncPending := func() {
		if len(pendingFiles) == 0 {
			return
		}

		// Copy pending files to pod
		for file := range pendingFiles {
			relPath, err := filepath.Rel(localPath, file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get relative path for %s: %v\n", file, err)
				continue
			}

			remotePath := filepath.Join("/workspace", relPath)
			remoteDir := filepath.Dir(remotePath)

			// Create parent directory in pod if needed
			//#nosec G204 -- kubectl exec with namespace/pod from session config
			mkdirCmd := exec.CommandContext(ctx, "kubectl", "exec",
				"-n", namespace,
				podName,
				"--",
				"mkdir", "-p", remoteDir,
			)
			if err := mkdirCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create directory %s: %v\n", remoteDir, err)
			}

			// Copy file to pod
			//#nosec G204 -- kubectl cp with namespace/pod from session config
			cpCmd := exec.CommandContext(ctx, "kubectl", "cp",
				"-n", namespace,
				file,
				fmt.Sprintf("%s:%s", podName, remotePath),
			)

			if output, err := cpCmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to copy %s: %v (output: %s)\n", relPath, err, string(output))
			} else {
				fmt.Printf("📤 Synced: %s\n", relPath)
			}
		}

		// Clear pending files
		pendingFiles = make(map[string]bool)
	}

	for {
		select {
		case <-stopChan:
			return

		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only handle writes and creates
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Skip ignored files
				if strings.Contains(event.Name, ".git/") ||
					strings.Contains(event.Name, "node_modules/") ||
					strings.Contains(event.Name, ".kodama/") ||
					strings.HasSuffix(event.Name, ".log") {
					continue
				}

				// Add to pending files
				pendingFiles[event.Name] = true

				// Reset debounce timer
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(300*time.Millisecond, syncPending)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// Stop terminates a sync session
func (s *simpleSyncManager) Stop(ctx context.Context, sessionName string) error {
	watcher, exists := s.watchers[sessionName]
	if !exists {
		// Session not found is considered success
		return nil
	}

	// Close stop channel
	if stopChan, ok := s.stopChan[sessionName]; ok {
		close(stopChan)
		delete(s.stopChan, sessionName)
	}

	// Close watcher
	if err := watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	delete(s.watchers, sessionName)
	return nil
}

// Status retrieves the status of a sync session
func (s *simpleSyncManager) Status(ctx context.Context, sessionName string) (*SyncStatus, error) {
	_, exists := s.watchers[sessionName]
	if !exists {
		return nil, fmt.Errorf("sync session '%s' not found", sessionName)
	}

	return &SyncStatus{
		Name:   sessionName,
		Status: "watching",
	}, nil
}
