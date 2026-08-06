// Package summary aggregates multi-attempt eval outcomes into a batch report.
package summary

import (
	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/result"
)

// Attempt is one scheduled try in a multi-run batch.
type Attempt struct {
	// Result is nil when the runner failed before producing a Result.
	Result *result.Result
	// Verdict is the per-attempt checker outcome (zero when Err is set).
	Verdict checker.Verdict
	// Err is a runner/harness error that prevented a Result.
	Err error
}

// Report is the aggregate score across a batch of attempts.
type Report struct {
	Attempts int     `json:"attempts"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"passRate"`

	// AvgTurns is nil when no finished attempts contributed.
	AvgTurns *float64 `json:"avgTurns,omitempty"`
	// AvgCostUSD is nil when no finished attempt had a non-nil cost.
	AvgCostUSD *float64 `json:"avgCostUSD,omitempty"`
	// AvgDurationMs is nil when no finished attempts contributed.
	AvgDurationMs *float64 `json:"avgDurationMs,omitempty"`
}

// Aggregate computes pass rate and averages across attempts.
//
// Pass rate is Passed / len(attempts); an attempt counts as passed only when
// Err is nil and Verdict.Passed is true.
// Metric averages use only attempts with Result.Status == finished.
// Cost averages skip nil CostUSD values.
func Aggregate(attempts []Attempt) Report {
	r := Report{Attempts: len(attempts)}
	if len(attempts) == 0 {
		return r
	}

	var turnsSum float64
	var turnsN int
	var costSum float64
	var costN int
	var durSum float64
	var durN int

	for _, a := range attempts {
		if a.Err == nil && a.Verdict.Passed {
			r.Passed++
		}
		if a.Result == nil || a.Result.Status != result.StatusFinished {
			continue
		}
		turnsSum += float64(a.Result.Metrics.Turns)
		turnsN++
		durSum += float64(a.Result.Metrics.DurationMs)
		durN++
		if a.Result.Metrics.CostUSD != nil {
			costSum += *a.Result.Metrics.CostUSD
			costN++
		}
	}

	r.PassRate = float64(r.Passed) / float64(r.Attempts)
	if turnsN > 0 {
		v := turnsSum / float64(turnsN)
		r.AvgTurns = &v
	}
	if costN > 0 {
		v := costSum / float64(costN)
		r.AvgCostUSD = &v
	}
	if durN > 0 {
		v := durSum / float64(durN)
		r.AvgDurationMs = &v
	}
	return r
}

// MeetsPassRate reports whether r.PassRate is at least min.
func MeetsPassRate(r Report, min float64) bool {
	return r.PassRate >= min
}
