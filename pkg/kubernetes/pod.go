package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/illumination-k/kodama/pkg/gitcmd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/ptr"
)

// createClaudeInstallerInitContainer creates init container for Claude Code installation
func createClaudeInstallerInitContainer() corev1.Container {
	return corev1.Container{
		Name:    "claude-installer",
		Image:   "ubuntu:24.04",
		Command: []string{"/bin/bash", "-c"},
		Args: []string{
			`set -e
echo "Installing Claude Code CLI..."
apt-get update -qq && apt-get install -y -qq curl ca-certificates
curl -fsSL https://claude.ai/install.sh | bash -s latest
echo "Copying binaries to /kodama/bin..."
mkdir -p /kodama/bin
cp -rL /root/.local/bin/* /kodama/bin/
echo "Claude Code installation complete"`,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "kodama-bin",
				MountPath: "/kodama/bin",
			},
		},
	}
}

// DefaultDiffViewerImage is the default container image for the difit diff viewer
const DefaultDiffViewerImage = "node:21-slim"

// DefaultDiffViewerPort is the default port for the difit web server
const DefaultDiffViewerPort int32 = 4966

// createDiffViewerSidecarContainer creates the difit diff viewer sidecar container
func createDiffViewerSidecarContainer(spec *DiffViewerSpec) *corev1.Container {
	if spec == nil || !spec.Enabled {
		return nil
	}

	image := spec.Image
	if image == "" {
		image = DefaultDiffViewerImage
	}

	port := spec.Port
	if port == 0 {
		port = DefaultDiffViewerPort
	}

	return &corev1.Container{
		Name:       "diff-viewer",
		Image:      image,
		WorkingDir: "/workspace",
		Command:    []string{"/bin/sh", "-c"},
		Args: []string{
			`echo "Installing difit diff viewer..."
npm install -g difit
echo "Starting difit on port $DIFIT_PORT..."
difit --port $DIFIT_PORT`,
		},
		Env: []corev1.EnvVar{
			{
				Name:  "DIFIT_PORT",
				Value: fmt.Sprintf("%d", port),
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "difit",
				ContainerPort: port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
		},
	}
}

// createWorkspaceInitializerInitContainer creates init container for git operations
func createWorkspaceInitializerInitContainer(spec *PodSpec) *corev1.Container {
	// Skip if no repository specified
	if spec.GitRepo == "" {
		return nil
	}

	// Build git clone options from spec
	opts := &gitcmd.CloneOptions{
		Depth:        spec.GitCloneDepth,
		SingleBranch: spec.GitSingleBranch,
		ExtraArgs:    spec.GitCloneArgs,
	}

	// Build git initialization script using gitcmd package
	initScript := gitcmd.BuildGitInitScript(spec.GitRepo, spec.GitBranch, opts)

	container := corev1.Container{
		Name:    "workspace-initializer",
		Image:   "ubuntu:24.04",
		Command: []string{"/bin/bash", "-c"},
		Args:    []string{initScript},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
		},
	}

	// Add git token environment variable if available
	if spec.GitSecretName != "" {
		container.Env = append(container.Env, corev1.EnvVar{
			Name: "GH_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: spec.GitSecretName,
					},
					Key:      "token",
					Optional: ptr.To(true),
				},
			},
		})
	}

	return &container
}

