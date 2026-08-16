package checker_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
	"gopkg.in/yaml.v3"
)

func TestCheckFixtures(t *testing.T) {
	tests := []struct {
		name         string
		wantPassed   bool
		wantFailPath []string
	}{
		{name: "pass", wantPassed: true},
		{name: "pass-regex", wantPassed: true},
		{name: "pass-numeric-bounds", wantPassed: true},
		{name: "pass-extended-metrics", wantPassed: true},
		{name: "pass-tool-calls-order", wantPassed: true},
		{name: "pass-tool-calls-order-first-name", wantPassed: true},
		{name: "pass-tool-calls-order-later-name", wantPassed: true},
		{name: "pass-empty-bounds", wantPassed: true},
		{name: "empty-expects", wantPassed: true},
		{name: "pass-file-deleted", wantPassed: true},
		{name: "error-status", wantPassed: false, wantFailPath: []string{"run.status"}},
		{name: "cancelled-status", wantPassed: false, wantFailPath: []string{"run.status"}},
		{name: "fail-turns", wantPassed: false, wantFailPath: []string{"turns.max"}},
		{name: "fail-turns-min", wantPassed: false, wantFailPath: []string{"turns.min"}},
		{name: "fail-turns-gt", wantPassed: false, wantFailPath: []string{"turns.gt"}},
		{name: "fail-turns-lt", wantPassed: false, wantFailPath: []string{"turns.lt"}},
		{name: "fail-turns-eq", wantPassed: false, wantFailPath: []string{"turns.eq"}},
		{name: "fail-turns-combo", wantPassed: false, wantFailPath: []string{"turns.min", "turns.max"}},
		{name: "fail-duration-ms", wantPassed: false, wantFailPath: []string{"durationMs.max"}},
		{name: "fail-tool-calls", wantPassed: false, wantFailPath: []string{"toolCalls.min"}},
		{name: "fail-tool-calls-order", wantPassed: false, wantFailPath: []string{"toolCalls.order[1]"}},
		{name: "fail-tool-calls-order-names", wantPassed: false, wantFailPath: []string{"toolCalls.order[0]"}},
		{name: "fail-tool-calls-named", wantPassed: false, wantFailPath: []string{"toolCalls.named.edit.min"}},
		{name: "fail-tool-calls-args", wantPassed: false, wantFailPath: []string{"toolCalls.order[0]"}},
		{
			name:         "fail-usage",
			wantPassed:   false,
			wantFailPath: []string{"usage.inputTokens.max", "usage.totalTokens.lt"},
		},
		{name: "fail-null-cost", wantPassed: false, wantFailPath: []string{"costUSD.max"}},
		{name: "fail-null-cost-bounds", wantPassed: false, wantFailPath: []string{"costUSD.min", "costUSD.eq"}},
		{name: "fail-cost-exceeded", wantPassed: false, wantFailPath: []string{"costUSD.max"}},
		{name: "fail-cost-min", wantPassed: false, wantFailPath: []string{"costUSD.min"}},
		{name: "fail-cost-gt", wantPassed: false, wantFailPath: []string{"costUSD.gt"}},
		{name: "fail-cost-lt", wantPassed: false, wantFailPath: []string{"costUSD.lt"}},
		{name: "fail-cost-eq", wantPassed: false, wantFailPath: []string{"costUSD.eq"}},
		{
			name:         "fail-tools",
			wantPassed:   false,
			wantFailPath: []string{"toolsUsed.includes", "toolsUsed.excludes"},
		},
		{name: "fail-skills", wantPassed: false, wantFailPath: []string{"skills.activated.includes"}},
		{name: "fail-skills-excludes", wantPassed: false, wantFailPath: []string{"skills.activated.excludes"}},
		{name: "fail-file-status", wantPassed: false, wantFailPath: []string{"files[src/foo.go].status"}},
		{name: "fail-file-missing", wantPassed: false, wantFailPath: []string{"files[src/foo.go].status"}},
		{name: "fail-file-deleted", wantPassed: false, wantFailPath: []string{"files[src/gone.go].status"}},
		{name: "fail-deleted-contains", wantPassed: false, wantFailPath: []string{"files[src/gone.go]"}},
		{name: "fail-workspace-missing", wantPassed: false, wantFailPath: []string{"files[src/foo.go]"}},
		{
			name:         "fail-final-message",
			wantPassed:   false,
			wantFailPath: []string{"finalMessage.contains", "finalMessage.equals"},
		},
		{
			name:         "fail-final-message-regex",
			wantPassed:   false,
			wantFailPath: []string{"finalMessage.contains", "finalMessage.equals"},
		},
		{name: "fail-file-contains", wantPassed: false, wantFailPath: []string{"files[src/foo.go].contains"}},
		{name: "fail-file-contains-regex", wantPassed: false, wantFailPath: []string{"files[src/foo.go].contains"}},
		{
			name:         "fail-file-excludes",
			wantPassed:   false,
			wantFailPath: []string{"files[src/foo.go].excludes", "files[src/foo.go].excludes"},
		},
		{
			name:         "fail-many",
			wantPassed:   false,
			wantFailPath: []string{"turns.max", "costUSD.max", "finalMessage.contains"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join("testdata", tt.name)
			e, err := eval.Load(filepath.Join(dir, "eval.yaml"))
			if err != nil {
				t.Fatalf("eval.Load: %v", err)
			}
			r := loadResult(t, filepath.Join(dir, "result.json"))
			workspace := filepath.Join(dir, "workspace")
			if _, err := os.Stat(workspace); err != nil {
				workspace = ""
			}

			got := checker.Check(r, e.Expects, workspace)
			if got.Passed != tt.wantPassed {
				t.Fatalf("Passed = %v, want %v; failures=%v", got.Passed, tt.wantPassed, got.Failures)
			}
			gotPaths := failurePaths(got.Failures)
			wantPaths := append([]string(nil), tt.wantFailPath...)
			sort.Strings(gotPaths)
			sort.Strings(wantPaths)
			if len(gotPaths) != len(wantPaths) {
				t.Fatalf("failure paths = %v, want %v", gotPaths, wantPaths)
			}
			for i := range wantPaths {
				if gotPaths[i] != wantPaths[i] {
					t.Fatalf("failure paths = %v, want %v", gotPaths, wantPaths)
				}
			}
			if tt.name == "error-status" || tt.name == "cancelled-status" {
				if len(got.Failures) != 1 || got.Failures[0].Path != "run.status" {
					t.Fatalf("non-finished run should only report run.status, got %#v", got.Failures)
				}
			}
		})
	}
}

