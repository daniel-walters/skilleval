package ratesync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncRetriesTransientFetchError(t *testing.T) {
	origDelay := fetchRetryDelay
	fetchRetryDelay = func(int) time.Duration { return 0 }
	t.Cleanup(func() { fetchRetryDelay = origDelay })

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"File not found"}`))
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		b, err := os.ReadFile(filepath.Join("testdata", "pricing_sample.md"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	ratesPath := filepath.Join(dir, "rates.json")
	base, err := os.ReadFile(filepath.Join("testdata", "rates_base.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ratesPath, base, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Sync(context.Background(), Options{
		RatesPath:   ratesPath,
		AliasesPath: filepath.Join("testdata", "aliases_ok.json"),
		SourceURL:   srv.URL,
		AsOf:        "2026-08-16",
		Write:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits = %d, want 3", hits.Load())
	}
	if !res.Changed {
		t.Fatal("expected catalog changes after retried fetch")
	}
}
