package kubernetes

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// CreatePod creates a new pod in the cluster
func (c *Client) CreatePod(ctx context.Context, spec *PodSpec) error {
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
			Containers: []corev1.Container{
				{
					Name:       "claude-code",
					Image:      spec.Image,
					Command:    spec.Command,
					WorkingDir: "/workspace",
					Resources:  c.buildResourceRequirements(spec.CPULimit, spec.MemoryLimit),
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// Add git secret as environment variable if specified
	if spec.GitSecretName != "" {
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name: "GH_TOKEN",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: spec.GitSecretName,
						},
						Key: "token",
					},
				},
			},
		}
	}

	// Add Claude authentication based on auth type
	if spec.ClaudeAuthType != "" {
		c.injectClaudeAuth(pod, spec)
	}

	// Add volume mounts if PVCs are specified (for future use)
	if spec.WorkspacePVC != "" || spec.ClaudeHomePVC != "" {
		volumes := []corev1.Volume{}
		volumeMounts := []corev1.VolumeMount{}

		if spec.WorkspacePVC != "" {
			volumes = append(volumes, corev1.Volume{
				Name: "workspace",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: spec.WorkspacePVC,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "workspace",
				MountPath: "/workspace",
			})
		}

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
	}

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
	case "federated":
		c.injectFederatedAuth(pod, spec)
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

// injectFederatedAuth injects federated/OAuth authentication
func (c *Client) injectFederatedAuth(pod *corev1.Pod, spec *PodSpec) {
	// Inject OAuth configuration as environment variables
	for key, value := range spec.ClaudeEnvVars {
		envVar := corev1.EnvVar{
			Name:  key,
			Value: value,
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envVar)
	}

	// Inject client secret from K8s secret if specified
	if spec.ClaudeSecretName != "" {
		envVar := corev1.EnvVar{
			Name: "CLAUDE_OAUTH_CLIENT_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: spec.ClaudeSecretName,
					},
					Key: "clientSecret",
				},
			},
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envVar)
	}
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
