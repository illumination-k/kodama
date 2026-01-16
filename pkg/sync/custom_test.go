package sync

import (
	"context"
	"testing"

	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/sync/exclude"
)

// mockSyncManager implements SyncManager for testing
type mockSyncManager struct {
	syncedPaths map[string]string // localPath -> remotePath
}

func newMockSyncManager() *mockSyncManager {
	return &mockSyncManager{
		syncedPaths: make(map[string]string),
	}
}

func (m *mockSyncManager) InitialSync(ctx context.Context, localPath, namespace, podName string, excludeCfg *exclude.Config) error {
	m.syncedPaths[localPath] = "/workspace"
	return nil
}

func (m *mockSyncManager) InitialSyncToCustomPath(ctx context.Context, localPath, remotePath, namespace, podName string, excludeCfg *exclude.Config) error {
	m.syncedPaths[localPath] = remotePath
	return nil
}

func (m *mockSyncManager) Start(ctx context.Context, sessionName, localPath, namespace, podName string, excludeCfg *exclude.Config) error {
	return nil
}

func (m *mockSyncManager) Stop(ctx context.Context, sessionName string) error {
	return nil
}

func (m *mockSyncManager) Status(ctx context.Context, sessionName string) (*SyncStatus, error) {
	return nil, nil
}

func TestCustomDirSyncManager_SyncCustomDirs_Empty(t *testing.T) {
	mockMgr := newMockSyncManager()
	customMgr := NewCustomDirSyncManager(mockMgr)

	ctx := context.Background()
	globalConfig := config.DefaultGlobalConfig()

	err := customMgr.SyncCustomDirs(ctx, []config.CustomDirSync{}, "default", "test-pod", globalConfig)
	if err != nil {
		t.Errorf("Expected no error for empty custom dirs, got: %v", err)
	}

	if len(mockMgr.syncedPaths) != 0 {
		t.Errorf("Expected 0 synced paths, got: %d", len(mockMgr.syncedPaths))
	}
}

func TestCustomDirSyncManager_buildExcludeConfig_Inheritance(t *testing.T) {
	mockMgr := newMockSyncManager()
	customMgr := NewCustomDirSyncManager(mockMgr)

	useGitignoreTrue := true
	useGitignoreFalse := false

	tests := []struct {
		name             string
		globalConfig     *config.GlobalConfig
		customDir        config.CustomDirSync
		wantPatterns     []string
		wantUseGitignore bool
	}{
		{
			name: "use global patterns",
			globalConfig: &config.GlobalConfig{
				Sync: config.GlobalSyncConfig{
					Exclude:      []string{"*.log", "*.tmp"},
					UseGitignore: &useGitignoreTrue,
				},
			},
			customDir: config.CustomDirSync{
				Source:      "~/test",
				Destination: "/root/test",
			},
			wantPatterns:     []string{"*.log", "*.tmp"},
			wantUseGitignore: true,
		},
		{
			name: "override with custom patterns",
			globalConfig: &config.GlobalConfig{
				Sync: config.GlobalSyncConfig{
					Exclude:      []string{"*.log"},
					UseGitignore: &useGitignoreTrue,
				},
			},
			customDir: config.CustomDirSync{
				Source:      "~/test",
				Destination: "/root/test",
				Exclude:     []string{"*.bak"},
			},
			wantPatterns:     []string{"*.bak"},
			wantUseGitignore: true,
		},
		{
			name: "override gitignore setting",
			globalConfig: &config.GlobalConfig{
				Sync: config.GlobalSyncConfig{
					Exclude:      []string{"*.log"},
					UseGitignore: &useGitignoreTrue,
				},
			},
			customDir: config.CustomDirSync{
				Source:       "~/test",
				Destination:  "/root/test",
				UseGitignore: &useGitignoreFalse,
			},
			wantPatterns:     []string{"*.log"},
			wantUseGitignore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excludeCfg := customMgr.buildExcludeConfig(tt.customDir, tt.globalConfig)

			if len(excludeCfg.Patterns) != len(tt.wantPatterns) {
				t.Errorf("Expected %d patterns, got %d", len(tt.wantPatterns), len(excludeCfg.Patterns))
			}

			for i, pattern := range tt.wantPatterns {
				if excludeCfg.Patterns[i] != pattern {
					t.Errorf("Pattern[%d] = %q, want %q", i, excludeCfg.Patterns[i], pattern)
				}
			}

			if excludeCfg.UseGitignore != tt.wantUseGitignore {
				t.Errorf("UseGitignore = %v, want %v", excludeCfg.UseGitignore, tt.wantUseGitignore)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{1, "y"},
		{0, "ies"},
		{2, "ies"},
		{100, "ies"},
	}

	for _, tt := range tests {
		got := pluralize(tt.count)
		if got != tt.want {
			t.Errorf("pluralize(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}
