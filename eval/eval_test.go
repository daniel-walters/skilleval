package eval_test

import (
	"os"
	"path/filepath"
	"strings"
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
	skillSrc := filepath.Join("testdata", "skills", "refactor-helper")
	skillDst := filepath.Join(dir, "skills", "refactor-helper")
	if err := copyDir(skillSrc, skillDst); err != nil {
		t.Fatalf("copy skill: %v", err)
	}

	again, err := eval.Load(tmpPath)
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	assertGoldenEval(t, again)
}

func TestLoadRejectsBadSchemaVersion(t *testing.T) {
	path := writeTempEvalWithSkill(t, "schemaVersion: 99\nname: x\nprompt: p\nskill: skill-dir\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for unsupported schemaVersion")
	}
}

func TestLoadRejectsMissingName(t *testing.T) {
	path := writeTempEvalWithSkill(t, "schemaVersion: 1\nprompt: p\nskill: skill-dir\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadRejectsMissingInputDir(t *testing.T) {
	path := writeTempEvalWithSkill(t, "schemaVersion: 1\nname: x\nprompt: p\nskill: skill-dir\ninput: missing-fixtures\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for missing input directory")
	}
}

func TestLoadRejectsMissingSkillDir(t *testing.T) {
	path := writeTempEval(t, "schemaVersion: 1\nname: x\nprompt: p\nskill: missing-skill\n")
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for missing skill directory")
	}
}

func TestLoadRejectsSkillWithoutSKILLMD(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skill-dir")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	path := filepath.Join(dir, "eval.yaml")
	body := "schemaVersion: 1\nname: x\nprompt: p\nskill: skill-dir\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write eval: %v", err)
	}
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for skill missing SKILL.md")
	}
}

func TestLoadAttemptsAndPassRate(t *testing.T) {
	t.Run("explicit attempts and passRate", func(t *testing.T) {
		path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
attempts: 10
passRate:
  min: 0.8
`)
		e, err := eval.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if e.Attempts != 10 {
			t.Fatalf("Attempts = %d, want 10", e.Attempts)
		}
		if e.PassRate == nil || e.PassRate.Min == nil || *e.PassRate.Min != 0.8 {
			t.Fatalf("PassRate = %#v", e.PassRate)
		}
	})

	t.Run("omitted attempts defaults to 1", func(t *testing.T) {
		path := writeTempEvalWithSkill(t, "schemaVersion: 1\nname: x\nprompt: p\nskill: skill-dir\n")
		e, err := eval.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if e.Attempts != 1 {
			t.Fatalf("Attempts = %d, want 1", e.Attempts)
		}
	})

	t.Run("non-positive attempts defaults to 1", func(t *testing.T) {
		path := writeTempEvalWithSkill(t, "schemaVersion: 1\nname: x\nprompt: p\nskill: skill-dir\nattempts: 0\n")
		e, err := eval.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if e.Attempts != 1 {
			t.Fatalf("Attempts = %d, want 1", e.Attempts)
		}
	})
}

func TestLoadRejectsInvalidPassRate(t *testing.T) {
	t.Run("missing min", func(t *testing.T) {
		path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
passRate: {}
`)
		if _, err := eval.Load(path); err == nil {
			t.Fatal("expected error for passRate without min")
		}
	})

	t.Run("min below 0", func(t *testing.T) {
		path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
passRate:
  min: -0.1
`)
		if _, err := eval.Load(path); err == nil {
			t.Fatal("expected error for passRate.min < 0")
		}
	})

	t.Run("min above 1", func(t *testing.T) {
		path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
passRate:
  min: 1.1
`)
		if _, err := eval.Load(path); err == nil {
			t.Fatal("expected error for passRate.min > 1")
		}
	})
}

func TestLoadRejectsInvalidFileStatus(t *testing.T) {
	path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
expects:
  files:
    a.go:
      status: touched
`)
	if _, err := eval.Load(path); err == nil {
		t.Fatal("expected error for invalid file status")
	}
}

func TestLoadRejectsEscapingFilePath(t *testing.T) {
	path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
expects:
  files:
    "../secret.txt":
      contains: "x"
`)
	_, err := eval.Load(path)
	if err == nil {
		t.Fatal("expected error for escaping file path")
	}
	if !strings.Contains(err.Error(), "relative to workspace") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRejectsInvalidFinalMessageRegex(t *testing.T) {
	path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
expects:
  finalMessage:
    contains: /(/
`)
	_, err := eval.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid finalMessage.contains regex")
	}
	if !strings.Contains(err.Error(), "finalMessage.contains") || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("error = %v, want finalMessage.contains invalid regex", err)
	}
}

func TestLoadRejectsInvalidFileContainsRegex(t *testing.T) {
	path := writeTempEvalWithSkill(t, `schemaVersion: 1
name: x
prompt: p
skill: skill-dir
expects:
  files:
    a.go:
      contains: /(/
`)
	_, err := eval.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid files[].contains regex")
	}
	if !strings.Contains(err.Error(), `files["a.go"].contains`) || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("error = %v, want files[a.go].contains invalid regex", err)
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
	if !strings.Contains(e.Prompt, "Delete src/gone.go") {
		t.Fatalf("Prompt = %q", e.Prompt)
	}
	if e.Skill != "skills/refactor-helper" {
		t.Fatalf("Skill = %q", e.Skill)
	}
	if e.Input != "fixtures/refactor-helper" {
		t.Fatalf("Input = %q", e.Input)
	}
	if e.Attempts != 1 {
		t.Fatalf("Attempts = %d, want 1 (default)", e.Attempts)
	}
	if e.PassRate != nil {
		t.Fatalf("PassRate = %#v, want nil", e.PassRate)
	}

	if e.Expects.Turns == nil || e.Expects.Turns.Max == nil || *e.Expects.Turns.Max != 15 {
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
	if foo.Status != result.FileModified || foo.Contains.String() != "/func Foo/" || !foo.Contains.IsRegex() {
		t.Fatalf("foo = %#v", foo)
	}
	newFile := e.Expects.Files["src/new.go"]
	if newFile.Status != result.FileCreated || newFile.Contains.String() != "package demo" || newFile.Contains.IsRegex() {
		t.Fatalf("new = %#v", newFile)
	}
	if e.Expects.Files["src/gone.go"].Status != result.FileDeleted {
		t.Fatalf("gone = %#v", e.Expects.Files["src/gone.go"])
	}

	if e.Expects.FinalMessage == nil ||
		e.Expects.FinalMessage.Contains.String() != "/Refactor/" ||
		!e.Expects.FinalMessage.Contains.IsRegex() {
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

func writeTempEvalWithSkill(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skill-dir")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	skillMD := "---\nname: helper\ndescription: test\n---\n\n# Helper\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
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
