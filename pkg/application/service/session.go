package service

import (
	"context"

	"github.com/illumination-k/kodama/pkg/application/port"
	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/kubernetes"
)

// SessionService provides high-level session operations using dependency injection
type SessionService struct {
	sessionRepo   port.SessionRepository
	configRepo    port.ConfigRepository
	k8sClient     port.KubernetesClient
	syncMgr       port.SyncManager
	agentExecutor port.AgentExecutor
}

// NewSessionService creates a new SessionService with injected dependencies
func NewSessionService(
	sessionRepo port.SessionRepository,
	configRepo port.ConfigRepository,
	k8sClient port.KubernetesClient,
	syncMgr port.SyncManager,
	agentExecutor port.AgentExecutor,
) *SessionService {
	return &SessionService{
		sessionRepo:   sessionRepo,
		configRepo:    configRepo,
		k8sClient:     k8sClient,
		syncMgr:       syncMgr,
		agentExecutor: agentExecutor,
	}
}

// GetSessionRepository returns the session repository
func (s *SessionService) GetSessionRepository() port.SessionRepository {
	return s.sessionRepo
}

// GetConfigRepository returns the config repository
func (s *SessionService) GetConfigRepository() port.ConfigRepository {
	return s.configRepo
}

// GetKubernetesClient returns the Kubernetes client
func (s *SessionService) GetKubernetesClient() port.KubernetesClient {
	return s.k8sClient
}

// GetSyncManager returns the sync manager
func (s *SessionService) GetSyncManager() port.SyncManager {
	return s.syncMgr
}

// GetAgentExecutor returns the agent executor
func (s *SessionService) GetAgentExecutor() port.AgentExecutor {
	return s.agentExecutor
}

// LoadSession loads a session configuration by name
func (s *SessionService) LoadSession(name string) (*config.SessionConfig, error) {
	return s.sessionRepo.LoadSession(name)
}

// SaveSession saves a session configuration
func (s *SessionService) SaveSession(session *config.SessionConfig) error {
	return s.sessionRepo.SaveSession(session)
}

// DeleteSessionConfig deletes a session configuration file
func (s *SessionService) DeleteSessionConfig(name string) error {
	return s.sessionRepo.DeleteSession(name)
}

// ListSessions returns all session configurations
func (s *SessionService) ListSessions() ([]*config.SessionConfig, error) {
	return s.sessionRepo.ListSessions()
}

// SessionExists checks if a session configuration exists
func (s *SessionService) SessionExists(name string) bool {
	return s.sessionRepo.SessionExists(name)
}

// DeletePod deletes a pod for a session
func (s *SessionService) DeletePod(ctx context.Context, podName, namespace string) error {
	return s.k8sClient.DeletePod(ctx, podName, namespace)
}

// DeleteSecret deletes a secret
func (s *SessionService) DeleteSecret(ctx context.Context, name, namespace string) error {
	return s.k8sClient.DeleteSecret(ctx, name, namespace)
}

// StopSync stops a sync session
func (s *SessionService) StopSync(ctx context.Context, sessionName string) error {
	return s.syncMgr.Stop(ctx, sessionName)
}

// LoadGlobalConfig loads the global configuration
func (s *SessionService) LoadGlobalConfig() (*config.GlobalConfig, error) {
	return s.configRepo.LoadGlobalConfig()
}

// GetPod retrieves the status of a pod
func (s *SessionService) GetPod(ctx context.Context, name, namespace string) (*kubernetes.PodStatus, error) {
	// Need to import kubernetes package
	return s.k8sClient.GetPod(ctx, name, namespace)
}
