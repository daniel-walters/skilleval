package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnv_missingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := loadDotEnv(path); err != nil {
		t.Fatalf("missing file: %v", err)
	}
}

func TestLoadDotEnv_fillsUnset(t *testing.T) {
	const key = "SKILLEVAL_TEST_ENVFILE_FILL"
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=fromfile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv(key)
	})

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := os.Getenv(key); got != "fromfile" {
		t.Fatalf("Getenv(%q) = %q, want fromfile", key, got)
	}
}

func TestLoadDotEnv_doesNotOverride(t *testing.T) {
	const key = "SKILLEVAL_TEST_ENVFILE_KEEP"
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=fromfile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(key, "fromenv")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := os.Getenv(key); got != "fromenv" {
		t.Fatalf("Getenv(%q) = %q, want fromenv", key, got)
	}
}

func TestLoadDotEnv_parseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	// Unterminated double-quoted value is rejected by godotenv.
	if err := os.WriteFile(path, []byte("FOO=\"bar\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := loadDotEnv(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "envfile:") {
		t.Fatalf("error = %v, want envfile: prefix", err)
	}
}
