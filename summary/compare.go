package summary

import (
	"fmt"
	"io"
)

// MetricDelta is one numeric field compared across two Reports.
type MetricDelta struct {
	// Baseline is nil when the baseline Report omitted the metric.
	Baseline *float64
	// Current is nil when the current Report omitted the metric.
	Current *float64
	// Abs is Current - Baseline when both are present.
	Abs *float64
	// Rel is Abs / Baseline when both are present and Baseline != 0.
	Rel *float64
}

// Diff is the comparison of a current Report against a baseline.
type Diff struct {
	PassRate      MetricDelta
	AvgTurns      MetricDelta
	AvgCostUSD    MetricDelta
	AvgDurationMs MetricDelta
}

// Compare returns deltas for key aggregates between current and baseline.
func Compare(current, baseline Report) Diff {
	return Diff{
		PassRate:      metricDelta(ptr(current.PassRate), ptr(baseline.PassRate)),
		AvgTurns:      metricDelta(current.AvgTurns, baseline.AvgTurns),
		AvgCostUSD:    metricDelta(current.AvgCostUSD, baseline.AvgCostUSD),
		AvgDurationMs: metricDelta(current.AvgDurationMs, baseline.AvgDurationMs),
	}
}

func metricDelta(current, baseline *float64) MetricDelta {
	d := MetricDelta{Baseline: baseline, Current: current}
	if current != nil && baseline != nil {
		abs := *current - *baseline
		d.Abs = &abs
		if *baseline != 0 {
			rel := abs / *baseline
			d.Rel = &rel
		}
	}
	return d
}

func ptr(v float64) *float64 { return &v }

// FormatDiff writes a short human-readable comparison block to w.
func FormatDiff(w io.Writer, d Diff) error {
	if _, err := fmt.Fprintln(w, "vs baseline:"); err != nil {
		return err
	}
	if err := formatMetric(w, "passRate", d.PassRate); err != nil {
		return err
	}
	if err := formatMetric(w, "avgCostUSD", d.AvgCostUSD); err != nil {
		return err
	}
	if err := formatMetric(w, "avgTurns", d.AvgTurns); err != nil {
		return err
	}
	return formatMetric(w, "avgDurationMs", d.AvgDurationMs)
}

func formatMetric(w io.Writer, name string, m MetricDelta) error {
	if m.Baseline == nil && m.Current == nil {
		return nil
	}
	left := formatOptional(m.Baseline)
	right := formatOptional(m.Current)
	if m.Abs == nil {
		_, err := fmt.Fprintf(w, "  %s: %s → %s\n", name, left, right)
		return err
	}
	if m.Rel != nil {
		_, err := fmt.Fprintf(w, "  %s: %s → %s (%+g / %+.1f%%)\n", name, left, right, *m.Abs, *m.Rel*100)
		return err
	}
	_, err := fmt.Fprintf(w, "  %s: %s → %s (%+g)\n", name, left, right, *m.Abs)
	return err
}

func formatOptional(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%g", *v)
}
