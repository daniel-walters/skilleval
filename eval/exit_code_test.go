package eval

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExitCodeExpectMatch(t *testing.T) {
	zero := 0
	one := 1
	e := ExitCodeExpect{0, 1}
	if !e.Match(&zero) || !e.Match(&one) {
		t.Fatalf("Match missed %v", e)
	}
	two := 2
	if e.Match(&two) || e.Match(nil) || (ExitCodeExpect{}).Match(&zero) {
		t.Fatal("Match should fail for missing/empty")
	}
}

func TestExitCodeExpectYAMLRoundTrip(t *testing.T) {
	tests := []struct {
		in   string
		want ExitCodeExpect
		out  string
	}{
		{in: "0", want: ExitCodeExpect{0}, out: "0\n"},
		{in: "[0, 1]", want: ExitCodeExpect{0, 1}, out: "- 0\n- 1\n"},
	}
	for _, tt := range tests {
		var e ExitCodeExpect
		if err := yaml.Unmarshal([]byte(tt.in), &e); err != nil {
			t.Fatalf("unmarshal %q: %v", tt.in, err)
		}
		if len(e) != len(tt.want) {
			t.Fatalf("unmarshal %q: got %v, want %v", tt.in, e, tt.want)
		}
		for i := range e {
			if e[i] != tt.want[i] {
				t.Fatalf("unmarshal %q: got %v, want %v", tt.in, e, tt.want)
			}
		}
		b, err := yaml.Marshal(e)
		if err != nil {
			t.Fatalf("marshal %v: %v", e, err)
		}
		if string(b) != tt.out {
			t.Fatalf("marshal %v = %q, want %q", e, b, tt.out)
		}
	}
}

func TestExitCodeExpectYAMLRejectsNonInteger(t *testing.T) {
	var e ExitCodeExpect
	err := yaml.Unmarshal([]byte("1.5"), &e)
	if err == nil {
		t.Fatal("expected error for float")
	}
	if !strings.Contains(err.Error(), "integer") {
		t.Fatalf("error = %v", err)
	}
	err = yaml.Unmarshal([]byte("{a: b}"), &e)
	if err == nil {
		t.Fatal("expected error for mapping")
	}
}

func TestIsShellToolName(t *testing.T) {
	if !IsShellToolName("shell") || !IsShellToolName("Bash") {
		t.Fatal("shell and Bash should be shell tools")
	}
	if IsShellToolName("bash") || IsShellToolName("edit") {
		t.Fatal("bash/edit should not count")
	}
}
