package auth

import (
	"fmt"
	"strings"
	"sync"
)

// Sanitizer handles token sanitization in error messages and logs
type Sanitizer struct {
	tokens map[string]struct{} // Set of tokens to redact
	mu     sync.RWMutex
}

// NewSanitizer creates a new sanitizer instance
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		tokens: make(map[string]struct{}),
	}
}

// AddToken registers a token for sanitization
func (s *Sanitizer) AddToken(token string) {
	if token == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token] = struct{}{}
}

// Sanitize replaces all registered tokens with [REDACTED]
func (s *Sanitizer) Sanitize(text string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := text
	for token := range s.tokens {
		result = strings.ReplaceAll(result, token, "[REDACTED]")
	}

	return result
}

// SanitizeError wraps an error with a sanitized message
func (s *Sanitizer) SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s", s.Sanitize(err.Error()))
}

// Clear removes all registered tokens
func (s *Sanitizer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens = make(map[string]struct{})
}
