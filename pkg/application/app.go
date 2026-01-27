package application

import (
	"fmt"

	"github.com/illumination-k/kodama/pkg/application/service"
	agentAdapter "github.com/illumination-k/kodama/pkg/infrastructure/agent"
	kubernetesAdapter "github.com/illumination-k/kodama/pkg/infrastructure/kubernetes"
	"github.com/illumination-k/kodama/pkg/infrastructure/repository"
	syncAdapter "github.com/illumination-k/kodama/pkg/infrastructure/sync"
)

// App holds all application services and dependencies
type App struct {
	SessionService *service.SessionService
}

// NewApp creates and wires up the entire application with all dependencies
func NewApp(kubeconfigPath string) (*App, error) {
	// Create infrastructure adapters
	k8sClient, err := kubernetesAdapter.NewAdapter(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	syncMgr := syncAdapter.NewAdapter()
	agentExec := agentAdapter.NewAdapter()

	sessionRepo, err := repository.NewSessionFileRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create session repository: %w", err)
	}

	configRepo, err := repository.NewConfigFileRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create config repository: %w", err)
	}

	// Wire services
	sessionService := service.NewSessionService(
		sessionRepo,
		configRepo,
		k8sClient,
		syncMgr,
		agentExec,
	)

	return &App{
		SessionService: sessionService,
	}, nil
}
