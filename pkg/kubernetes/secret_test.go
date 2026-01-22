package kubernetes

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateSecret(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		namespace  string
		data       map[string]string
		wantErr    bool
	}{
		{
			name:       "create secret with valid data",
			secretName: "kodama-env-test-session",
			namespace:  "default",
			data: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			wantErr: false,
		},
		{
			name:       "create secret with empty data",
			secretName: "kodama-env-empty",
			namespace:  "default",
			data:       map[string]string{},
			wantErr:    false,
		},
		{
			name:       "create secret with special characters in value",
			secretName: "kodama-env-special",
			namespace:  "default",
			data: map[string]string{
				"DATABASE_URL": "postgresql://user:p@ssw0rd!@localhost:5432/db",
				"API_KEY":      "sk-1234567890abcdefghijklmnopqrstuvwxyz",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			fakeClientset := fake.NewSimpleClientset()
			client := &Client{clientset: fakeClientset}

			// Create secret
			err := client.CreateSecret(context.Background(), tt.secretName, tt.namespace, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify secret was created
			secret, getErr := fakeClientset.CoreV1().Secrets(tt.namespace).Get(
				context.Background(),
				tt.secretName,
				metav1.GetOptions{},
			)
			if getErr != nil {
				t.Fatalf("failed to get created secret: %v", getErr)
			}

			// Verify secret data
			if len(secret.Data) != len(tt.data) {
				t.Errorf("secret data length = %d, want %d", len(secret.Data), len(tt.data))
			}

			for key, expectedValue := range tt.data {
				actualValue, ok := secret.Data[key]
				if !ok {
					t.Errorf("secret missing key %s", key)
					continue
				}
				if string(actualValue) != expectedValue {
					t.Errorf("secret[%s] = %s, want %s", key, string(actualValue), expectedValue)
				}
			}

			// Verify labels
			if secret.Labels["app"] != "kodama" {
				t.Errorf("secret label app = %s, want kodama", secret.Labels["app"])
			}
			if secret.Labels["managed-by"] != "kodama" {
				t.Errorf("secret label managed-by = %s, want kodama", secret.Labels["managed-by"])
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	tests := []struct {
		name         string
		secretName   string
		namespace    string
		existingObjs []runtime.Object
		wantErr      bool
	}{
		{
			name:       "delete existing secret",
			secretName: "kodama-env-test",
			namespace:  "default",
			existingObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kodama-env-test",
						Namespace: "default",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "delete non-existent secret (should not error)",
			secretName:   "kodama-env-nonexistent",
			namespace:    "default",
			existingObjs: []runtime.Object{},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset with existing objects
			fakeClientset := fake.NewSimpleClientset(tt.existingObjs...)
			client := &Client{clientset: fakeClientset}

			// Delete secret
			err := client.DeleteSecret(context.Background(), tt.secretName, tt.namespace)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify secret was deleted (if it existed)
			if len(tt.existingObjs) > 0 {
				_, getErr := fakeClientset.CoreV1().Secrets(tt.namespace).Get(
					context.Background(),
					tt.secretName,
					metav1.GetOptions{},
				)
				if getErr == nil {
					t.Errorf("secret still exists after deletion")
				}
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	tests := []struct {
		name         string
		secretName   string
		namespace    string
		existingObjs []runtime.Object
		want         bool
		wantErr      bool
	}{
		{
			name:       "secret exists",
			secretName: "kodama-env-test",
			namespace:  "default",
			existingObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kodama-env-test",
						Namespace: "default",
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:         "secret does not exist",
			secretName:   "kodama-env-nonexistent",
			namespace:    "default",
			existingObjs: []runtime.Object{},
			want:         false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset with existing objects
			fakeClientset := fake.NewSimpleClientset(tt.existingObjs...)
			client := &Client{clientset: fakeClientset}

			// Check if secret exists
			exists, err := client.SecretExists(context.Background(), tt.secretName, tt.namespace)

			if (err != nil) != tt.wantErr {
				t.Errorf("SecretExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.want {
				t.Errorf("SecretExists() = %v, want %v", exists, tt.want)
			}
		})
	}
}