// CreatePod creates a new pod in the cluster
func (c *Client) CreatePod(ctx context.Context, spec *PodSpec) error {
	// Build init containers
	initContainers := []corev1.Container{
		createClaudeInstallerInitContainer(),
	}

	// Add workspace initializer if git repo specified
	if workspaceInit := createWorkspaceInitializerInitContainer(spec); workspaceInit != nil {
		initContainers = append(initContainers, *workspaceInit)
	}

	// Build main containers list
	containers := []corev1.Container{
		{
			Name:       "claude-code",
			Image:      spec.Image,
			Command:    spec.Command,
			WorkingDir: "/workspace",
			Resources:  c.buildResourceRequirements(spec.CPULimit, spec.MemoryLimit),
		},
	}

	// Add diff viewer sidecar if enabled
	if diffViewerContainer := createDiffViewerSidecarContainer(spec.DiffViewer); diffViewerContainer != nil {
		containers = append(containers, *diffViewerContainer)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels: map[string]string{
				"app":     "kodama",
				"session": spec.Name,
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: initContainers,
			Containers:     containers,
			RestartPolicy:  corev1.RestartPolicyNever,
		},
	}

	// Add PATH environment variable to include kodama-bin (contains Claude Code and other tools)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{
			Name:  "PATH",
			Value: "/kodama/bin:/root/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	// Add git secret as environment variable if specified
	if spec.GitSecretName != "" {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name: "GH_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: spec.GitSecretName,
					},
					Key: "token",
				},
			},
		})
	}

	// Add Claude authentication based on auth type
	if spec.ClaudeAuthType != "" {
		c.injectClaudeAuth(pod, spec)
	}

	// Build volumes and volume mounts
	volumes := []corev1.Volume{
		// Kodama bin volume - for Claude Code and kodama-specific tools
		{
			Name: "kodama-bin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		// Kodama bin mount - contains Claude Code and allows runtime tool installations
		{
			Name:      "kodama-bin",
			MountPath: "/kodama/bin",
		},
	}

	// Workspace volume - always included (PVC or emptyDir)
	if spec.WorkspacePVC != "" {
		// Use PVC if specified
		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: spec.WorkspacePVC,
				},
			},
		})
	} else {
		// Use emptyDir by default (needed for git clone in init container)
		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "workspace",
		MountPath: "/workspace",
	})

	if spec.ClaudeHomePVC != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "claude-home",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: spec.ClaudeHomePVC,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "claude-home",
			MountPath: "/home/claude",
		})
	}

	pod.Spec.Volumes = volumes
	pod.Spec.Containers[0].VolumeMounts = volumeMounts

	_, err := c.clientset.CoreV1().Pods(spec.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("pod %s already exists in namespace %s", spec.Name, spec.Namespace)
		}
		return fmt.Errorf("failed to create pod %s in namespace %s: %w", spec.Name, spec.Namespace, err)
	}

	return nil
}

// buildResourceRequirements creates resource requirements from CPU and memory limits
func (c *Client) buildResourceRequirements(cpu, memory string) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	if cpu != "" {
		cpuQuantity, err := resource.ParseQuantity(cpu)
		if err == nil {
			requirements.Limits[corev1.ResourceCPU] = cpuQuantity
			// Set requests to 50% of limits
			requestCPU := cpuQuantity.DeepCopy()
			requestCPU.Set(requestCPU.Value() / 2)
			requirements.Requests[corev1.ResourceCPU] = requestCPU
		}
	}

	if memory != "" {
		memQuantity, err := resource.ParseQuantity(memory)
		if err == nil {
			requirements.Limits[corev1.ResourceMemory] = memQuantity
			// Set requests to 50% of limits
			requestMem := memQuantity.DeepCopy()
			requestMem.Set(requestMem.Value() / 2)
			requirements.Requests[corev1.ResourceMemory] = requestMem
		}
	}

	return requirements
}

// injectClaudeAuth injects Claude authentication configuration into the pod
func (c *Client) injectClaudeAuth(pod *corev1.Pod, spec *PodSpec) {
	switch spec.ClaudeAuthType {
	case "token":
		c.injectTokenAuth(pod, spec)
	case "file":
		c.injectFileAuth(pod, spec)
	}
}

// injectTokenAuth injects token-based authentication
func (c *Client) injectTokenAuth(pod *corev1.Pod, spec *PodSpec) {
	if spec.ClaudeSecretName == "" {
		return
	}

	secretKey := spec.ClaudeSecretKey
	if secretKey == "" {
		secretKey = "token"
	}

	envVar := corev1.EnvVar{
		Name: "CLAUDE_CODE_AUTH_TOKEN",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: spec.ClaudeSecretName,
				},
				Key: secretKey,
			},
		},
	}

	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envVar)
}

// injectFileAuth injects file-based authentication
func (c *Client) injectFileAuth(pod *corev1.Pod, spec *PodSpec) {
	if spec.ClaudeSecretName == "" || spec.ClaudeAuthFile == "" {
		return
	}

	// Add volume for auth file
	volume := corev1.Volume{
		Name: "claude-auth-file",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: spec.ClaudeSecretName,
			},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

	// Mount auth file
	volumeMount := corev1.VolumeMount{
		Name:      "claude-auth-file",
		MountPath: "/.kodama/claude-auth.json",
		SubPath:   "auth.json",
		ReadOnly:  true,
	}
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, volumeMount)

	// Set environment variable for auth file location
	envVar := corev1.EnvVar{
		Name:  "CLAUDE_AUTH_FILE",
		Value: "/.kodama/claude-auth.json",
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envVar)
}

