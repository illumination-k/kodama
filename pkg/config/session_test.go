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

func TestSessionConfig_RecordAgentExecution(t *testing.T) {
	config := &SessionConfig{
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	execution := AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "test prompt",
		TaskID:     "task-1",
		Status:     "completed",
	}

	oldTime := config.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	config.RecordAgentExecution(execution)

	assert.Len(t, config.AgentExecutions, 1)
	assert.Equal(t, "test prompt", config.AgentExecutions[0].Prompt)
	assert.Equal(t, "task-1", config.AgentExecutions[0].TaskID)
	assert.NotNil(t, config.LastAgentRun)
	assert.True(t, config.UpdatedAt.After(oldTime))
}

func TestSessionConfig_GetLastAgentExecution(t *testing.T) {
	config := &SessionConfig{}

	// No executions
	last := config.GetLastAgentExecution()
	assert.Nil(t, last)

	// Add one execution
	execution1 := AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "prompt 1",
		TaskID:     "task-1",
		Status:     "completed",
	}
	config.RecordAgentExecution(execution1)

	last = config.GetLastAgentExecution()
	assert.NotNil(t, last)
	assert.Equal(t, "prompt 1", last.Prompt)

	// Add another execution
	execution2 := AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "prompt 2",
		TaskID:     "task-2",
		Status:     "completed",
	}
	config.RecordAgentExecution(execution2)

	last = config.GetLastAgentExecution()
	assert.NotNil(t, last)
	assert.Equal(t, "prompt 2", last.Prompt)
}

func TestSessionConfig_HasPendingAgentTask(t *testing.T) {
	config := &SessionConfig{}

	// No executions
	assert.False(t, config.HasPendingAgentTask())

	// Add completed execution
	config.RecordAgentExecution(AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "test",
		Status:     "completed",
	})
	assert.False(t, config.HasPendingAgentTask())

	// Add pending execution
	config.RecordAgentExecution(AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "test 2",
		Status:     "pending",
	})
	assert.True(t, config.HasPendingAgentTask())

	// Add running execution
	config2 := &SessionConfig{}
	config2.RecordAgentExecution(AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "test 3",
		Status:     "running",
	})
	assert.True(t, config2.HasPendingAgentTask())

	// Add failed execution
	config3 := &SessionConfig{}
	config3.RecordAgentExecution(AgentExecution{
		ExecutedAt: time.Now(),
		Prompt:     "test 4",
		Status:     "failed",
	})
	assert.False(t, config3.HasPendingAgentTask())
}
