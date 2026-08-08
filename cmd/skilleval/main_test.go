package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/history"
	"github.com/daniel-walters/skilleval/summary"
)

func TestVersionDefault(t *testing.T) {
	if version != "dev" {
		t.Fatalf("version = %q, want %q", version, "dev")
	}
}

func TestSplitFlagsAndPositionals(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		boolFlags []string
		wantFlags []string
		wantPos   []string
	}{
		{
			name:      "flags after path",
			args:      []string{"eval.yaml", "--model", "m", "--out", "r.json"},
			wantFlags: []string{"--model", "m", "--out", "r.json"},
			wantPos:   []string{"eval.yaml"},
		},
		{
			name:      "flags before path",
			args:      []string{"--model", "m", "--out", "r.json", "eval.yaml"},
			wantFlags: []string{"--model", "m", "--out", "r.json"},
			wantPos:   []string{"eval.yaml"},
		},
		{
			name:      "equals form",
			args:      []string{"--model=m", "eval.yaml"},
			wantFlags: []string{"--model=m"},
			wantPos:   []string{"eval.yaml"},
		},
		{
			name:      "bool flag before path",
			args:      []string{"--no-history", "eval.yaml"},
			boolFlags: []string{"no-history"},
			wantFlags: []string{"--no-history"},
			wantPos:   []string{"eval.yaml"},
		},
		{
			name:      "bool flag does not swallow following value flag",
			args:      []string{"--no-history", "--baseline", "b.json", "eval.yaml"},
			boolFlags: []string{"no-history", "no-baseline"},
			wantFlags: []string{"--no-history", "--baseline", "b.json"},
			wantPos:   []string{"eval.yaml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, pos, err := splitFlagsAndPositionals(tt.args, tt.boolFlags...)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !reflect.DeepEqual(flags, tt.wantFlags) {
				t.Fatalf("flags = %#v, want %#v", flags, tt.wantFlags)
			}
			if !reflect.DeepEqual(pos, tt.wantPos) {
				t.Fatalf("pos = %#v, want %#v", pos, tt.wantPos)
			}
		})
	}
}

