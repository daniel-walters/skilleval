package history_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-walters/skilleval/history"
	"github.com/daniel-walters/skilleval/summary"
)

func TestRetainWritesTimestampAndLatest(t *testing.T) {
	dir := t.TempDir()
	r := summary.Report{Attempts: 2, Passed: 1, PassRate: 0.5}
	path, err := history.Retain(dir, "my-eval", r)
	if err != nil {
		t.Fatalf("Retain: %v", err)
	}
	if !strings.HasPrefix(path, filepath.Join(dir, "my-eval")+string(filepath.Separator)) {
		t.Fatalf("path %q not under eval dir", path)
	}
	if filepath.Base(path) == "latest.json" {
		t.Fatal("timestamped path should not be latest.json")
	}

	got, err := summary.Load(path)
	if err != nil {
		t.Fatalf("Load timestamped: %v", err)
	}
	if got.PassRate != 0.5 {
		t.Fatalf("timestamped PassRate = %g", got.PassRate)
	}

	latest, err := summary.Load(filepath.Join(dir, "my-eval", "latest.json"))
	if err != nil {
		t.Fatalf("Load latest: %v", err)
	}
	if latest.PassRate != 0.5 || latest.Attempts != 2 {
		t.Fatalf("latest = %#v", latest)
	}
}

func TestRetainUpdatesLatest(t *testing.T) {
	dir := t.TempDir()
	if _, err := history.Retain(dir, "e", summary.Report{PassRate: 1}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := history.Retain(dir, "e", summary.Report{PassRate: 0.25}); err != nil {
		t.Fatalf("second: %v", err)
	}
	latest, err := summary.Load(filepath.Join(dir, "e", "latest.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if latest.PassRate != 0.25 {
		t.Fatalf("PassRate = %g, want 0.25", latest.PassRate)
	}
}

func TestRetainRejectsEmpty(t *testing.T) {
	if _, err := history.Retain("", "e", summary.Report{}); err == nil {
		t.Fatal("expected error for empty dir")
	}
	if _, err := history.Retain(t.TempDir(), "", summary.Report{}); err == nil {
		t.Fatal("expected error for empty eval name")
	}
}
