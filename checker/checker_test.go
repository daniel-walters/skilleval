package checker_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
)

func TestCheckFixtures(t *testing.T) {
	tests := []struct {
		name         string
		wantPassed   bool
		wantFailPath []string
	}{
		{name: "pass", wantPassed: true},
		{name: "empty-expects", wantPassed: true},
		{name: "pass-file-deleted", wantPassed: true},
		{name: "error-status", wantPassed: false, wantFailPath: []string{"run.status"}},
		{name: "cancelled-status", wantPassed: false, wantFailPath: []string{"run.status"}},
		{name: "fail-turns", wantPassed: false, wantFailPath: []string{"turns.max"}},
		{name: "fail-null-cost", wantPassed: false, wantFailPath: []string{"costUSD.max"}},
		{name: "fail-cost-exceeded", wantPassed: false, wantFailPath: []string{"costUSD.max"}},
		{
			name:         "fail-tools",
			wantPassed:   false,
			wantFailPath: []string{"toolsUsed.includes", "toolsUsed.excludes"},
		},
		{name: "fail-skills", wantPassed: false, wantFailPath: []string{"skills.activated.includes"}},
		{name: "fail-file-status", wantPassed: false, wantFailPath: []string{"files[src/foo.go].status"}},
		{name: "fail-file-deleted", wantPassed: false, wantFailPath: []string{"files[src/gone.go].status"}},
		{name: "fail-deleted-contains", wantPassed: false, wantFailPath: []string{"files[src/gone.go]"}},
		{
			name:         "fail-final-message",
			wantPassed:   false,
			wantFailPath: []string{"finalMessage.contains", "finalMessage.equals"},
		},
		{name: "fail-file-contains", wantPassed: false, wantFailPath: []string{"files[src/foo.go].contains"}},
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