func TestCheckNilResult(t *testing.T) {
	got := checker.Check(nil, eval.Expects{}, "")
	if got.Passed || len(got.Failures) != 1 || got.Failures[0].Path != "run" {
		t.Fatalf("unexpected verdict: %#v", got)
	}
}

func TestCheckRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("TOPSECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &result.Result{
		SchemaVersion: result.SchemaVersion,
		Status:        result.StatusFinished,
		Outcomes: result.Outcomes{Files: map[string]result.FileOutcome{
			"../secret.txt": {Status: result.FileModified},
		}},
	}
	got := checker.Check(r, eval.Expects{
		Files: map[string]eval.FileExpect{
			"../secret.txt": {Contains: mustLiteralMatch(t, "TOPSECRET")},
		},
	}, workspace)
	if got.Passed {
		t.Fatal("expected failure for path traversal")
	}
	for _, f := range got.Failures {
		if f.Path == "files[../secret.txt]" && f.Reason != "" {
			return
		}
	}
	t.Fatalf("want traversal failure, got %#v", got.Failures)
}

func TestCheckContentRequiresFileOutcome(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "seeded.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &result.Result{
		SchemaVersion: result.SchemaVersion,
		Status:        result.StatusFinished,
		Outcomes:      result.Outcomes{Files: map[string]result.FileOutcome{}},
	}
	got := checker.Check(r, eval.Expects{
		Files: map[string]eval.FileExpect{
			"seeded.go": {Equals: mustLiteralMatch(t, "package main\n")},
		},
	}, workspace)
	if got.Passed {
		t.Fatal("content expect without outcomes.files entry should fail")
	}
	if len(got.Failures) != 1 || got.Failures[0].Path != "files[seeded.go]" {
		t.Fatalf("failures = %#v", got.Failures)
	}
}

func mustLiteralMatch(t *testing.T, s string) eval.StringMatch {
	t.Helper()
	var m eval.StringMatch
	if err := yaml.Unmarshal([]byte(strconv.Quote(s)), &m); err != nil {
		t.Fatalf("StringMatch: %v", err)
	}
	return m
}

func loadResult(t *testing.T, path string) *result.Result {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var r result.Result
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return &r
}

func failurePaths(failures []checker.Failure) []string {
	paths := make([]string, len(failures))
	for i, f := range failures {
		paths[i] = f.Path
	}
	return paths
}