func TestReportVerdict(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		var buf bytes.Buffer
		err := reportVerdict(&buf, checker.Verdict{Passed: true})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got := buf.String(); got != "PASS\n" {
			t.Fatalf("output = %q, want %q", got, "PASS\n")
		}
	})

	t.Run("fail", func(t *testing.T) {
		var buf bytes.Buffer
		err := reportVerdict(&buf, checker.Verdict{
			Passed: false,
			Failures: []checker.Failure{
				{Path: "turns.max", Reason: "turns 12 exceeds max 10"},
				{Path: "finalMessage.contains", Reason: "finalMessage does not contain \"Done\""},
			},
		})
		if err == nil || err.Error() != "check failed" {
			t.Fatalf("err = %v, want check failed", err)
		}
		got := buf.String()
		want := "FAIL\n  turns.max: turns 12 exceeds max 10\n  finalMessage.contains: finalMessage does not contain \"Done\"\n"
		if got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

func TestAttemptOutPath(t *testing.T) {
	if got := attemptOutPath("/tmp/result.json", 1, 1); got != "/tmp/result.json" {
		t.Fatalf("single = %q", got)
	}
	if got := attemptOutPath("/tmp/result.json", 2, 5); got != "/tmp/result-2.json" {
		t.Fatalf("multi = %q", got)
	}
	if got := attemptOutPath("/tmp/out", 3, 3); got != "/tmp/out-3" {
		t.Fatalf("no ext = %q", got)
	}
}

func TestSummaryOutPath(t *testing.T) {
	if got := summaryOutPath("/tmp/result.json"); got != "/tmp/result-summary.json" {
		t.Fatalf("got %q", got)
	}
}

func TestPrintSummary(t *testing.T) {
	turns := 12.5
	cost := 0.4
	var buf bytes.Buffer
	err := printSummary(&buf, summary.Report{
		Attempts:   4,
		Passed:     3,
		PassRate:   0.75,
		AvgTurns:   &turns,
		AvgCostUSD: &cost,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	got := buf.String()
	want := "---\npassRate: 0.75 (3/4)\navgTurns: 12.5\navgCostUSD: 0.4\n"
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestEmitReportWritesSummaryAndCompares(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.json")
	baselinePath := filepath.Join(dir, "baseline.json")
	cost := 1.0
	if err := summary.Write(baselinePath, summary.Report{
		Attempts:   1,
		Passed:     1,
		PassRate:   1,
		AvgCostUSD: &cost,
	}); err != nil {
		t.Fatalf("Write baseline: %v", err)
	}

	histDir := filepath.Join(dir, "hist")
	curCost := 2.0
	var buf bytes.Buffer
	err := emitReport(&buf, outPath, summary.Report{
		Attempts:   1,
		Passed:     1,
		PassRate:   1,
		AvgCostUSD: &curCost,
	}, reportOpts{
		historyDir: histDir,
		baseline:   baselinePath,
		evalName:   "sample",
	})
	if err != nil {
		t.Fatalf("emitReport: %v", err)
	}

	sumPath := filepath.Join(dir, "result-summary.json")
	if _, err := summary.Load(sumPath); err != nil {
		t.Fatalf("Load summary: %v", err)
	}
	if _, err := summary.Load(filepath.Join(histDir, "sample", "latest.json")); err != nil {
		t.Fatalf("Load retained latest: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "vs baseline:") || !strings.Contains(out, "avgCostUSD:") {
		t.Fatalf("missing compare output: %q", out)
	}
	if !strings.Contains(out, "retained ") {
		t.Fatalf("missing retained line: %q", out)
	}
}

func TestEmitReportBaselineBeforeRetainLatest(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.json")
	histDir := filepath.Join(dir, "hist")
	oldCost := 1.0
	if _, err := history.Retain(histDir, "e", summary.Report{
		PassRate:   1,
		AvgCostUSD: &oldCost,
	}); err != nil {
		t.Fatalf("seed history: %v", err)
	}
	baseline := filepath.Join(histDir, "e", "latest.json")

	newCost := 3.0
	var buf bytes.Buffer
	if err := emitReport(&buf, outPath, summary.Report{
		PassRate:   1,
		AvgCostUSD: &newCost,
	}, reportOpts{
		historyDir: histDir,
		baseline:   baseline,
		evalName:   "e",
	}); err != nil {
		t.Fatalf("emitReport: %v", err)
	}
	if !strings.Contains(buf.String(), "1 → 3") {
		t.Fatalf("expected delta against prior latest, got %q", buf.String())
	}
}

func TestEmitReportBaselineSameAsSummaryPath(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.json")
	sumPath := summaryOutPath(outPath)
	oldCost := 1.0
	if err := summary.Write(sumPath, summary.Report{
		PassRate:   1,
		AvgCostUSD: &oldCost,
	}); err != nil {
		t.Fatalf("seed summary: %v", err)
	}

	newCost := 4.0
	var buf bytes.Buffer
	if err := emitReport(&buf, outPath, summary.Report{
		PassRate:   1,
		AvgCostUSD: &newCost,
	}, reportOpts{
		baseline: sumPath,
		evalName: "e",
	}); err != nil {
		t.Fatalf("emitReport: %v", err)
	}
	if !strings.Contains(buf.String(), "1 → 4") {
		t.Fatalf("expected delta against prior summary file, got %q", buf.String())
	}
}

func TestResolveReportOpts(t *testing.T) {
	dir := t.TempDir()
	histDir := filepath.Join(dir, "hist")

	t.Run("default history no latest", func(t *testing.T) {
		opts, err := resolveReportOpts("e", histDir, false, "", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if opts.historyDir != histDir {
			t.Fatalf("historyDir = %q, want %q", opts.historyDir, histDir)
		}
		if opts.baseline != "" {
			t.Fatalf("baseline = %q, want empty", opts.baseline)
		}
		if opts.evalName != "e" {
			t.Fatalf("evalName = %q", opts.evalName)
		}
	})

	t.Run("auto baseline when latest exists", func(t *testing.T) {
		if _, err := history.Retain(histDir, "e", summary.Report{PassRate: 1}); err != nil {
			t.Fatalf("seed: %v", err)
		}
		latest := filepath.Join(histDir, "e", "latest.json")
		opts, err := resolveReportOpts("e", histDir, false, "", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if opts.baseline != latest {
			t.Fatalf("baseline = %q, want %q", opts.baseline, latest)
		}
	})

	t.Run("no-history clears retain and auto baseline", func(t *testing.T) {
		opts, err := resolveReportOpts("e", histDir, true, "", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if opts.historyDir != "" || opts.baseline != "" {
			t.Fatalf("opts = %+v, want empty history and baseline", opts)
		}
	})

	t.Run("no-history with explicit baseline", func(t *testing.T) {
		base := filepath.Join(dir, "base.json")
		if err := summary.Write(base, summary.Report{PassRate: 1}); err != nil {
			t.Fatalf("Write: %v", err)
		}
		opts, err := resolveReportOpts("e", histDir, true, base, false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if opts.historyDir != "" {
			t.Fatalf("historyDir = %q, want empty", opts.historyDir)
		}
		if opts.baseline != base {
			t.Fatalf("baseline = %q, want %q", opts.baseline, base)
		}
	})

	t.Run("no-baseline retains without compare", func(t *testing.T) {
		opts, err := resolveReportOpts("e", histDir, false, "", true)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if opts.historyDir != histDir {
			t.Fatalf("historyDir = %q, want %q", opts.historyDir, histDir)
		}
		if opts.baseline != "" {
			t.Fatalf("baseline = %q, want empty", opts.baseline)
		}
	})

	t.Run("explicit baseline wins over latest", func(t *testing.T) {
		other := filepath.Join(dir, "other.json")
		if err := summary.Write(other, summary.Report{PassRate: 0.5}); err != nil {
			t.Fatalf("Write: %v", err)
		}
		opts, err := resolveReportOpts("e", histDir, false, other, false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if opts.baseline != other {
			t.Fatalf("baseline = %q, want %q", opts.baseline, other)
		}
	})

	t.Run("no-baseline conflicts with baseline", func(t *testing.T) {
		_, err := resolveReportOpts("e", histDir, false, filepath.Join(dir, "x.json"), true)
		if err == nil || !strings.Contains(err.Error(), "--no-baseline conflicts with --baseline") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("empty historyDir uses default when retaining", func(t *testing.T) {
		opts, err := resolveReportOpts("e", "", false, "", false)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		want, err := filepath.Abs(defaultHistoryDir)
		if err != nil {
			t.Fatalf("Abs: %v", err)
		}
		if opts.historyDir != want {
			t.Fatalf("historyDir = %q, want %q", opts.historyDir, want)
		}
	})
}

func TestEmitReportAutoBaselineBeforeRetain(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.json")
	histDir := filepath.Join(dir, "hist")
	oldCost := 1.0
	if _, err := history.Retain(histDir, "e", summary.Report{
		PassRate:   1,
		AvgCostUSD: &oldCost,
	}); err != nil {
		t.Fatalf("seed history: %v", err)
	}

	opts, err := resolveReportOpts("e", histDir, false, "", false)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	newCost := 3.0
	var buf bytes.Buffer
	if err := emitReport(&buf, outPath, summary.Report{
		PassRate:   1,
		AvgCostUSD: &newCost,
	}, opts); err != nil {
		t.Fatalf("emitReport: %v", err)
	}
	if !strings.Contains(buf.String(), "1 → 3") {
		t.Fatalf("expected delta against prior latest, got %q", buf.String())
	}
}
