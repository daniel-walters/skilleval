package eval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
	"gopkg.in/yaml.v3"
)

func TestSchemaVersion(t *testing.T) {
	if eval.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", eval.SchemaVersion)
	}
}

func TestLoadGolden(t *testing.T) {
	goldenPath := filepath.Join("testdata", "eval.yaml")
	got, err := eval.Load(goldenPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertGoldenEval(t, got)
}

func TestGoldenRoundTrip(t *testing.T) {
	goldenPath := filepath.Join("testdata", "eval.yaml")
	got, err := eval.Load(goldenPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	encoded, err := yaml.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Write to a temp file so Load can re-validate input relative to the YAML.
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "eval.yaml")
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	fixtureSrc := filepath.Join("testdata", "fixtures", "refactor-helper")
	fixtureDst := filepath.Join(dir, "fixtures", "refactor-helper")
	if err := copyDir(fixtureSrc, fixtureDst); err != nil {
		t.Fatalf("copy fixtures: %v", err)
	}

	again, err := eval.Load(tmpPath)
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	assertGoldenEval(t, again)
}

func TestLoadRejectsBadSchemaVersion(t *testing.T) {
	path := writeTempEval(t, "schemaVersion: 99\nname: x\nprompt: p\nskill: s\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for unsupported schemaVersion")
	}
}

func TestLoadRejectsMissingName(t *testing.T) {
	path := writeTempEval(t, "schemaVersion: 1\nprompt: p\nskill: s\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadRejectsMissingInputDir(t *testing.T) {
	path := writeTempEval(t, "schemaVersion: 1\nname: x\nprompt: p\nskill: s\ninput: missing-fixtures\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for missing input directory")
	}
}

func TestLoadRejectsInvalidFileStatus(t *testing.T) {
	path := writeTempEval(t, `schemaVersion: 1
name: x
prompt: p
skill: s
expects:
  files:
    a.go:
      status: touched
`)
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for invalid file status")
	}
}

func assertGoldenEval(t *testing.T, e *eval.Eval) {
	t.Helper()

	if e.SchemaVersion != eval.SchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", e.SchemaVersion, eval.SchemaVersion)
	}
	if e.Name != "refactor-helper" {
		t.Fatalf("Name = %q", e.Name)
	}
	if e.Prompt != "Refactor src/foo.go for clarity" {
		t.Fatalf("Prompt = %q", e.Prompt)
	}
	if e.Skill != "refactor-helper" {
		t.Fatalf("Skill = %q", e.Skill)
	}
	if e.Input != "fixtures/refactor-helper" {
		t.Fatalf("Input = %q", e.Input)
	}

	if e.Expects.Turns == nil || e.Expects.Turns.Max == nil || *e.Expects.Turns.Max != 10 {
		t.Fatalf("Turns = %#v", e.Expects.Turns)
	}
	if e.Expects.CostUSD == nil || e.Expects.CostUSD.Max == nil || *e.Expects.CostUSD.Max != 1.0 {
		t.Fatalf("CostUSD = %#v", e.Expects.CostUSD)
	}
	if e.Expects.ToolsUsed == nil ||
		len(e.Expects.ToolsUsed.Includes) != 2 ||
		e.Expects.ToolsUsed.Includes[0] != "read" ||
		e.Expects.ToolsUsed.Includes[1] != "edit" ||
		len(e.Expects.ToolsUsed.Excludes) != 1 ||
		e.Expects.ToolsUsed.Excludes[0] != "web" {
		t.Fatalf("ToolsUsed = %#v", e.Expects.ToolsUsed)
	}
	if e.Expects.Skills == nil || e.Expects.Skills.Activated == nil ||
		len(e.Expects.Skills.Activated.Includes) != 1 ||
		e.Expects.Skills.Activated.Includes[0] != "refactor-helper" {
		t.Fatalf("Skills = %#v", e.Expects.Skills)
	}

	if len(e.Expects.Files) != 3 {
		t.Fatalf("files len = %d", len(e.Expects.Files))
	}
	foo := e.Expects.Files["src/foo.go"]
	if foo.Status != result.FileModified || foo.Contains != "func Foo" {
		t.Fatalf("foo = %#v", foo)
	}
	if e.Expects.Files["src/new.go"].Status != result.FileCreated {
		t.Fatalf("new = %#v", e.Expects.Files["src/new.go"])
	}
	if e.Expects.Files["src/gone.go"].Status != result.FileDeleted {
		t.Fatalf("gone = %#v", e.Expects.Files["src/gone.go"])
	}

	if e.Expects.FinalMessage == nil || e.Expects.FinalMessage.Contains != "Refactored" {
		t.Fatalf("FinalMessage = %#v", e.Expects.FinalMessage)
	}
}

func writeTempEval(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "eval.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write temp eval: %v", err)
	}
	return path
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
