package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// heartbeat shows that an eval attempt is still in progress.
type heartbeat struct {
	w    io.Writer
	tty  bool
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

// startHeartbeat writes progress to w. When tty is true, it rewrites a single
// line with elapsed seconds until Stop. Otherwise it prints one static line.
func startHeartbeat(w io.Writer, tty bool) *heartbeat {
	h := &heartbeat{
		w:    w,
		tty:  tty,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	if !tty {
		_, _ = fmt.Fprintln(w, "agent running…")
		close(h.done)
		return h
	}
	go h.loop()
	return h
}

func (h *heartbeat) loop() {
	defer close(h.done)
	start := time.Now()
	h.render(0)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-h.stop:
			return
		case now := <-t.C:
			h.render(int(now.Sub(start).Seconds()))
		}
	}
}

func (h *heartbeat) render(secs int) {
	// \r return to start of line; \033[K clear to end of line.
	_, _ = fmt.Fprintf(h.w, "\ragent running… %ds\033[K", secs)
}

// Stop ends the heartbeat. Safe to call more than once.
func (h *heartbeat) Stop() {
	h.once.Do(func() {
		if !h.tty {
			return
		}
		close(h.stop)
		<-h.done
		_, _ = fmt.Fprint(h.w, "\r\033[K")
	})
}

func stderrIsTerminal() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}
