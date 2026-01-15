package auth

import (
	"errors"
	"testing"
)

func TestSanitizer_Sanitize(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		input  string
		want   string
	}{
		{
			name:   "single token",
			tokens: []string{"secret123"},
			input:  "Error: authentication failed with token secret123",
			want:   "Error: authentication failed with token [REDACTED]",
		},
		{
			name:   "multiple tokens",
			tokens: []string{"token1", "token2"},
			input:  "token1 was used but token2 failed",
			want:   "[REDACTED] was used but [REDACTED] failed",
		},
		{
			name:   "no tokens registered",
			tokens: []string{},
			input:  "Error: something went wrong",
			want:   "Error: something went wrong",
		},
		{
			name:   "token not in text",
			tokens: []string{"mytoken123"},
			input:  "No secrets here",
			want:   "No secrets here",
		},
		{
			name:   "empty token ignored",
			tokens: []string{""},
			input:  "Some text",
			want:   "Some text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSanitizer()
			for _, token := range tt.tokens {
				s.AddToken(token)
			}

			got := s.Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizer_SanitizeError(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		err       error
		wantError string
	}{
		{
			name:      "sanitize error message",
			token:     "secret123",
			err:       errors.New("failed with token secret123"),
			wantError: "failed with token [REDACTED]",
		},
		{
			name:  "nil error",
			token: "secret",
			err:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSanitizer()
			s.AddToken(tt.token)

			got := s.SanitizeError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Errorf("SanitizeError() = %v, want nil", got)
				}
				return
			}

			if got.Error() != tt.wantError {
				t.Errorf("SanitizeError() = %q, want %q", got.Error(), tt.wantError)
			}
		})
	}
}

func TestSanitizer_Clear(t *testing.T) {
	s := NewSanitizer()
	s.AddToken("token1")
	s.AddToken("token2")

	// Tokens should be redacted before clear
	if got := s.Sanitize("token1 and token2"); got != "[REDACTED] and [REDACTED]" {
		t.Errorf("Before clear: Sanitize() = %q, want tokens redacted", got)
	}

	s.Clear()

	// Tokens should not be redacted after clear
	if got := s.Sanitize("token1 and token2"); got != "token1 and token2" {
		t.Errorf("After clear: Sanitize() = %q, want original text", got)
	}
}
