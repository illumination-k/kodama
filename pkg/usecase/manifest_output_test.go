package usecase

import (
	"bytes"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWriteManifestsYAML(t *testing.T) {
	tests := []struct {
		name      string
		manifests *ManifestCollection
		wantErr   bool
		contains  []string // Strings that should be in output
	}{
		{
			name: "pod only",
			manifests: &ManifestCollection{
				Pod: &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				},
			},
			wantErr:  false,
			contains: []string{"kind: Pod", "name: test-pod", "namespace: default"},
		},
		{
			name: "pod with env secret",
			manifests: &ManifestCollection{
				EnvSecret: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"KEY": []byte("value"),
					},
				},
				Pod: &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				},
			},
			wantErr:  false,
			contains: []string{"kind: Secret", "kind: Pod", "---", "name: test-secret", "name: test-pod"},
		},
		{
			name:      "nil manifests",
			manifests: nil,
			wantErr:   true,
		},
		{
			name: "missing pod",
			manifests: &ManifestCollection{
				EnvSecret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-secret",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteManifestsYAML(tt.manifests, &buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("WriteManifestsYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("WriteManifestsYAML() output missing %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestWriteManifestsJSON(t *testing.T) {
	tests := []struct {
		name      string
		manifests *ManifestCollection
		wantErr   bool
		contains  []string
	}{
		{
			name: "pod only",
			manifests: &ManifestCollection{
				Pod: &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				},
			},
			wantErr:  false,
			contains: []string{`"kind": "List"`, `"kind": "Pod"`, `"name": "test-pod"`},
		},
		{
			name:      "nil manifests",
			manifests: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteManifestsJSON(tt.manifests, &buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("WriteManifestsJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("WriteManifestsJSON() output missing %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name      string
		manifests *ManifestCollection
		wantNil   bool
	}{
		{
			name: "redact env secret",
			manifests: &ManifestCollection{
				EnvSecret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"API_KEY": []byte("secret-value"),
						"TOKEN":   []byte("another-secret"),
					},
				},
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				},
			},
			wantNil: false,
		},
		{
			name: "redact file secret",
			manifests: &ManifestCollection{
				FileSecret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "file-secret",
					},
					Data: map[string][]byte{
						"key1": []byte("file-content"),
					},
				},
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				},
			},
			wantNil: false,
		},
		{
			name:      "nil manifests",
			manifests: nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSecrets(tt.manifests)

			if tt.wantNil {
				if result != nil {
					t.Errorf("RedactSecrets() expected nil, got non-nil")
				}
				return
			}

			if result == nil {
				t.Errorf("RedactSecrets() returned nil, want non-nil")
				return
			}

			// Verify secrets are redacted
			if result.EnvSecret != nil {
				for key, value := range result.EnvSecret.Data {
					if string(value) != "<REDACTED>" {
						t.Errorf("EnvSecret[%s] = %q, want <REDACTED>", key, string(value))
					}
				}

				// Verify metadata is preserved
				if result.EnvSecret.Name != tt.manifests.EnvSecret.Name {
					t.Errorf("EnvSecret name = %q, want %q", result.EnvSecret.Name, tt.manifests.EnvSecret.Name)
				}
			}

			if result.FileSecret != nil {
				for key, value := range result.FileSecret.Data {
					if string(value) != "<REDACTED>" {
						t.Errorf("FileSecret[%s] = %q, want <REDACTED>", key, string(value))
					}
				}
			}

			// Verify pod is preserved (not modified)
			if result.Pod == nil {
				t.Errorf("RedactSecrets() pod is nil, want non-nil")
			}
		})
	}
}
