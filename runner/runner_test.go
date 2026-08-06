package runner_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
	"github.com/daniel-walters/skilleval/runner"
)

type fakeAgent struct {
	obs     runner.AgentObservables
	err     error
	mutate  func(workspace string) error
	lastReq runner.AgentRequest
}

func (f *fakeAgent) Run(ctx context.Context, req runner.AgentRequest) (runner.AgentObservables, error) {
	f.lastReq = req
	if f.mutate != nil {
		if err := f.mutate(req.Workspace); err != nil {
			return runner.AgentObservables{}, err
		}
	}
	if f.err != nil {
		return runner.AgentObservables{}, f.err
	}
	return f.obs, nil
}

func TestRunSeedsInputPlacesSkillAndDiffs(t *testing.T) {
	dir := t.TempDir()
	setupEval(t, dir)

	agent := &fakeAgent{
		obs: runner.AgentObservables{
			ID:           "run_test",
			Status:       result.StatusFinished,
			FinalMessage: "done",
			DurationMs:   42,
			Turns:        2,
			ToolsUsed:    []string{"read", "edit"},
			ToolCalls: []result.ToolCall{
				{Name: "read", Status: result.ToolCallCompleted},
				{Name: "edit", Status: result.ToolCallCompleted},
			},
			Usage: result.Usage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
			Skills: result.Skills{
				Activated: []string{"demo"},
			},
		},
		mutate: func(workspace string) error {
			// modify seeded file
			p := filepath.Join(workspace, "src", "foo.go")
			return os.WriteFile(p, []byte("package src\n\nfunc Foo() {}\n"), 0o644)
		},
	}

	ev, err := eval.Load(filepath.Join(dir, "eval.yaml"))
	if err != nil {
		t.Fatalf("eval.Load: %v", err)
	}
	r, workspace, err := runner.Run(context.Background(), ev, filepath.Join(dir, "eval.yaml"), runner.Options{
		Model:   "composer-2.5",
		Attempt: 1,
		Agent:   agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	if agent.lastReq.Workspace != workspace {
		t.Fatalf("agent cwd = %q, want %q", agent.lastReq.Workspace, workspace)
	}
	if agent.lastReq.Prompt != "Refactor src/foo.go" {
		t.Fatalf("prompt = %q", agent.lastReq.Prompt)
	}
	if agent.lastReq.SkillName != "demo" {
		t.Fatalf("skill name = %q", agent.lastReq.SkillName)
	}

	// skill placed for discovery
	if _, err := os.Stat(filepath.Join(workspace, ".cursor", "skills", "demo", "SKILL.md")); err != nil {
		t.Fatalf("skill not placed: %v", err)
	}

	if r.ID != "run_test" || r.Status != result.StatusFinished {
		t.Fatalf("result id/status = %s %s", r.ID, r.Status)
	}
	if r.Eval.Skill != "demo" || r.Eval.Runner != "cursor" || r.Eval.Model != "composer-2.5" {
		t.Fatalf("eval = %+v", r.Eval)
	}
	if r.FinalMessage != "done" || r.Metrics.Turns != 2 || r.Metrics.DurationMs != 42 {
		t.Fatalf("metrics/message = %+v %q", r.Metrics, r.FinalMessage)
	}
	if r.Metrics.Usage.TotalTokens != 15 {
		t.Fatalf("usage = %+v", r.Metrics.Usage)
	}
	// composer-2.5: (10*0.5 + 5*2.5) / 1e6 = 0.0000175
	if r.Metrics.CostUSD == nil {
		t.Fatal("costUSD = nil, want estimate")
	}
	if want := 0.0000175; math.Abs(*r.Metrics.CostUSD-want) > 1e-12 {
		t.Fatalf("costUSD = %g, want %g", *r.Metrics.CostUSD, want)
	}
	if len(r.Metrics.ToolsUsed) != 2 || r.Metrics.ToolsUsed[0] != "read" {
		t.Fatalf("toolsUsed = %#v", r.Metrics.ToolsUsed)
	}
	if r.Skills.Activated[0] != "demo" {
		t.Fatalf("skills = %+v", r.Skills)
	}
	if r.Outcomes.Files["src/foo.go"].Status != result.FileModified {
		t.Fatalf("outcomes = %#v", r.Outcomes.Files)
	}
	// .cursor must not appear in outcomes
	for path := range r.Outcomes.Files {
		if len(path) >= 7 && path[:7] == ".cursor" {
			t.Fatalf("unexpected .cursor outcome %q", path)
		}
	}
}

func TestRunFileCreatedAndDeleted(t *testing.T) {
	dir := t.TempDir()
	setupEval(t, dir)

	agent := &fakeAgent{
		obs: runner.AgentObservables{
			ID:     "run_files",
			Status: result.StatusFinished,
		},
		mutate: func(workspace string) error {
			if err := os.WriteFile(filepath.Join(workspace, "src", "new.go"), []byte("package src\n"), 0o644); err != nil {
				return err
			}
			return os.Remove(filepath.Join(workspace, "src", "foo.go"))
		},
	}

	ev, err := eval.Load(filepath.Join(dir, "eval.yaml"))
	if err != nil {
		t.Fatalf("eval.Load: %v", err)
	}
	r, workspace, err := runner.Run(context.Background(), ev, filepath.Join(dir, "eval.yaml"), runner.Options{
		Model: "m",
		Agent: agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	if r.Outcomes.Files["src/new.go"].Status != result.FileCreated {
		t.Fatalf("new = %#v", r.Outcomes.Files["src/new.go"])
	}
	if r.Outcomes.Files["src/foo.go"].Status != result.FileDeleted {
		t.Fatalf("foo = %#v", r.Outcomes.Files["src/foo.go"])
	}
}

func TestRunUnknownModelCostNil(t *testing.T) {
	dir := t.TempDir()
	setupEval(t, dir)

	agent := &fakeAgent{
		obs: runner.AgentObservables{
			ID:     "run_unknown_model",
			Status: result.StatusFinished,
			Usage: result.Usage{
				InputTokens:  1000,
				OutputTokens: 100,
				TotalTokens:  1100,
			},
		},
	}

	ev, err := eval.Load(filepath.Join(dir, "eval.yaml"))
	if err != nil {
		t.Fatalf("eval.Load: %v", err)
	}
	r, workspace, err := runner.Run(context.Background(), ev, filepath.Join(dir, "eval.yaml"), runner.Options{
		Model: "not-a-real-model",
		Agent: agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	if r.Metrics.CostUSD != nil {
		t.Fatalf("costUSD = %v, want nil for unknown model", *r.Metrics.CostUSD)
	}
}

func TestRunErrorStatus(t *testing.T) {
	dir := t.TempDir()
	setupEval(t, dir)
	msg := "boom"
	agent := &fakeAgent{
		obs: runner.AgentObservables{
			ID:     "run_err",
			Status: result.StatusError,
			Error:  &msg,
		},
	}
	ev, err := eval.Load(filepath.Join(dir, "eval.yaml"))
	if err != nil {
		t.Fatalf("eval.Load: %v", err)
	}
	r, workspace, err := runner.Run(context.Background(), ev, filepath.Join(dir, "eval.yaml"), runner.Options{
		Model: "m",
		Agent: agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()
	if r.Status != result.StatusError || r.Error == nil || *r.Error != "boom" {
		t.Fatalf("status/error = %s %v", r.Status, r.Error)
	}
}

func TestMapHelperOutputJSON(t *testing.T) {
	// Contract smoke: CursorAgent can decode a fixture blob via a fake node script dir.
	helperDir := t.TempDir()
	script := `#!/usr/bin/env node
const out = {
  id: "from-helper",
  status: "finished",
  finalMessage: "hi",
  error: null,
  durationMs: 9,
  turns: 1,
  toolsUsed: ["read"],
  toolCalls: [{ name: "read", status: "completed" }],
  usage: { inputTokens: 1, outputTokens: 2, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 3 },
  skills: { activated: [] }
};
process.stdout.write(JSON.stringify(out) + "\n");
`
	if err := os.WriteFile(filepath.Join(helperDir, "run.mjs"), []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	// package.json unused when HelperDir set without npm; ensure file exists for realism
	if err := os.WriteFile(filepath.Join(helperDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	dir := t.TempDir()
	setupEval(t, dir)
	ev, err := eval.Load(filepath.Join(dir, "eval.yaml"))
	if err != nil {
		t.Fatalf("eval.Load: %v", err)
	}

	agent := &runner.CursorAgent{HelperDir: helperDir}
	r, workspace, err := runner.Run(context.Background(), ev, filepath.Join(dir, "eval.yaml"), runner.Options{
		Model: "m",
		Agent: agent,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()
	if r.ID != "from-helper" || r.FinalMessage != "hi" || r.Metrics.Usage.TotalTokens != 3 {
		t.Fatalf("result = %+v", r)
	}
}

func setupEval(t *testing.T, dir string) {
	t.Helper()
	skillDir := filepath.Join(dir, "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	skillMD := "---\nname: demo\ndescription: demo skill\n---\n\n# Demo\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	inputDir := filepath.Join(dir, "fixtures", "in")
	if err := os.MkdirAll(filepath.Join(inputDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "src", "foo.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	body := `schemaVersion: 1
name: demo-eval
prompt: Refactor src/foo.go
skill: skills/demo
input: fixtures/in
`
	if err := os.WriteFile(filepath.Join(dir, "eval.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write eval: %v", err)
	}
}
