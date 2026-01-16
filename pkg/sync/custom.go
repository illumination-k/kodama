package sync

import (
	"context"
	"fmt"

	"github.com/illumination-k/kodama/pkg/config"
	"github.com/illumination-k/kodama/pkg/sync/exclude"
)

// CustomDirSyncManager orchestrates syncing of custom directories
type CustomDirSyncManager struct {
	syncMgr SyncManager
}

// NewCustomDirSyncManager creates a new CustomDirSyncManager
func NewCustomDirSyncManager(syncMgr SyncManager) *CustomDirSyncManager {
	return &CustomDirSyncManager{
		syncMgr: syncMgr,
	}
}

// SyncCustomDirs syncs all custom directories to the pod
func (c *CustomDirSyncManager) SyncCustomDirs(
	ctx context.Context,
	customDirs []config.CustomDirSync,
	namespace, podName string,
	globalConfig *config.GlobalConfig,
) error {
	if len(customDirs) == 0 {
		return nil
	}

	fmt.Printf("🔄 Syncing %d custom director%s...\n", len(customDirs), pluralize(len(customDirs)))

	successCount := 0
	for i, customDir := range customDirs {
		// Validate custom directory config
		if err := customDir.Validate(); err != nil {
			fmt.Printf("⚠️  Warning: Skipping custom directory %d: %v\n", i+1, err)
			continue
		}

		// Resolve source path
		resolvedSource, err := customDir.ResolveSource()
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to resolve source path '%s': %v\n", customDir.Source, err)
			continue
		}

		// Build exclude config for this directory
		excludeCfg := c.buildExcludeConfig(customDir, globalConfig)

		// Sync to custom path
		if err := c.syncMgr.InitialSyncToCustomPath(
			ctx,
			resolvedSource,
			customDir.Destination,
			namespace,
			podName,
			excludeCfg,
		); err != nil {
			fmt.Printf("⚠️  Warning: Failed to sync '%s' to '%s': %v\n",
				customDir.Source, customDir.Destination, err)
			continue
		}

		fmt.Printf("✓ Synced: %s → %s\n", customDir.Source, customDir.Destination)
		successCount++
	}

	if successCount == 0 && len(customDirs) > 0 {
		return fmt.Errorf("failed to sync any custom directories")
	}

	fmt.Printf("✓ Successfully synced %d/%d custom director%s\n",
		successCount, len(customDirs), pluralize(len(customDirs)))

	return nil
}

// buildExcludeConfig builds exclude configuration for a custom directory
// with proper inheritance from global config
func (c *CustomDirSyncManager) buildExcludeConfig(
	customDir config.CustomDirSync,
	globalConfig *config.GlobalConfig,
) *exclude.Config {
	// Start with global exclude patterns
	excludePatterns := make([]string, len(globalConfig.Sync.Exclude))
	copy(excludePatterns, globalConfig.Sync.Exclude)

	// Override with custom directory exclude patterns if specified
	if len(customDir.Exclude) > 0 {
		excludePatterns = customDir.Exclude
	}

	// Determine useGitignore setting
	useGitignore := false
	if globalConfig.Sync.UseGitignore != nil {
		useGitignore = *globalConfig.Sync.UseGitignore
	}

	// Override with custom directory setting if specified
	if customDir.UseGitignore != nil {
		useGitignore = *customDir.UseGitignore
	}

	return &exclude.Config{
		Patterns:     excludePatterns,
		UseGitignore: useGitignore,
	}
}

// pluralize returns "y" for count of 1, "ies" otherwise
func pluralize(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}
