package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/daniel-walters/skilleval/checker"
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
