package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestHeartbeatNonTTY(t *testing.T) {
	var buf bytes.Buffer
	h := startHeartbeat(&buf, false)
	h.Stop()
	h.Stop() // idempotent

	got := buf.String()
	if got != "agent running…\n" {
		t.Fatalf("got %q, want %q", got, "agent running…\n")
	}
	if strings.Contains(got, "\r") {
		t.Fatalf("non-TTY output should not use \\r: %q", got)
	}
}

func TestHeartbeatTTY(t *testing.T) {
	var buf bytes.Buffer
	h := startHeartbeat(&buf, true)

	deadline := time.After(2 * time.Second)
	for !strings.Contains(buf.String(), "agent running… 0s") {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for first render; got %q", buf.String())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	h.Stop()
	h.Stop() // idempotent

	got := buf.String()
	if !strings.Contains(got, "\r") {
		t.Fatalf("TTY output should use \\r: %q", got)
	}
	if !strings.HasSuffix(got, "\r\033[K") {
		t.Fatalf("Stop should clear the line; got %q", got)
	}
}
