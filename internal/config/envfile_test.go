package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistEnvVarCreatesAndUpdatesEntry(t *testing.T) {
	baseDir := t.TempDir()
	path := EnvFilePath(baseDir)

	if err := PersistEnvVar(baseDir, "KIMI_API_KEY", "sk-first"); err != nil {
		t.Fatalf("PersistEnvVar() first error = %v", err)
	}
	if err := PersistEnvVar(baseDir, "KIMI_API_KEY", "sk-second"); err != nil {
		t.Fatalf("PersistEnvVar() second error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	text := string(data)
	if strings.Count(text, "KIMI_API_KEY=") != 1 {
		t.Fatalf("expected exactly one key line, got %q", text)
	}
	if !strings.Contains(text, "KIMI_API_KEY=sk-second\n") {
		t.Fatalf("expected updated value in env file, got %q", text)
	}
}

func TestPersistEnvVarPreservesOtherLines(t *testing.T) {
	baseDir := t.TempDir()
	path := EnvFilePath(baseDir)
	original := "# comment\nOTHER_KEY=1\n\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := PersistEnvVar(baseDir, "NEW_KEY", "value with space"); err != nil {
		t.Fatalf("PersistEnvVar() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "# comment\n") || !strings.Contains(text, "OTHER_KEY=1\n") {
		t.Fatalf("expected old lines to be preserved, got %q", text)
	}
	if !strings.Contains(text, "NEW_KEY=\"value with space\"\n") {
		t.Fatalf("expected quoted inserted line, got %q", text)
	}
}

func TestLoadPersistedEnvLoadsMissingKeysOnly(t *testing.T) {
	baseDir := t.TempDir()
	path := EnvFilePath(baseDir)
	content := "EXISTING_KEY=from-file\nNEW_KEY=\"hello world\"\n# ignored\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restoreExisting := captureEnv(t, "EXISTING_KEY")
	defer restoreExisting()
	restoreNew := captureEnv(t, "NEW_KEY")
	defer restoreNew()

	if err := os.Setenv("EXISTING_KEY", "from-process"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	if err := os.Unsetenv("NEW_KEY"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}

	if err := LoadPersistedEnv(baseDir); err != nil {
		t.Fatalf("LoadPersistedEnv() error = %v", err)
	}

	if got := os.Getenv("EXISTING_KEY"); got != "from-process" {
		t.Fatalf("expected EXISTING_KEY to keep process value, got %q", got)
	}
	if got := os.Getenv("NEW_KEY"); got != "hello world" {
		t.Fatalf("expected NEW_KEY loaded from env file, got %q", got)
	}
}

func TestPersistEnvVarRejectsInvalidInput(t *testing.T) {
	baseDir := t.TempDir()
	if err := PersistEnvVar(baseDir, "", "value"); err == nil {
		t.Fatal("expected empty key error")
	}
	if err := PersistEnvVar(baseDir, "BAD KEY", "value"); err == nil {
		t.Fatal("expected invalid key error")
	}
	if err := PersistEnvVar(baseDir, "KEY", "line1\nline2"); err == nil {
		t.Fatal("expected newline value error")
	}
}

func captureEnv(t *testing.T, key string) func() {
	t.Helper()
	value, exists := os.LookupEnv(key)
	return func() {
		if exists {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	}
}
