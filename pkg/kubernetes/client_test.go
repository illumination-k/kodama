package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewClient_WithKubeconfig tests client creation with kubeconfig
// Note: This test requires a valid kubeconfig file
func TestNewClient_WithKubeconfig(t *testing.T) {
	// Skip if not in a Kubernetes environment
	t.Skip("Requires valid kubeconfig")

	client, err := NewClient("")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

// TestClient_Ping tests cluster connectivity
// Note: This test requires a running Kubernetes cluster
func TestClient_Ping(t *testing.T) {
	t.Skip("Requires running Kubernetes cluster")

	client, err := NewClient("")
	if err != nil {
		t.Skipf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.Ping(ctx)
	assert.NoError(t, err)
}
