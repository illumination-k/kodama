package kubernetes

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"
)

// StartPortForward starts kubectl port-forward and waits for it to be ready
func (c *Client) StartPortForward(ctx context.Context, podName string, localPort, remotePort int) (*exec.Cmd, error) {
	// Construct the kubectl port-forward command
	args := []string{
		"port-forward",
		"-n", c.config.Namespace,
		podName,
		fmt.Sprintf("%d:%d", localPort, remotePort),
	}

	//#nosec G204 -- kubectl port-forward with validated session data from config
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	// Start the port-forward in the background
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Wait for port-forward to be ready
	if err := waitForPortForward(localPort, 30*time.Second); err != nil {
		// Kill the process if it failed to become ready
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("port-forward failed to become ready: %w", err)
	}

	return cmd, nil
}

// waitForPortForward polls the local port until it's ready or times out
func waitForPortForward(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		// Try to connect to the local port
		dialer := &net.Dialer{Timeout: 1 * time.Second}
		conn, err := dialer.DialContext(context.Background(), "tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			_ = conn.Close()
			return nil
		}

		// Wait before next retry
		<-ticker.C
	}

	return fmt.Errorf("timeout waiting for port %d to become ready", port)
}
