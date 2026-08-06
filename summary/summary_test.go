package summary_test

import (
	"errors"
	"testing"

	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/result"
	"github.com/daniel-walters/skilleval/summary"
)

func TestAggregateEmpty(t *testing.T) {
	r := summary.Aggregate(nil)
	if r.Attempts != 0 || r.Passed != 0 || r.PassRate != 0 {
		t.Fatalf("empty report = %#v", r)
	}
	if r.AvgTurns != nil || r.AvgCostUSD != nil || r.AvgDurationMs != nil {
		t.Fatalf("empty averages should be nil: %#v", r)
	}
}

func TestAggregatePassRate(t *testing.T) {
	r := summary.Aggregate([]summary.Attempt{
		{Verdict: checker.Verdict{Passed: true}, Result: finished(10, 100, ptr(0.5))},
		{Verdict: checker.Verdict{Passed: false}, Result: finished(20, 200, ptr(1.5))},
		{Err: errors.New("agent boom")},
		{Verdict: checker.Verdict{Passed: true}, Result: finished(30, 300, nil)},
	})
	if r.Attempts != 4 {
		t.Fatalf("Attempts = %d, want 4", r.Attempts)
	}
	if r.Passed != 2 {
		t.Fatalf("Passed = %d, want 2", r.Passed)
	}
	if r.PassRate != 0.5 {
		t.Fatalf("PassRate = %g, want 0.5", r.PassRate)
	}
}

func TestAggregateAveragesFinishedOnly(t *testing.T) {
	r := summary.Aggregate([]summary.Attempt{
		{Verdict: checker.Verdict{Passed: true}, Result: finished(10, 100, ptr(1.0))},
		{Verdict: checker.Verdict{Passed: false}, Result: &result.Result{
			Status: result.StatusError,
			Metrics: result.Metrics{
				Turns:      999,
				DurationMs: 9999,
				CostUSD:    ptr(99.0),
			},
		}},
		{Verdict: checker.Verdict{Passed: true}, Result: finished(30, 300, nil)},
		{Err: errors.New("no result")},
	})

	if r.AvgTurns == nil || *r.AvgTurns != 20 {
		t.Fatalf("AvgTurns = %v, want 20", r.AvgTurns)
	}
	if r.AvgDurationMs == nil || *r.AvgDurationMs != 200 {
		t.Fatalf("AvgDurationMs = %v, want 200", r.AvgDurationMs)
	}
	// Only one finished attempt had non-nil cost.
	if r.AvgCostUSD == nil || *r.AvgCostUSD != 1.0 {
		t.Fatalf("AvgCostUSD = %v, want 1.0", r.AvgCostUSD)
	}
}

func TestAggregateErrorOnlyNoAverages(t *testing.T) {
	r := summary.Aggregate([]summary.Attempt{
		{Err: errors.New("a")},
		{Err: errors.New("b")},
	})
	if r.PassRate != 0 {
		t.Fatalf("PassRate = %g, want 0", r.PassRate)
	}
	if r.AvgTurns != nil || r.AvgCostUSD != nil || r.AvgDurationMs != nil {
		t.Fatalf("want nil averages, got %#v", r)
	}
}

func TestMeetsPassRate(t *testing.T) {
	r := summary.Report{PassRate: 0.8}
	if !summary.MeetsPassRate(r, 0.8) {
		t.Fatal("0.8 should meet min 0.8")
	}
	if !summary.MeetsPassRate(r, 0.5) {
		t.Fatal("0.8 should meet min 0.5")
	}
	if summary.MeetsPassRate(r, 0.81) {
		t.Fatal("0.8 should not meet min 0.81")
	}
}

func finished(turns int, durationMs int64, cost *float64) *result.Result {
	return &result.Result{
		Status: result.StatusFinished,
		Metrics: result.Metrics{
			Turns:      turns,
			DurationMs: durationMs,
			CostUSD:    cost,
		},
	}
}

func ptr(v float64) *float64 { return &v }
