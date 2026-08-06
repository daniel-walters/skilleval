package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/daniel-walters/skilleval/checker"
	"github.com/daniel-walters/skilleval/summary"
)

func TestSplitFlagsAndPositionals(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, pos, err := splitFlagsAndPositionals(tt.args)
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
