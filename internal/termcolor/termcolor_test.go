package termcolor_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/daniel-walters/skilleval/internal/termcolor"
)

func TestEnabledBufferNever(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	var buf bytes.Buffer
	if termcolor.Enabled(&buf) {
		t.Fatal("bytes.Buffer must stay uncolored")
	}
}

func TestEnabledNoColorWins(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("NO_COLOR", "1")
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	if termcolor.Enabled(w) {
		t.Fatal("NO_COLOR must disable color")
	}
}

func TestEnabledForceColorOnPipe(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("NO_COLOR", "")
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	if !termcolor.Enabled(w) {
		t.Fatal("FORCE_COLOR on *os.File should enable color")
	}
}

func TestWrap(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("NO_COLOR", "")
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })

	got := termcolor.Wrap(w, termcolor.Green, "PASS")
	if !strings.HasPrefix(got, termcolor.Green) || !strings.HasSuffix(got, termcolor.Reset) {
		t.Fatalf("wrap = %q", got)
	}
	if termcolor.Wrap(&bytes.Buffer{}, termcolor.Green, "PASS") != "PASS" {
		t.Fatal("buffer wrap should be plain")
	}
}

func TestWrapRoundTripWrite(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("NO_COLOR", "")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })

	line := termcolor.Wrap(w, termcolor.Red, "FAIL")
	if _, err := io.WriteString(w, line+"\n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), termcolor.Red+"FAIL"+termcolor.Reset) {
		t.Fatalf("got %q", out)
	}
}
