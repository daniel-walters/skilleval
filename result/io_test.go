package result_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daniel-walters/skilleval/result"
)

func TestLoadGolden(t *testing.T) {
	got, err := result.Load(filepath.Join("testdata", "result.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertGoldenResult(t, *got)
}

func TestWriteRoundTrip(t *testing.T) {
	got, err := result.Load(filepath.Join("testdata", "result.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	out := filepath.Join(t.TempDir(), "result.json")
	if err := result.Write(out, got); err != nil {
		t.Fatalf("Write: %v", err)
	}
	again, err := result.Load(out)
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	assertGoldenResult(t, *again)
}

func TestLoadRejectsBadSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":99,"id":"x"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := result.Load(path); err == nil {
		t.Fatal("expected error for unsupported schemaVersion")
	}
}

func TestWriteRejectsNil(t *testing.T) {
	if err := result.Write(filepath.Join(t.TempDir(), "x.json"), nil); err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestWriteRejectsBadSchemaVersion(t *testing.T) {
	r := &result.Result{
		SchemaVersion: 99,
		ID:            "x",
		StartedAt:     time.Now().UTC(),
		FinishedAt:    time.Now().UTC(),
	}
	if err := result.Write(filepath.Join(t.TempDir(), "x.json"), r); err == nil {
		t.Fatal("expected error for unsupported schemaVersion")
	}
}
