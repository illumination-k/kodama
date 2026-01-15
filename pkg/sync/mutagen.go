package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MutagenManager provides interface for managing mutagen sync sessions
type MutagenManager interface {
	CheckInstallation(ctx context.Context) error
	Start(ctx context.Context, sessionName, localPath, namespace, podName string) error
	Stop(ctx context.Context, sessionName string) error
	Status(ctx context.Context, sessionName string) (*SyncStatus, error)
	List(ctx context.Context) ([]*SyncStatus, error)
}

// SyncStatus represents the status of a sync session
type SyncStatus struct {
	Name       string
	Status     string // "watching", "syncing", "paused", "halted"
	LocalPath  string
	RemotePath string
	LastSync   time.Time
	Errors     []string
}

// mutagenManager implements MutagenManager interface
type mutagenManager struct{}

// NewMutagenManager creates a new MutagenManager instance
func NewMutagenManager() MutagenManager {
	return &mutagenManager{}
}

// CheckInstallation verifies that mutagen CLI is installed and available
func (m *mutagenManager) CheckInstallation(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "mutagen", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mutagen not found. Install from: https://mutagen.io/documentation/introduction/installation")
	}
	return nil
}

// Start creates a new mutagen sync session
func (m *mutagenManager) Start(ctx context.Context, sessionName, localPath, namespace, podName string) error {
	// Check mutagen is installed
	if err := m.CheckInstallation(ctx); err != nil {
		return err
	}

	// Build remote endpoint using kubectl exec
	// Mutagen can use docker:// protocol with custom exec commands
	remote := fmt.Sprintf("docker://exec:kubectl exec -i -n %s %s -- sh:/workspace", namespace, podName)

	// Create sync session with appropriate flags
	args := []string{
		"sync", "create",
		"--name", sessionName,
		"--ignore", ".git",
		"--ignore", "node_modules",
		"--ignore", ".kodama",
		"--ignore", "*.log",
		"--default-owner-beta", "root",
		"--default-group-beta", "root",
		"--sync-mode", "two-way-resolved",
		localPath,
		remote,
	}

	cmd := exec.CommandContext(ctx, "mutagen", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if session already exists
		if strings.Contains(string(output), "unable to create session") && strings.Contains(string(output), "name already in use") {
			return fmt.Errorf("sync session '%s' already exists. Use 'mutagen sync terminate %s' to remove it first", sessionName, sessionName)
		}
		return fmt.Errorf("failed to create sync session: %w (output: %s)", err, string(output))
	}

	return nil
}

// Stop terminates a mutagen sync session
func (m *mutagenManager) Stop(ctx context.Context, sessionName string) error {
	cmd := exec.CommandContext(ctx, "mutagen", "sync", "terminate", sessionName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "session not found" errors as they indicate the session is already gone
		if strings.Contains(string(output), "unable to locate") ||
			strings.Contains(string(output), "no sessions found") {
			return nil
		}
		return fmt.Errorf("failed to stop sync session: %w (output: %s)", err, string(output))
	}
	return nil
}

// Status retrieves the status of a specific sync session
func (m *mutagenManager) Status(ctx context.Context, sessionName string) (*SyncStatus, error) {
	cmd := exec.CommandContext(ctx, "mutagen", "sync", "list", sessionName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "no sessions found") ||
			strings.Contains(string(output), "unable to locate") {
			return nil, fmt.Errorf("sync session '%s' not found", sessionName)
		}
		return nil, fmt.Errorf("failed to get sync status: %w (output: %s)", err, string(output))
	}

	// Parse the output to extract status information
	// Mutagen list output format is human-readable by default
	status := &SyncStatus{
		Name:   sessionName,
		Status: "unknown",
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Status:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.Status = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Alpha:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.LocalPath = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Beta:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.RemotePath = strings.TrimSpace(parts[1])
			}
		}
	}

	// Check for error indicators
	if strings.Contains(strings.ToLower(outputStr), "error") ||
		strings.Contains(strings.ToLower(outputStr), "failed") {
		status.Errors = append(status.Errors, "Sync session has errors. Run 'mutagen sync list' for details.")
	}

	return status, nil
}

// List retrieves all mutagen sync sessions
func (m *mutagenManager) List(ctx context.Context) ([]*SyncStatus, error) {
	cmd := exec.CommandContext(ctx, "mutagen", "sync", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If no sessions exist, mutagen returns an error
		if strings.Contains(string(output), "no sessions found") {
			return []*SyncStatus{}, nil
		}
		return nil, fmt.Errorf("failed to list sync sessions: %w (output: %s)", err, string(output))
	}

	// Parse the list output
	// For MVP, we'll use a simple parser. Future versions could use JSON output.
	sessions := []*SyncStatus{}

	outputStr := string(output)
	sessionBlocks := strings.Split(outputStr, "Name:")

	for _, block := range sessionBlocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		status := &SyncStatus{
			Status: "unknown",
		}

		lines := strings.Split(block, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)

			// First line after split contains the name
			if i == 0 {
				status.Name = strings.TrimSpace(line)
				continue
			}

			if strings.HasPrefix(line, "Status:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					status.Status = strings.TrimSpace(parts[1])
				}
			}
		}

		if status.Name != "" {
			sessions = append(sessions, status)
		}
	}

	return sessions, nil
}

// mutagenListOutput represents the JSON output from mutagen sync list
// This structure is for future use when we switch to JSON output parsing
type mutagenListOutput struct {
	Sessions []mutagenSession `json:"sessions"`
}

type mutagenSession struct {
	Identifier string          `json:"identifier"`
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Alpha      mutagenEndpoint `json:"alpha"`
	Beta       mutagenEndpoint `json:"beta"`
	LastError  string          `json:"lastError,omitempty"`
}

type mutagenEndpoint struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

// convertToSyncStatus converts mutagen JSON output to SyncStatus
// This is for future use when we implement JSON parsing
func convertToSyncStatus(output mutagenListOutput) []*SyncStatus {
	statuses := make([]*SyncStatus, 0, len(output.Sessions))

	for _, session := range output.Sessions {
		status := &SyncStatus{
			Name:       session.Name,
			Status:     session.Status,
			LocalPath:  session.Alpha.Path,
			RemotePath: session.Beta.Path,
		}

		if session.LastError != "" {
			status.Errors = []string{session.LastError}
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// ParseMutagenJSON parses JSON output from mutagen (for future use)
func ParseMutagenJSON(data []byte) ([]*SyncStatus, error) {
	var output mutagenListOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse mutagen JSON output: %w", err)
	}

	return convertToSyncStatus(output), nil
}
