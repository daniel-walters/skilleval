package runner

import (
	"encoding/json"
	"testing"

	"github.com/daniel-walters/skilleval/result"
)

func TestMapHelperOutput(t *testing.T) {
	raw := []byte(`{
  "id": "abc",
  "status": "cancelled",
  "finalMessage": "partial",
  "error": null,
  "durationMs": 5,
  "turns": 3,
  "toolsUsed": ["read"],
  "toolCalls": [{"name":"read","status":"completed"}],
  "usage": {"inputTokens":1,"outputTokens":1,"cacheReadTokens":0,"cacheWriteTokens":0,"totalTokens":2},
  "skills": {"activated":["helper"]}
}`)
	var out helperOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	obs := mapHelperOutput(out)
	if obs.Status != result.StatusCancelled || obs.ID != "abc" || obs.Turns != 3 {
		t.Fatalf("obs = %+v", obs)
	}
	if obs.Usage.TotalTokens != 2 || len(obs.Skills.Activated) != 1 || obs.Skills.Activated[0] != "helper" {
		t.Fatalf("usage/skills = %+v %+v", obs.Usage, obs.Skills)
	}
}

func TestMapHelperOutputPromotesRunningTools(t *testing.T) {
	raw := helperOutput{
		ID:     "abc",
		Status: "finished",
		ToolCalls: []helperToolCall{
			{Name: "glob", Status: "running"},
			{Name: "read", Status: "completed"},
		},
	}
	obs := mapHelperOutput(raw)
	if obs.ToolCalls[0].Status != result.ToolCallCompleted {
		t.Fatalf("running tool = %q, want completed", obs.ToolCalls[0].Status)
	}
	if obs.ToolCalls[1].Status != result.ToolCallCompleted {
		t.Fatalf("read = %q", obs.ToolCalls[1].Status)
	}
}

func TestMapHelperOutputRunningBecomesErrorOnFailedRun(t *testing.T) {
	raw := helperOutput{
		Status:    "error",
		ToolCalls: []helperToolCall{{Name: "shell", Status: "running"}},
	}
	obs := mapHelperOutput(raw)
	if obs.ToolCalls[0].Status != result.ToolCallError {
		t.Fatalf("status = %q, want error", obs.ToolCalls[0].Status)
	}
}

func TestMapHelperOutputPreservesArgs(t *testing.T) {
	raw := helperOutput{
		Status: "finished",
		ToolCalls: []helperToolCall{
			{
				Name:   "edit",
				Status: "completed",
				Args:   map[string]any{"path": "src/foo.go"},
			},
			{
				Name:   "shell",
				Status: "completed",
				Args:   map[string]any{"command": "go test"},
			},
		},
	}
	obs := mapHelperOutput(raw)
	if obs.ToolCalls[0].Args["path"] != "src/foo.go" {
		t.Fatalf("edit args = %+v", obs.ToolCalls[0].Args)
	}
	if obs.ToolCalls[1].Args["command"] != "go test" {
		t.Fatalf("shell args = %+v", obs.ToolCalls[1].Args)
	}
}

func TestMapHelperOutputUnknownStatusIsError(t *testing.T) {
	obs := mapHelperOutput(helperOutput{Status: "thinking"})
	if obs.Status != result.StatusError {
		t.Fatalf("status = %q, want error", obs.Status)
	}
	obs = mapHelperOutput(helperOutput{Status: ""})
	if obs.Status != result.StatusError {
		t.Fatalf("empty status = %q, want error", obs.Status)
	}
}

func TestExtractJSONObjectIgnoresLogLines(t *testing.T) {
	stdout := []byte(`07:31:54.279 INFO  LocalCursorRulesService load completed meta={durationMs: 29, ruleCount: 0}
07:31:54.288 INFO  AgentSkillsCursorRulesService load completed meta={durationMs: 36, ruleCount: 18, skillCount: 18}
{"id":"run_1","status":"finished","finalMessage":"ok","error":null,"durationMs":1,"turns":1,"toolsUsed":[],"toolCalls":[],"usage":{"inputTokens":0,"outputTokens":0,"cacheReadTokens":0,"cacheWriteTokens":0,"totalTokens":0},"skills":{"activated":[]}}
`)
	payload, err := extractJSONObject(stdout)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	var out helperOutput
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID != "run_1" || out.Status != "finished" {
		t.Fatalf("out = %+v", out)
	}
}
