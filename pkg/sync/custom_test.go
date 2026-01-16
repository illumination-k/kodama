package sync

import (
	"context"
	"os"
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

func TestCustomDirSyncManager_expandCustomDirs(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create test directory structure:
	// tmpDir/
	//   configs/
	//     vim/
	//     shell/
	//     git/
	//     .hidden/
	//     file.txt (should be skipped as it's not a directory)
	configsDir := tmpDir + "/configs"
	if err := os.MkdirAll(configsDir+"/vim", 0o750); err != nil {
		t.Fatalf("Failed to create vim directory: %v", err)
	}
	if err := os.MkdirAll(configsDir+"/shell", 0o750); err != nil {
		t.Fatalf("Failed to create shell directory: %v", err)
	}
	if err := os.MkdirAll(configsDir+"/git", 0o750); err != nil {
		t.Fatalf("Failed to create git directory: %v", err)
	}
	if err := os.MkdirAll(configsDir+"/.hidden", 0o750); err != nil {
		t.Fatalf("Failed to create hidden directory: %v", err)
	}
	if err := os.WriteFile(configsDir+"/file.txt", []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create empty directory for testing
	emptyDir := tmpDir + "/empty"
	if err := os.MkdirAll(emptyDir, 0o750); err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	mockMgr := newMockSyncManager()
	customMgr := NewCustomDirSyncManager(mockMgr)
	globalConfig := config.DefaultGlobalConfig()

	t.Run("non-recursive entries pass through unchanged", func(t *testing.T) {
		dirs := []config.CustomDirSync{
			{
				Source:      configsDir + "/vim",
				Destination: "/root/.config/vim",
				Recursive:   false,
			},
			{
				Source:      configsDir + "/shell",
				Destination: "/root/.config/shell",
				Recursive:   false,
			},
		}

		expanded, err := customMgr.expandCustomDirs(dirs, globalConfig)
		if err != nil {
			t.Fatalf("expandCustomDirs failed: %v", err)
		}

		if len(expanded) != 2 {
			t.Errorf("Expected 2 entries, got %d", len(expanded))
		}

		// Check that entries are unchanged
		if expanded[0].Source != dirs[0].Source || expanded[0].Destination != dirs[0].Destination {
			t.Errorf("First entry changed unexpectedly")
		}
		if expanded[1].Source != dirs[1].Source || expanded[1].Destination != dirs[1].Destination {
			t.Errorf("Second entry changed unexpectedly")
		}
	})

	t.Run("recursive entry expands to subdirectories", func(t *testing.T) {
		dirs := []config.CustomDirSync{
			{
				Source:      configsDir,
				Destination: "/root/.config",
				Recursive:   true,
			},
		}

		expanded, err := customMgr.expandCustomDirs(dirs, globalConfig)
		if err != nil {
			t.Fatalf("expandCustomDirs failed: %v", err)
		}

		// Should expand to 4 subdirectories (vim, shell, git, .hidden)
		// file.txt should be skipped
		if len(expanded) != 4 {
			t.Errorf("Expected 4 expanded entries, got %d", len(expanded))
		}

		// Check that all entries are non-recursive
		for i, entry := range expanded {
			if entry.Recursive {
				t.Errorf("Entry %d should not be recursive", i)
			}
		}

		// Check that subdirectories were created correctly
		expectedDirs := map[string]string{
			configsDir + "/vim":     "/root/.config/vim",
			configsDir + "/shell":   "/root/.config/shell",
			configsDir + "/git":     "/root/.config/git",
			configsDir + "/.hidden": "/root/.config/.hidden",
		}

		for _, entry := range expanded {
			expectedDest, found := expectedDirs[entry.Source]
			if !found {
				t.Errorf("Unexpected source: %s", entry.Source)
			}
			if entry.Destination != expectedDest {
				t.Errorf("Source %s: expected destination %s, got %s",
					entry.Source, expectedDest, entry.Destination)
			}
		}
	})

	t.Run("empty recursive directory returns empty list", func(t *testing.T) {
		dirs := []config.CustomDirSync{
			{
				Source:      emptyDir,
				Destination: "/root/empty",
				Recursive:   true,
			},
		}

		expanded, err := customMgr.expandCustomDirs(dirs, globalConfig)
		if err != nil {
			t.Fatalf("expandCustomDirs failed: %v", err)
		}

		if len(expanded) != 0 {
			t.Errorf("Expected 0 entries for empty directory, got %d", len(expanded))
		}
	})

	t.Run("exclude patterns filter subdirectories", func(t *testing.T) {
		dirs := []config.CustomDirSync{
			{
				Source:      configsDir,
				Destination: "/root/.config",
				Recursive:   true,
				Exclude:     []string{"vim", "shell"},
			},
		}

		expanded, err := customMgr.expandCustomDirs(dirs, globalConfig)
		if err != nil {
			t.Fatalf("expandCustomDirs failed: %v", err)
		}

		// Should only have git and .hidden (vim and shell excluded)
		if len(expanded) != 2 {
			t.Errorf("Expected 2 entries after exclusion, got %d", len(expanded))
		}

		// Verify excluded directories are not present
		for _, entry := range expanded {
			if entry.Source == configsDir+"/vim" || entry.Source == configsDir+"/shell" {
				t.Errorf("Excluded directory found in results: %s", entry.Source)
			}
		}
	})

	t.Run("settings inheritance", func(t *testing.T) {
		useGitignoreFalse := false
		dirs := []config.CustomDirSync{
			{
				Source:       configsDir,
				Destination:  "/root/.config",
				Recursive:    true,
				Exclude:      []string{"*.backup"},
				UseGitignore: &useGitignoreFalse,
			},
		}

		expanded, err := customMgr.expandCustomDirs(dirs, globalConfig)
		if err != nil {
			t.Fatalf("expandCustomDirs failed: %v", err)
		}

		// Check that all expanded entries inherit parent settings
		for i, entry := range expanded {
			if len(entry.Exclude) != 1 || entry.Exclude[0] != "*.backup" {
				t.Errorf("Entry %d did not inherit exclude patterns", i)
			}
			if entry.UseGitignore == nil || *entry.UseGitignore != false {
				t.Errorf("Entry %d did not inherit UseGitignore setting", i)
			}
			if entry.Recursive {
				t.Errorf("Entry %d should not be recursive", i)
			}
		}
	})
}
