package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_EnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	err := store.EnsureConfigDir()
	require.NoError(t, err)

	// Check main directory exists
	_, err = os.Stat(tmpDir)
	assert.NoError(t, err)

	// Check sessions subdirectory exists
	sessionsDir := filepath.Join(tmpDir, SessionsSubdir)
	_, err = os.Stat(sessionsDir)
	assert.NoError(t, err)
}

func TestStore_SaveAndLoadSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	session := &SessionConfig{
		Name:      "test-session",
		Namespace: "default",
		Repo:      "github.com/test/repo",
		Branch:    "main",
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save session
	err := store.SaveSession(session)
	require.NoError(t, err)

	// Load session
	loaded, err := store.LoadSession("test-session")
	require.NoError(t, err)

	assert.Equal(t, session.Name, loaded.Name)
	assert.Equal(t, session.Namespace, loaded.Namespace)
	assert.Equal(t, session.Repo, loaded.Repo)
	assert.Equal(t, session.Branch, loaded.Branch)
	assert.Equal(t, session.Status, loaded.Status)
}

func TestStore_LoadSession_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	_, err := store.LoadSession("nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestStore_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	// Create multiple sessions
	sessions := []*SessionConfig{
		{Name: "session1", Namespace: "default", Repo: "repo1"},
		{Name: "session2", Namespace: "default", Repo: "repo2"},
		{Name: "session3", Namespace: "default", Repo: "repo3"},
	}

	for _, session := range sessions {
		err := store.SaveSession(session)
		require.NoError(t, err)
	}

	// List sessions
	loaded, err := store.ListSessions()
	require.NoError(t, err)
	assert.Len(t, loaded, 3)

	// Verify session names
	names := make(map[string]bool)
	for _, s := range loaded {
		names[s.Name] = true
	}
	assert.True(t, names["session1"])
	assert.True(t, names["session2"])
	assert.True(t, names["session3"])
}

func TestStore_ListSessions_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	// List when directory doesn't exist
	sessions, err := store.ListSessions()
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestStore_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	session := &SessionConfig{
		Name:      "test-session",
		Namespace: "default",
		Repo:      "github.com/test/repo",
	}

	// Save and then delete
	err := store.SaveSession(session)
	require.NoError(t, err)

	err = store.DeleteSession("test-session")
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.LoadSession("test-session")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestStore_DeleteSession_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	err := store.DeleteSession("nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestStore_LoadGlobalConfig_WithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	// Load when no config exists (should return defaults)
	config, err := store.LoadGlobalConfig()
	require.NoError(t, err)

	assert.NotNil(t, config)
	assert.Equal(t, "default", config.Defaults.Namespace)
	assert.NotEmpty(t, config.Defaults.Image)
	assert.Equal(t, "1", config.Defaults.Resources.CPU)
	assert.Equal(t, "2Gi", config.Defaults.Resources.Memory)
}

func TestStore_SaveAndLoadGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	config := &GlobalConfig{
		Defaults: DefaultsConfig{
			Namespace: "custom-namespace",
			Image:     "custom-image:latest",
			Resources: ResourceConfig{
				CPU:    "2",
				Memory: "4Gi",
			},
			Storage: StorageConfig{
				Workspace:  "20Gi",
				ClaudeHome: "2Gi",
			},
			BranchPrefix: "feature/",
		},
		Git: GitConfig{
			SecretName: "my-secret",
		},
	}

	// Save config
	err := store.SaveGlobalConfig(config)
	require.NoError(t, err)

	// Load config
	loaded, err := store.LoadGlobalConfig()
	require.NoError(t, err)

	assert.Equal(t, "custom-namespace", loaded.Defaults.Namespace)
	assert.Equal(t, "custom-image:latest", loaded.Defaults.Image)
	assert.Equal(t, "2", loaded.Defaults.Resources.CPU)
	assert.Equal(t, "4Gi", loaded.Defaults.Resources.Memory)
	assert.Equal(t, "my-secret", loaded.Git.SecretName)
}

func TestStore_SessionExists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	// Session doesn't exist
	assert.False(t, store.SessionExists("nonexistent"))

	// Create session
	session := &SessionConfig{
		Name:      "test-session",
		Namespace: "default",
		Repo:      "github.com/test/repo",
	}
	err := store.SaveSession(session)
	require.NoError(t, err)

	// Session exists
	assert.True(t, store.SessionExists("test-session"))
}

func TestStore_GetSessionPath(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	path := store.GetSessionPath("my-session")
	expected := filepath.Join(tmpDir, SessionsSubdir, "my-session.yaml")

	assert.Equal(t, expected, path)
}

func TestStore_GetGlobalConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	path := store.GetGlobalConfigPath()
	expected := filepath.Join(tmpDir, GlobalConfigFile)

	assert.Equal(t, expected, path)
}

func TestStore_SaveSession_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithPath(tmpDir)

	// Session with missing required fields
	session := &SessionConfig{
		Name: "test-session",
		// Missing Namespace and Repo
	}

	err := store.SaveSession(session)
	assert.Error(t, err)
}
