package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

	// Expand recursive entries into individual subdirectory syncs
	expandedDirs, err := c.expandCustomDirs(customDirs, globalConfig)
	if err != nil {
		return fmt.Errorf("failed to expand custom directories: %w", err)
	}

	fmt.Printf("🔄 Syncing %d custom director%s...\n", len(expandedDirs), pluralize(len(expandedDirs)))

	successCount := 0
	for i, customDir := range expandedDirs {
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

	if successCount == 0 && len(expandedDirs) > 0 {
		return fmt.Errorf("failed to sync any custom directories")
	}

	fmt.Printf("✓ Successfully synced %d/%d custom director%s\n",
		successCount, len(expandedDirs), pluralize(len(expandedDirs)))

	return nil
}

// expandCustomDirs expands recursive directory entries into individual subdirectory syncs
func (c *CustomDirSyncManager) expandCustomDirs(
	customDirs []config.CustomDirSync,
	globalConfig *config.GlobalConfig,
) ([]config.CustomDirSync, error) {
	var result []config.CustomDirSync

	for _, dir := range customDirs {
		if !dir.Recursive {
			result = append(result, dir)
			continue
		}

		subdirs, err := c.discoverSubdirectories(dir, globalConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to expand recursive directory %s: %w", dir.Source, err)
		}

		result = append(result, subdirs...)
	}

	return result, nil
}

// discoverSubdirectories finds all immediate subdirectories of a recursive directory
func (c *CustomDirSyncManager) discoverSubdirectories(
	parentDir config.CustomDirSync,
	globalConfig *config.GlobalConfig,
) ([]config.CustomDirSync, error) {
	// 1. Resolve source path
	resolvedSource, err := parentDir.ResolveSource()
	if err != nil {
		return nil, err
	}

	// 2. Build exclude manager for filtering
	excludeCfg := c.buildExcludeConfig(parentDir, globalConfig)
	excludeCfg.BasePath = resolvedSource
	excludeMgr, err := exclude.NewManager(*excludeCfg)
	if err != nil {
		return nil, err
	}

	// 3. Read immediate subdirectories
	entries, err := os.ReadDir(resolvedSource)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// 4. Create CustomDirSync for each valid subdirectory
	result := make([]config.CustomDirSync, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files
		}

		subdirPath := filepath.Join(resolvedSource, entry.Name())

		// Check if subdirectory should be excluded
		if excludeMgr.ShouldExcludeDir(subdirPath) {
			continue
		}

		// Create new entry inheriting parent settings
		subdirSync := config.CustomDirSync{
			Source:       subdirPath,
			Destination:  filepath.Join(parentDir.Destination, entry.Name()),
			Exclude:      parentDir.Exclude,
			UseGitignore: parentDir.UseGitignore,
			Recursive:    false, // Do NOT recurse further
		}

		result = append(result, subdirSync)
	}

	return result, nil
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
