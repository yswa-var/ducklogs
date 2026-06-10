package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnv(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd returned error: %v", err)
	}
	defer os.Chdir(cwd)

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir returned error: %v", err)
	}

	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte(`OPENROUTER_API_KEY="Bearer test-key"
DUCKLOG_DATABASE=./from-env.duckdb
`), 0644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("DUCKLOG_DATABASE", "")

	cfg := Load()

	if cfg.OpenRouterAPIKey != "test-key" {
		t.Fatalf("OpenRouterAPIKey = %q, want test-key", cfg.OpenRouterAPIKey)
	}
	if cfg.Database != "./from-env.duckdb" {
		t.Fatalf("Database = %q, want ./from-env.duckdb", cfg.Database)
	}
}

func TestLoadDotEnvOverridesProcessEnv(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd returned error: %v", err)
	}
	defer os.Chdir(cwd)

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir returned error: %v", err)
	}

	if err := os.WriteFile(".env", []byte("OPENROUTER_API_KEY=from-dot-env\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	t.Setenv("OPENROUTER_API_KEY", "from-process")

	cfg := Load()

	if cfg.OpenRouterAPIKey != "from-dot-env" {
		t.Fatalf("OpenRouterAPIKey = %q, want from-dot-env", cfg.OpenRouterAPIKey)
	}
}

func TestNormalizeAPIKey(t *testing.T) {
	t.Parallel()

	got := normalizeAPIKey(" Bearer sk-or-v1-example ")
	if got != "sk-or-v1-example" {
		t.Fatalf("normalizeAPIKey returned %q", got)
	}
}
