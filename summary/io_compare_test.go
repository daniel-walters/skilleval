package summary_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-walters/skilleval/internal/termcolor"
	"github.com/daniel-walters/skilleval/summary"
)

func TestWriteLoadRoundTrip(t *testing.T) {
	r := summary.Report{
		Attempts:      10,
		Passed:        8,
		PassRate:      0.8,
		AvgTurns:      ptr(12.5),
		AvgCostUSD:    ptr(1.25),
		AvgDurationMs: ptr(1000.0),
	}
	path := filepath.Join(t.TempDir(), "summary.json")
	if err := summary.Write(path, r); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := summary.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Attempts != 10 || got.Passed != 8 || got.PassRate != 0.8 {
		t.Fatalf("got %#v", got)
	}
	if got.AvgTurns == nil || *got.AvgTurns != 12.5 {
		t.Fatalf("AvgTurns = %v", got.AvgTurns)
	}
	if got.AvgCostUSD == nil || *got.AvgCostUSD != 1.25 {
		t.Fatalf("AvgCostUSD = %v", got.AvgCostUSD)
	}
	if got.AvgDurationMs == nil || *got.AvgDurationMs != 1000 {
		t.Fatalf("AvgDurationMs = %v", got.AvgDurationMs)
	}
}

func TestLoadRejectsBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := summary.Load(path); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestComparePassRateAndCost(t *testing.T) {
	baseline := summary.Report{
		PassRate:   1.0,
		AvgCostUSD: ptr(1.0),
		AvgTurns:   ptr(10.0),
	}
	current := summary.Report{
		PassRate:   0.5,
		AvgCostUSD: ptr(2.0),
		AvgTurns:   ptr(12.0),
	}
	d := summary.Compare(current, baseline)
	if d.PassRate.Abs == nil || *d.PassRate.Abs != -0.5 {
		t.Fatalf("passRate Abs = %v, want -0.5", d.PassRate.Abs)
	}
	if d.AvgCostUSD.Abs == nil || *d.AvgCostUSD.Abs != 1.0 {
		t.Fatalf("avgCostUSD Abs = %v, want 1.0", d.AvgCostUSD.Abs)
	}
	if d.AvgTurns.Abs == nil || *d.AvgTurns.Abs != 2.0 {
		t.Fatalf("avgTurns Abs = %v, want 2.0", d.AvgTurns.Abs)
	}
}

func TestCompareNilCosts(t *testing.T) {
	baseline := summary.Report{PassRate: 1, AvgCostUSD: nil}
	current := summary.Report{PassRate: 1, AvgCostUSD: ptr(1.5)}
	d := summary.Compare(current, baseline)
	if d.AvgCostUSD.Abs != nil {
		t.Fatalf("Abs should be nil when baseline cost absent, got %v", d.AvgCostUSD.Abs)
	}
	if d.AvgCostUSD.Current == nil || *d.AvgCostUSD.Current != 1.5 {
		t.Fatalf("Current = %v", d.AvgCostUSD.Current)
	}
	if d.AvgCostUSD.Baseline != nil {
		t.Fatalf("Baseline should be nil")
	}
}

func TestCompareEmptyAverages(t *testing.T) {
	d := summary.Compare(summary.Report{PassRate: 0}, summary.Report{PassRate: 0})
	if d.AvgTurns.Baseline != nil || d.AvgTurns.Current != nil {
		t.Fatalf("turns should be omitted: %#v", d.AvgTurns)
	}
	if d.AvgCostUSD.Baseline != nil || d.AvgCostUSD.Current != nil {
		t.Fatalf("cost should be omitted: %#v", d.AvgCostUSD)
	}
}

func TestFormatDiff(t *testing.T) {
	d := summary.Compare(
		summary.Report{PassRate: 0.5, AvgCostUSD: ptr(2.0)},
		summary.Report{PassRate: 1.0, AvgCostUSD: ptr(1.0)},
	)
	var buf bytes.Buffer
	if err := summary.FormatDiff(&buf, d); err != nil {
		t.Fatalf("FormatDiff: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "vs baseline:") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "passRate:") || !strings.Contains(out, "avgCostUSD:") {
		t.Fatalf("missing metrics: %q", out)
	}
	if !strings.Contains(out, "1 → 2") {
		t.Fatalf("missing cost arrow: %q", out)
	}
}

func TestFormatDiffAbsentMetric(t *testing.T) {
	d := summary.Compare(
		summary.Report{PassRate: 1, AvgCostUSD: ptr(1.0)},
		summary.Report{PassRate: 1},
	)
	var buf bytes.Buffer
	if err := summary.FormatDiff(&buf, d); err != nil {
		t.Fatalf("FormatDiff: %v", err)
	}
	if !strings.Contains(buf.String(), "— → 1") {
		t.Fatalf("want absent→present, got %q", buf.String())
	}
}

func TestFormatDiffColors(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("NO_COLOR", "")

	// Higher passRate + lower cost = improvements (green).
	// Lower passRate + higher cost = regressions (red).
	improve := summary.Compare(
		summary.Report{PassRate: 1.0, AvgCostUSD: ptr(0.5)},
		summary.Report{PassRate: 0.5, AvgCostUSD: ptr(1.0)},
	)
	regress := summary.Compare(
		summary.Report{PassRate: 0.5, AvgCostUSD: ptr(2.0)},
		summary.Report{PassRate: 1.0, AvgCostUSD: ptr(1.0)},
	)

	readDiff := func(t *testing.T, d summary.Diff) string {
		t.Helper()
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := summary.FormatDiff(w, d); err != nil {
			_ = w.Close()
			_ = r.Close()
			t.Fatal(err)
		}
		_ = w.Close()
		out, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}

	gotImprove := readDiff(t, improve)
	if !strings.Contains(gotImprove, "vs baseline:") {
		t.Fatalf("missing header: %q", gotImprove)
	}
	if !strings.Contains(gotImprove, termcolor.Green) {
		t.Fatalf("want green improvements: %q", gotImprove)
	}
	if strings.Contains(gotImprove, termcolor.Red) {
		t.Fatalf("unexpected red in improvements: %q", gotImprove)
	}

	gotRegress := readDiff(t, regress)
	if !strings.Contains(gotRegress, termcolor.Red) {
		t.Fatalf("want red regressions: %q", gotRegress)
	}
	if strings.Contains(gotRegress, termcolor.Green) {
		t.Fatalf("unexpected green in regressions: %q", gotRegress)
	}

	// Zero delta stays uncolored.
	zero := summary.Compare(
		summary.Report{PassRate: 1},
		summary.Report{PassRate: 1},
	)
	gotZero := readDiff(t, zero)
	if strings.Contains(gotZero, termcolor.Green) || strings.Contains(gotZero, termcolor.Red) {
		t.Fatalf("zero delta should be plain: %q", gotZero)
	}
}
