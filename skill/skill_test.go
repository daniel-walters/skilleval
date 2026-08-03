package skill_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-walters/skilleval/skill"
)

func TestLoadDir(t *testing.T) {
	dir := writeSkill(t, "---\nname: my-skill\ndescription: does stuff\n---\n\n# Hello\n\nBody text.\n")
	got, err := skill.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != "my-skill" {
		t.Fatalf("Name = %q", got.Name)
	}
	if got.Description != "does stuff" {
		t.Fatalf("Description = %q", got.Description)
	}
	if got.Dir != dir {
		// Load returns abs path
		abs, _ := filepath.Abs(dir)
		if got.Dir != abs {
			t.Fatalf("Dir = %q, want %q", got.Dir, abs)
		}
	}
	if got.Body == "" || got.Body[:7] != "# Hello" {
		t.Fatalf("Body = %q", got.Body)
	}
}

func TestLoadSKILLMDPath(t *testing.T) {
	dir := writeSkill(t, "---\nname: from-file\n---\n\n# X\n")
	got, err := skill.Load(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != "from-file" {
		t.Fatalf("Name = %q", got.Name)
	}
}

func TestLoadRejectsMissingName(t *testing.T) {
	dir := writeSkill(t, "---\ndescription: no name\n---\n\n# X\n")
	if _, err := skill.Load(dir); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadRejectsNonSkillFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := skill.Load(path); err == nil {
		t.Fatal("expected error for non-SKILL.md file")
	}
}

func TestLoadRejectsMissingDir(t *testing.T) {
	if _, err := skill.Load(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func writeSkill(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return abs
}