// GetPod retrieves pod information
func (c *Client) GetPod(ctx context.Context, name, namespace string) (*PodStatus, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("pod %s not found in namespace %s", name, namespace)
		}
		return nil, fmt.Errorf("failed to get pod %s in namespace %s: %w", name, namespace, err)
	}

	status := &PodStatus{
		Phase:      pod.Status.Phase,
		IP:         pod.Status.PodIP,
		Conditions: pod.Status.Conditions,
		Ready:      false,
	}

	if pod.Status.StartTime != nil {
		status.StartTime = pod.Status.StartTime.Format(time.RFC3339)
	}

	// Check if pod is ready
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			status.Ready = true
			break
		}
	}

	return status, nil
}

// WaitForPodReady polls the pod until it reaches Ready state
func (c *Client) WaitForPodReady(ctx context.Context, name, namespace string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use watch interface for efficient waiting
	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("failed to watch pod %s: %w", name, err)
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed unexpectedly for pod %s", name)
			}

			if event.Type == watch.Error {
				return fmt.Errorf("watch error for pod %s", name)
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			// Check if pod is ready
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
					return nil
				}
			}

			// Check for pod failure
			if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("pod %s failed: %s", name, pod.Status.Message)
			}

		case <-ctx.Done():
			// Timeout - get pod events for debugging
			events, err := c.getPodEvents(context.Background(), name, namespace)
			if err != nil {
				return fmt.Errorf("pod %s did not become ready within %v", name, timeout)
			}
			return fmt.Errorf("pod %s did not become ready within %v. Recent events:\n%s", name, timeout, events)
		}
	}
}

// getPodEvents retrieves recent events for a pod
func (c *Client) getPodEvents(ctx context.Context, name, namespace string) (string, error) {
	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", name),
	})
	if err != nil {
		return "", err
	}

	if len(events.Items) == 0 {
		return "No events found", nil
	}

	result := ""
	for _, event := range events.Items {
		result += fmt.Sprintf("  %s: %s - %s\n", event.Type, event.Reason, event.Message)
	}
	return result, nil
}

// DeletePod removes a pod from the cluster
func (c *Client) DeletePod(ctx context.Context, name, namespace string) error {
	gracePeriod := int64(30)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod not found is considered success
			return nil
		}
		return fmt.Errorf("failed to delete pod %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

// WaitForPodDeleted waits for a pod to be fully deleted from the cluster
func (c *Client) WaitForPodDeleted(ctx context.Context, name, namespace string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// First check if pod already doesn't exist
	_, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod already deleted
			return nil
		}
		// Some other error occurred
		return fmt.Errorf("failed to check pod status: %w", err)
	}

	// Use watch interface to wait for deletion
	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("failed to watch pod %s: %w", name, err)
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				// Watch channel closed - verify pod is deleted
				_, err := c.clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("watch channel closed but pod %s still exists", name)
			}

			if event.Type == watch.Deleted {
				// Pod has been deleted
				return nil
			}

			if event.Type == watch.Error {
				return fmt.Errorf("watch error for pod %s", name)
			}

		case <-ctx.Done():
			// Timeout - check current pod status
			pod, err := c.clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				// Pod was deleted just as we timed out
				return nil
			}
			if err != nil {
				return fmt.Errorf("pod %s deletion timeout after %v: %w", name, timeout, err)
			}
			return fmt.Errorf("pod %s was not deleted within %v, current phase: %s", name, timeout, pod.Status.Phase)
		}
	}
}

// GetPodIP returns the pod's IP address for verification
func (c *Client) GetPodIP(ctx context.Context, name, namespace string) (string, error) {
	status, err := c.GetPod(ctx, name, namespace)
	if err != nil {
		return "", err
	}

	if status.IP == "" {
		return "", fmt.Errorf("pod %s does not have an IP address yet", name)
	}

	return status.IP, nil
}
