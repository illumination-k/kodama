package auth

import (
	"context"
	"os"
	"testing"
)

func TestTokenProvider_GetCredentials(t *testing.T) {
	tests := []struct {
		name      string
		config    TokenConfig
		envVars   map[string]string
		wantToken string
		wantErr   bool
	}{
		{
			name: "direct token",
			config: TokenConfig{
				Token: "test-token-123",
			},
			wantToken: "test-token-123",
			wantErr:   false,
		},
		{
			name: "from environment variable",
			config: TokenConfig{
				EnvVar: "TEST_TOKEN",
			},
			envVars: map[string]string{
				"TEST_TOKEN": "env-token-456",
			},
			wantToken: "env-token-456",
			wantErr:   false,
		},
		{
			name:   "from default environment variable",
			config: TokenConfig{},
			envVars: map[string]string{
				"CLAUDE_CODE_AUTH_TOKEN": "default-token-789",
			},
			wantToken: "default-token-789",
			wantErr:   false,
		},
		{
			name:    "no token available",
			config:  TokenConfig{},
			wantErr: true,
		},
		{
			name: "direct token takes precedence",
			config: TokenConfig{
				Token:  "direct-token",
				EnvVar: "TEST_TOKEN",
			},
			envVars: map[string]string{
				"TEST_TOKEN": "env-token",
			},
			wantToken: "direct-token",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
				defer func(k string) { _ = os.Unsetenv(k) }(key)
			}

			provider := NewTokenProvider(tt.config)
			creds, err := provider.GetCredentials(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && creds.Token != tt.wantToken {
				t.Errorf("GetCredentials() token = %v, want %v", creds.Token, tt.wantToken)
			}
		})
	}
}

func TestTokenProvider_Type(t *testing.T) {
	provider := NewTokenProvider(TokenConfig{})
	if provider.Type() != AuthTypeToken {
		t.Errorf("Type() = %v, want %v", provider.Type(), AuthTypeToken)
	}
}

func TestTokenProvider_NeedsRefresh(t *testing.T) {
	provider := NewTokenProvider(TokenConfig{})
	if provider.NeedsRefresh() {
		t.Error("NeedsRefresh() should return false for token provider")
	}
}

func TestTokenProvider_Refresh(t *testing.T) {
	provider := NewTokenProvider(TokenConfig{})
	err := provider.Refresh(context.Background())
	if err != nil {
		t.Errorf("Refresh() should not error for token provider, got %v", err)
	}
}
