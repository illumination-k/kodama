package sync

import (
	"context"

	"github.com/illumination-k/kodama/pkg/application/port"
	"github.com/illumination-k/kodama/pkg/sync"
	"github.com/illumination-k/kodama/pkg/sync/exclude"
)

// Adapter implements port.SyncManager using the existing sync.SyncManager
type Adapter struct {
	manager sync.SyncManager
}

// NewAdapter creates a new sync adapter
func NewAdapter() port.SyncManager {
	return &Adapter{
		manager: sync.NewSyncManager(),
	}
}

// InitialSync performs one-time sync from local to pod
func (a *Adapter) InitialSync(ctx context.Context, localPath, namespace, podName string, excludeCfg *exclude.Config) error {
	return a.manager.InitialSync(ctx, localPath, namespace, podName, excludeCfg)
}

// InitialSyncToCustomPath performs one-time sync from local to custom path in pod
func (a *Adapter) InitialSyncToCustomPath(ctx context.Context, localPath, remotePath, namespace, podName string, excludeCfg *exclude.Config) error {
	return a.manager.InitialSyncToCustomPath(ctx, localPath, remotePath, namespace, podName, excludeCfg)
}

// Start creates a continuous sync session
func (a *Adapter) Start(ctx context.Context, sessionName, localPath, namespace, podName string, excludeCfg *exclude.Config) error {
	return a.manager.Start(ctx, sessionName, localPath, namespace, podName, excludeCfg)
}

// Stop terminates a sync session
func (a *Adapter) Stop(ctx context.Context, sessionName string) error {
	return a.manager.Stop(ctx, sessionName)
}

// Status retrieves the status of a specific sync session
func (a *Adapter) Status(ctx context.Context, sessionName string) (*port.SyncStatus, error) {
	status, err := a.manager.Status(ctx, sessionName)
	if err != nil {
		return nil, err
	}

	// Convert sync.SyncStatus to port.SyncStatus
	return &port.SyncStatus{
		Name:       status.Name,
		Status:     status.Status,
		LocalPath:  status.LocalPath,
		RemotePath: status.RemotePath,
		LastSync:   status.LastSync,
		Errors:     status.Errors,
	}, nil
}
