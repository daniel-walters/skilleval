package eval

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestToolCallNamesMatch(t *testing.T) {
	n := ToolCallNames{"write", "edit"}
	if !n.Match("write") || !n.Match("edit") || n.Match("read") {
		t.Fatalf("Match mismatch for %v", n)
	}
	one := ToolCallNames{"edit"}
	if !one.Match("edit") || one.Match("write") {
		t.Fatalf("Match mismatch for %v", one)
	}
}

func TestToolCallNamesString(t *testing.T) {
	if got := (ToolCallNames{"edit"}).String(); got != `"edit"` {
		t.Fatalf("one name String = %s, want %q", got, `"edit"`)
	}
	if got := (ToolCallNames{"write", "edit"}).String(); got != `["write","edit"]` {
		t.Fatalf("list String = %s, want %s", got, `["write","edit"]`)
	}
}

func TestToolCallNamesYAMLRoundTrip(t *testing.T) {
	tests := []struct {
		in   string
		want ToolCallNames
		out  string
	}{
		{in: "edit", want: ToolCallNames{"edit"}, out: "edit\n"},
		{in: "[write, edit]", want: ToolCallNames{"write", "edit"}, out: "- write\n- edit\n"},
	}
	for _, tt := range tests {
		var n ToolCallNames
		if err := yaml.Unmarshal([]byte(tt.in), &n); err != nil {
			t.Fatalf("unmarshal %q: %v", tt.in, err)
		}
		if len(n) != len(tt.want) {
			t.Fatalf("unmarshal %q: got %v, want %v", tt.in, n, tt.want)
		}
		for i := range n {
			if n[i] != tt.want[i] {
				t.Fatalf("unmarshal %q: got %v, want %v", tt.in, n, tt.want)
			}
		}
		b, err := yaml.Marshal(n)
		if err != nil {
			t.Fatalf("marshal %v: %v", n, err)
		}
		if string(b) != tt.out {
			t.Fatalf("marshal %v = %q, want %q", n, b, tt.out)
		}
	}
}

func TestToolCallNamesYAMLRejectsMapping(t *testing.T) {
	var n ToolCallNames
	err := yaml.Unmarshal([]byte("{a: b}"), &n)
	if err == nil {
		t.Fatal("expected error for mapping")
	}
	if !strings.Contains(err.Error(), "string or a list") {
		t.Fatalf("error = %v", err)
	}
}
