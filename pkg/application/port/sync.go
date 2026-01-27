package port

import (
	"context"
	"time"

	"github.com/illumination-k/kodama/pkg/sync/exclude"
)

// SyncManager provides interface for managing file synchronization sessions
type SyncManager interface {
	// InitialSync performs one-time sync from local to pod
	InitialSync(ctx context.Context, localPath, namespace, podName string, excludeCfg *exclude.Config) error

	// InitialSyncToCustomPath performs one-time sync from local to custom path in pod
	InitialSyncToCustomPath(ctx context.Context, localPath, remotePath, namespace, podName string, excludeCfg *exclude.Config) error

	// Start creates a continuous sync session (for attach --sync)
	Start(ctx context.Context, sessionName, localPath, namespace, podName string, excludeCfg *exclude.Config) error

	// Stop terminates a sync session
	Stop(ctx context.Context, sessionName string) error

	// Status retrieves the status of a specific sync session
	Status(ctx context.Context, sessionName string) (*SyncStatus, error)
}

// SyncStatus represents the status of a sync session
type SyncStatus struct {
	Name       string
	Status     string // "watching", "syncing", "paused", "halted"
	LocalPath  string
	RemotePath string
	LastSync   time.Time
	Errors     []string
}
