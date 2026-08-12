package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/runner"
	"github.com/daniel-walters/skilleval/skill"
)

func TestLookupAgent(t *testing.T) {
	a, err := runner.LookupAgent("cursor")
	if err != nil || a.RunnerID() != "cursor" {
		t.Fatalf("cursor: %v %v", a, err)
	}
	a, err = runner.LookupAgent("")
	if err != nil || a.RunnerID() != "cursor" {
		t.Fatalf("default: %v %v", a, err)
	}
	a, err = runner.LookupAgent("claude")
	if err != nil || a.RunnerID() != "claude" {
		t.Fatalf("claude: %v %v", a, err)
	}
	if _, err := runner.LookupAgent("nope"); err == nil {
		t.Fatal("expected error for unknown runner")
	}
}

func TestClaudePrepareWorkspace(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\ndescription: d\n---\n\n# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sk, err := skill.Load(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	agent := &runner.ClaudeAgent{}
	if err := agent.PrepareWorkspace(ws, sk); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".claude", "skills", "demo", "SKILL.md")); err != nil {
		t.Fatalf("skill not placed: %v", err)
	}
	if !agent.IgnoreOutcomePath(".claude/skills/demo/SKILL.md") {
		t.Fatal("expected IgnoreOutcomePath for .claude")
	}
	if agent.IgnoreOutcomePath("src/foo.go") {
		t.Fatal("should not ignore src")
	}
}

func TestClaudeSeedMCP(t *testing.T) {
	ws := t.TempDir()
	src := filepath.Join(t.TempDir(), "mcp.json")
	body := []byte(`{"mcpServers":{"echo-mcp":{"command":"node","args":["servers/echo.mjs"]}}}`)
	if err := os.WriteFile(src, body, 0o644); err != nil {
		t.Fatal(err)
	}
	agent := &runner.ClaudeAgent{}
	if err := agent.SeedMCP(ws, src); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(ws, ".mcp.json"))
	if err != nil {
		t.Fatalf("read seeded mcp: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("seeded mcp = %s, want %s", got, body)
	}
	if !agent.IgnoreOutcomePath(".mcp.json") {
		t.Fatal("expected IgnoreOutcomePath for .mcp.json")
	}
	if agent.IgnoreOutcomePath("mcp.json") {
		t.Fatal("should not ignore non-dot mcp.json")
	}
}

func TestRunSeedsClaudeMCP(t *testing.T) {
	dir := t.TempDir()
	setupEval(t, dir)
	mcpBody := []byte(`{"mcpServers":{"echo-mcp":{"command":"node","args":["servers/echo.mjs"]}}}`)
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), mcpBody, 0o644); err != nil {
		t.Fatalf("write mcp.json: %v", err)
	}
	evalPath := filepath.Join(dir, "eval.yaml")
	body := `schemaVersion: 1
name: demo-eval
prompt: Refactor src/foo.go
skill: skills/demo
input: fixtures/in
mcp: mcp.json
`
	if err := os.WriteFile(evalPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write eval: %v", err)
	}

	helperDir := t.TempDir()
	script := `#!/usr/bin/env node
process.stdout.write(JSON.stringify({
  id: "claude-mcp",
  status: "finished",
  finalMessage: "ok",
  error: null,
  durationMs: 1,
  turns: 1,
  toolsUsed: [],
  toolCalls: [],
  usage: { inputTokens: 0, outputTokens: 0, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 0 },
  skills: { activated: [] }
}) + "\n");
`
	if err := os.WriteFile(filepath.Join(helperDir, "run.mjs"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ev, err := eval.Load(evalPath)
	if err != nil {
		t.Fatal(err)
	}
	agent := &runner.ClaudeAgent{HelperDir: helperDir}
	out, err := runner.Run(context.Background(), ev, evalPath, runner.Options{
		Model: "haiku",
		Agent: agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, workspace := out.Result, out.Workspace
	defer func() { _ = os.RemoveAll(workspace) }()

	got, err := os.ReadFile(filepath.Join(workspace, ".mcp.json"))
	if err != nil {
		t.Fatalf("read seeded mcp: %v", err)
	}
	if string(got) != string(mcpBody) {
		t.Fatalf("seeded mcp = %s, want %s", got, mcpBody)
	}
	if _, ok := r.Outcomes.Files[".mcp.json"]; ok {
		t.Fatal(".mcp.json must not appear in file outcomes")
	}
}

func TestClaudeHelperOutputJSON(t *testing.T) {
	helperDir := t.TempDir()
	script := `#!/usr/bin/env node
const out = {
  id: "claude-sess",
  status: "finished",
  finalMessage: "PIGEON-42",
  error: null,
  durationMs: 11,
  turns: 3,
  toolsUsed: ["Skill"],
  toolCalls: [{ name: "Skill", status: "completed" }],
  usage: { inputTokens: 1, outputTokens: 2, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 3 },
  skills: { activated: ["canary"] },
  costUSD: 0.039
};
process.stdout.write(JSON.stringify(out) + "\n");
`
	if err := os.WriteFile(filepath.Join(helperDir, "run.mjs"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	setupEval(t, dir)
	ev, err := eval.Load(filepath.Join(dir, "eval.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	agent := &runner.ClaudeAgent{HelperDir: helperDir}
	out, err := runner.Run(context.Background(), ev, filepath.Join(dir, "eval.yaml"), runner.Options{
		Model: "haiku",
		Agent: agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, workspace := out.Result, out.Workspace
	defer func() { _ = os.RemoveAll(workspace) }()

	if r.Eval.Runner != "claude" {
		t.Fatalf("runner = %q", r.Eval.Runner)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".claude", "skills", "demo", "SKILL.md")); err != nil {
		t.Fatalf("claude skill placement: %v", err)
	}
	if r.ID != "claude-sess" || r.FinalMessage != "PIGEON-42" || r.Metrics.Turns != 3 {
		t.Fatalf("result = %+v", r)
	}
	if len(r.Skills.Activated) != 1 || r.Skills.Activated[0] != "canary" {
		t.Fatalf("skills = %+v", r.Skills)
	}
	if r.Metrics.CostUSD == nil || *r.Metrics.CostUSD != 0.039 {
		t.Fatalf("costUSD = %v", r.Metrics.CostUSD)
	}
}
