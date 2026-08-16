package result_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daniel-walters/skilleval/result"
)

func TestSchemaVersion(t *testing.T) {
	if result.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", result.SchemaVersion)
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	goldenPath := filepath.Join("testdata", "result.json")
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var got result.Result
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	assertGoldenResult(t, got)

	encoded, err := json.Marshal(&got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var again result.Result
	if err := json.Unmarshal(encoded, &again); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	assertGoldenResult(t, again)

	// Re-encode both with the same settings and compare canonical form.
	wantCanonical, err := canonicalJSON(raw)
	if err != nil {
		t.Fatalf("canonical golden: %v", err)
	}
	gotCanonical, err := canonicalJSON(encoded)
	if err != nil {
		t.Fatalf("canonical encoded: %v", err)
	}
	if !bytes.Equal(wantCanonical, gotCanonical) {
		t.Fatalf("round-trip JSON mismatch\nwant: %s\ngot:  %s", wantCanonical, gotCanonical)
	}
}

func assertGoldenResult(t *testing.T, r result.Result) {
	t.Helper()

	if r.SchemaVersion != result.SchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", r.SchemaVersion, result.SchemaVersion)
	}
	if r.ID != "run_01hxyzexample000000000000" {
		t.Fatalf("ID = %q", r.ID)
	}
	if !r.StartedAt.Equal(time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("StartedAt = %v", r.StartedAt)
	}
	if !r.FinishedAt.Equal(time.Date(2026, 8, 2, 10, 0, 12, 0, time.UTC)) {
		t.Fatalf("FinishedAt = %v", r.FinishedAt)
	}

	if r.Eval.Name != "refactor-helper" || r.Eval.Runner != "cursor" {
		t.Fatalf("Eval = %+v", r.Eval)
	}
	if r.Eval.Attempt != 1 || r.Eval.TotalAttempts != 10 {
		t.Fatalf("attempt metadata = %d / %d", r.Eval.Attempt, r.Eval.TotalAttempts)
	}

	if r.Status != result.StatusFinished {
		t.Fatalf("Status = %q", r.Status)
	}
	if r.Error != nil {
		t.Fatalf("Error = %v, want nil", r.Error)
	}
	if r.FinalMessage != "Refactored foo.go and added new.go." {
		t.Fatalf("FinalMessage = %q", r.FinalMessage)
	}

	if r.Metrics.Turns != 4 || r.Metrics.DurationMs != 12000 {
		t.Fatalf("Metrics turns/duration = %d / %d", r.Metrics.Turns, r.Metrics.DurationMs)
	}
	if r.Metrics.CostUSD != nil {
		t.Fatalf("CostUSD = %v, want nil", r.Metrics.CostUSD)
	}
	if len(r.Metrics.ToolsUsed) != 2 || r.Metrics.ToolsUsed[0] != "read" {
		t.Fatalf("ToolsUsed = %#v", r.Metrics.ToolsUsed)
	}
	if len(r.Metrics.ToolCalls) != 2 || r.Metrics.ToolCalls[1].Name != "edit" {
		t.Fatalf("ToolCalls = %#v", r.Metrics.ToolCalls)
	}
	if r.Metrics.Usage.TotalTokens != 1750 {
		t.Fatalf("Usage.TotalTokens = %d", r.Metrics.Usage.TotalTokens)
	}

	if len(r.Skills.Activated) != 1 || r.Skills.Activated[0] != "refactor-helper" {
		t.Fatalf("Skills = %+v", r.Skills)
	}

	if len(r.Outcomes.Files) != 3 {
		t.Fatalf("files len = %d", len(r.Outcomes.Files))
	}
	if r.Outcomes.Files["src/foo.go"].Status != result.FileModified {
		t.Fatalf("foo status = %q", r.Outcomes.Files["src/foo.go"].Status)
	}
	if r.Outcomes.Files["src/new.go"].Status != result.FileCreated {
		t.Fatalf("new status = %q", r.Outcomes.Files["src/new.go"].Status)
	}
	if r.Outcomes.Files["src/gone.go"].Status != result.FileDeleted {
		t.Fatalf("gone status = %q", r.Outcomes.Files["src/gone.go"].Status)
	}
}

func TestToolCallExitCodeJSON(t *testing.T) {
	zero := 0
	one := 1
	cases := []struct {
		name string
		call result.ToolCall
		want string
	}{
		{
			name: "omitted",
			call: result.ToolCall{Name: "shell", Status: result.ToolCallCompleted},
			want: `{"name":"shell","status":"completed"}`,
		},
		{
			name: "zero",
			call: result.ToolCall{Name: "shell", Status: result.ToolCallCompleted, ExitCode: &zero},
			want: `{"name":"shell","status":"completed","exitCode":0}`,
		},
		{
			name: "nonzero",
			call: result.ToolCall{Name: "shell", Status: result.ToolCallCompleted, ExitCode: &one},
			want: `{"name":"shell","status":"completed","exitCode":1}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.call)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(raw) != tc.want {
				t.Fatalf("marshal = %s, want %s", raw, tc.want)
			}
			var got result.ToolCall
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Name != tc.call.Name || got.Status != tc.call.Status {
				t.Fatalf("got = %+v", got)
			}
			switch {
			case tc.call.ExitCode == nil:
				if got.ExitCode != nil {
					t.Fatalf("ExitCode = %v, want nil", got.ExitCode)
				}
			case got.ExitCode == nil:
				t.Fatal("ExitCode is nil")
			case *got.ExitCode != *tc.call.ExitCode:
				t.Fatalf("ExitCode = %d, want %d", *got.ExitCode, *tc.call.ExitCode)
			}
		})
	}
}

func canonicalJSON(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
