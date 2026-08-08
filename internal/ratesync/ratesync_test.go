package ratesync_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daniel-walters/skilleval/internal/ratesync"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParsePricingMarkdown(t *testing.T) {
	rows, err := ratesync.ParsePricingMarkdown(readTestdata(t, "pricing_sample.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	if rows[0].DisplayName != "Auto Cost" {
		t.Fatalf("row0 name = %q", rows[0].DisplayName)
	}
	if rows[1].DisplayName != "Composer 2.5" {
		t.Fatalf("row1 name = %q (link text)", rows[1].DisplayName)
	}
	if rows[1].Rates.CacheWrite != 0 {
		t.Fatalf("Composer cacheWrite = %v, want 0 for dash", rows[1].Rates.CacheWrite)
	}
	if rows[2].Rates.Input != 3 || rows[2].Rates.Output != 15 {
		t.Fatalf("Claude rates = %+v", rows[2].Rates)
	}
}

func TestParsePricingMarkdownMissingHeader(t *testing.T) {
	_, err := ratesync.ParsePricingMarkdown("# No table here\n")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMapRowsUnmapped(t *testing.T) {
	rows, err := ratesync.ParsePricingMarkdown(readTestdata(t, "pricing_sample.md"))
	if err != nil {
		t.Fatal(err)
	}
	aliases, err := ratesync.LoadAliases(filepath.Join("testdata", "aliases_missing_fable.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ratesync.MapRows(rows, aliases)
	if err == nil {
		t.Fatal("expected unmapped error")
	}
	if !strings.Contains(err.Error(), "Claude Fable 5") {
		t.Fatalf("error = %v", err)
	}
}

func TestSyncMergeUpdateAddPreserveStale(t *testing.T) {
	dir := t.TempDir()
	ratesPath := filepath.Join(dir, "rates.json")
	if err := os.WriteFile(ratesPath, []byte(readTestdata(t, "rates_base.json")), 0o644); err != nil {
		t.Fatal(err)
	}
	aliasesPath := filepath.Join("testdata", "aliases_ok.json")

	res, err := ratesync.Sync(context.Background(), ratesync.Options{
		RatesPath:   ratesPath,
		AliasesPath: aliasesPath,
		Markdown:    readTestdata(t, "pricing_sample.md"),
		AsOf:        "2026-08-08",
		Write:       true,
		SourceURL:   ratesync.DefaultSourceURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Fatal("expected changes")
	}
	if res.AsOf != "2026-08-08" {
		t.Fatalf("asOf = %q", res.AsOf)
	}
	if len(res.Updated) != 1 || res.Updated[0] != "grok-4.5" {
		t.Fatalf("updated = %v", res.Updated)
	}
	if len(res.Added) != 1 || res.Added[0] != "claude-fable-5" {
		t.Fatalf("added = %v", res.Added)
	}
	if len(res.Stale) != 1 || res.Stale[0] != "legacy-stale" {
		t.Fatalf("stale = %v", res.Stale)
	}

	raw, err := os.ReadFile(ratesPath)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		AsOf      string `json:"asOf"`
		Providers map[string]struct {
			Source string                    `json:"source"`
			Models map[string]ratesync.Rates `json:"models"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if file.AsOf != "2026-08-08" {
		t.Fatalf("written asOf = %q", file.AsOf)
	}
	anth := file.Providers["anthropic"]
	if anth.Models == nil {
		t.Fatal("anthropic models missing")
	}
	if len(anth.Models) != 0 {
		t.Fatalf("anthropic models = %v, want empty", anth.Models)
	}
	if _, ok := file.Providers["cursor"].Models["legacy-stale"]; !ok {
		t.Fatal("stale model was deleted")
	}
	fable := file.Providers["cursor"].Models["claude-fable-5"]
	if fable.Input != 10 || fable.Output != 50 {
		t.Fatalf("fable rates = %+v", fable)
	}
	grok := file.Providers["cursor"].Models["grok-4.5"]
	if grok.Input != 2 || grok.Output != 6 {
		t.Fatalf("grok rates = %+v", grok)
	}
}

func TestSyncUnmappedDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	ratesPath := filepath.Join(dir, "rates.json")
	before := []byte(readTestdata(t, "rates_base.json"))
	if err := os.WriteFile(ratesPath, before, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ratesync.Sync(context.Background(), ratesync.Options{
		RatesPath:   ratesPath,
		AliasesPath: filepath.Join("testdata", "aliases_missing_fable.json"),
		Markdown:    readTestdata(t, "pricing_sample.md"),
		Write:       true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	after, err := os.ReadFile(ratesPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("rates.json was written despite unmapped names")
	}
}

func TestSyncNoWriteWhenClean(t *testing.T) {
	dir := t.TempDir()
	ratesPath := filepath.Join(dir, "rates.json")
	// Build a rates file that already matches the sample docs (no stale).
	initial := map[string]any{
		"asOf": "2026-08-08",
		"providers": map[string]any{
			"cursor": map[string]any{
				"source": "https://cursor.com/docs/account/teams/pricing",
				"models": map[string]any{
					"auto":              map[string]float64{"input": 1.25, "output": 6, "cacheRead": 0.25, "cacheWrite": 1.25},
					"auto-smart":        map[string]float64{"input": 1.25, "output": 6, "cacheRead": 0.25, "cacheWrite": 1.25},
					"composer-2.5":      map[string]float64{"input": 0.5, "output": 2.5, "cacheRead": 0.2, "cacheWrite": 0},
					"claude-4.5-sonnet": map[string]float64{"input": 3, "output": 15, "cacheRead": 0.3, "cacheWrite": 3.75},
					"claude-fable-5":    map[string]float64{"input": 10, "output": 50, "cacheRead": 1, "cacheWrite": 12.5},
					"grok-4.5":          map[string]float64{"input": 2, "output": 6, "cacheRead": 0.5, "cacheWrite": 0},
				},
			},
			"anthropic": map[string]any{
				"source": "https://docs.anthropic.com/en/docs/about-claude/pricing",
				"models": map[string]any{},
			},
		},
	}
	b, err := json.MarshalIndent(initial, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(ratesPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ratesync.Sync(context.Background(), ratesync.Options{
		RatesPath:   ratesPath,
		AliasesPath: filepath.Join("testdata", "aliases_ok.json"),
		Markdown:    readTestdata(t, "pricing_sample.md"),
		AsOf:        "2099-01-01",
		Write:       true,
		SourceURL:   ratesync.DefaultSourceURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed || !res.Unchanged {
		t.Fatalf("expected unchanged, got %+v", res)
	}
	after, err := os.ReadFile(ratesPath)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		AsOf string `json:"asOf"`
	}
	if err := json.Unmarshal(after, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.AsOf != "2026-08-08" {
		t.Fatalf("asOf bumped without changes: %q", parsed.AsOf)
	}
}

func TestFormatSummary(t *testing.T) {
	s := ratesync.FormatSummary(ratesync.Result{
		Changed: true,
		AsOf:    "2026-08-08",
		Updated: []string{"grok-4.5"},
		Added:   []string{"claude-fable-5"},
		Stale:   []string{"legacy-stale"},
	})
	for _, want := range []string{"Updated", "Added", "Stale", "Anthropic", "2026-08-08"} {
		if !strings.Contains(s, want) {
			t.Fatalf("summary missing %q:\n%s", want, s)
		}
	}
}
