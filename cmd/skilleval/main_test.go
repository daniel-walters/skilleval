package main

import (
	"reflect"
	"testing"
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
