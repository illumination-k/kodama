package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// CommandExecutor abstracts command execution for testing
type CommandExecutor interface {
	// ExecInPod executes a command inside a Kubernetes pod
	ExecInPod(ctx context.Context, namespace, podName string, command []string) (stdout, stderr string, err error)
}

// KubectlExecutor implements CommandExecutor using kubectl exec
type KubectlExecutor struct{}

// NewKubectlExecutor creates a new KubectlExecutor
func NewKubectlExecutor() CommandExecutor {
	return &KubectlExecutor{}
}

// ExecInPod executes a command inside a Kubernetes pod using kubectl exec
func (k *KubectlExecutor) ExecInPod(ctx context.Context, namespace, podName string, command []string) (string, string, error) {
	args := []string{"exec", "-n", namespace, podName, "--"}
	args = append(args, command...)

	//#nosec G204 -- kubectl is a known command, args are controlled
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("command failed: %w", err)
	}

	return stdout.String(), stderr.String(), nil
}
