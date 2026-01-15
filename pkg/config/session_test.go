package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionConfig_Validate(t *testing.T) {
	tests := []struct {
		wantErr error
		config  *SessionConfig
		name    string
	}{
		{
			name: "valid config",
			config: &SessionConfig{
				Name:      "test-session",
				Namespace: "default",
				Repo:      "github.com/test/repo",
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			config: &SessionConfig{
				Namespace: "default",
				Repo:      "github.com/test/repo",
			},
			wantErr: ErrSessionNameRequired,
		},
		{
			name: "missing namespace",
			config: &SessionConfig{
				Name: "test-session",
				Repo: "github.com/test/repo",
			},
			wantErr: ErrNamespaceRequired,
		},
		{
			name: "missing repo (now optional)",
			config: &SessionConfig{
				Name:      "test-session",
				Namespace: "default",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSessionConfig_IsRunning(t *testing.T) {
	config := &SessionConfig{Status: StatusRunning}
	assert.True(t, config.IsRunning())

	config.Status = StatusStopped
	assert.False(t, config.IsRunning())

	config.Status = StatusPending
	assert.False(t, config.IsRunning())
}

func TestSessionConfig_IsStopped(t *testing.T) {
	config := &SessionConfig{Status: StatusStopped}
	assert.True(t, config.IsStopped())

	config.Status = StatusRunning
	assert.False(t, config.IsStopped())

	config.Status = StatusFailed
	assert.False(t, config.IsStopped())
}

func TestSessionConfig_UpdateStatus(t *testing.T) {
	config := &SessionConfig{
		Status:    StatusPending,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldTime := config.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	config.UpdateStatus(StatusRunning)

	assert.Equal(t, StatusRunning, config.Status)
	assert.True(t, config.UpdatedAt.After(oldTime))
}
