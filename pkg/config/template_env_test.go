package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/illumination-k/kodama/pkg/env"
)

func TestLoadSessionTemplate_WithEnvConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "kodama-template-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test template with env config
	templatePath := filepath.Join(tmpDir, ".kodama.yaml")
	templateContent := `
env:
  dotenvFiles:
    - .env
    - .env.local
  excludeVars:
    - VERBOSE
    - DEBUG_MODE
`

	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Load template
	store := NewStoreWithPath(tmpDir)
	template, err := store.LoadSessionTemplate(templatePath)
	if err != nil {
		t.Fatalf("failed to load session template: %v", err)
	}

	// Verify env config was loaded
	if len(template.Env.DotenvFiles) != 2 {
		t.Errorf("expected 2 dotenv files, got %d", len(template.Env.DotenvFiles))
	}

	expectedFiles := []string{".env", ".env.local"}
	for i, expected := range expectedFiles {
		if template.Env.DotenvFiles[i] != expected {
			t.Errorf("dotenv file %d: got %q, want %q", i, template.Env.DotenvFiles[i], expected)
		}
	}

	if len(template.Env.ExcludeVars) != 2 {
		t.Errorf("expected 2 exclude vars, got %d", len(template.Env.ExcludeVars))
	}

	expectedExcludes := []string{"VERBOSE", "DEBUG_MODE"}
	for i, expected := range expectedExcludes {
		if template.Env.ExcludeVars[i] != expected {
			t.Errorf("exclude var %d: got %q, want %q", i, template.Env.ExcludeVars[i], expected)
		}
	}
}

func TestConfigResolver_WithTemplateEnvConfig(t *testing.T) {
	// Create global config with env defaults
	global := DefaultGlobalConfig()
	global.Defaults.Env = env.EnvConfig{
		ExcludeVars: []string{"GLOBAL_VAR"},
	}

	// Create template config with env overrides
	template := &SessionConfig{
		Env: env.EnvConfig{
			DotenvFiles: []string{".env", ".env.production"},
			ExcludeVars: []string{"TEMPLATE_VAR"},
		},
	}

	// Resolve configs
	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Verify template dotenv files override global (global has none)
	if len(resolved.EnvDotenvFiles) != 2 {
		t.Errorf("expected 2 dotenv files, got %d", len(resolved.EnvDotenvFiles))
	}

	expectedFiles := []string{".env", ".env.production"}
	for i, expected := range expectedFiles {
		if resolved.EnvDotenvFiles[i] != expected {
			t.Errorf("dotenv file %d: got %q, want %q", i, resolved.EnvDotenvFiles[i], expected)
		}
	}

	// Verify exclusions are appended (global + template)
	if len(resolved.EnvExcludeVars) != 2 {
		t.Errorf("expected 2 exclude vars (global + template), got %d", len(resolved.EnvExcludeVars))
	}

	// Should contain both GLOBAL_VAR and TEMPLATE_VAR
	hasGlobal := false
	hasTemplate := false
	for _, v := range resolved.EnvExcludeVars {
		if v == "GLOBAL_VAR" {
			hasGlobal = true
		}
		if v == "TEMPLATE_VAR" {
			hasTemplate = true
		}
	}

	if !hasGlobal {
		t.Error("expected GLOBAL_VAR in exclusions")
	}
	if !hasTemplate {
		t.Error("expected TEMPLATE_VAR in exclusions")
	}
}

func TestConfigResolver_EnvConfigPriority(t *testing.T) {
	// Global has some dotenv files
	global := DefaultGlobalConfig()
	global.Defaults.Env = env.EnvConfig{
		DotenvFiles: []string{".env.global"},
		ExcludeVars: []string{"GLOBAL_VAR"},
	}

	// Template overrides dotenv files, appends to exclusions
	template := &SessionConfig{
		Env: env.EnvConfig{
			DotenvFiles: []string{".env.template"},
			ExcludeVars: []string{"TEMPLATE_VAR"},
		},
	}

	resolver := NewConfigResolver(global, template)
	resolved := resolver.Resolve()

	// Template dotenv files should completely replace global
	if len(resolved.EnvDotenvFiles) != 1 {
		t.Errorf("expected 1 dotenv file (template only), got %d", len(resolved.EnvDotenvFiles))
	}
	if resolved.EnvDotenvFiles[0] != ".env.template" {
		t.Errorf("expected .env.template, got %s", resolved.EnvDotenvFiles[0])
	}

	// Exclusions should be appended
	if len(resolved.EnvExcludeVars) != 2 {
		t.Errorf("expected 2 exclude vars, got %d", len(resolved.EnvExcludeVars))
	}
}
