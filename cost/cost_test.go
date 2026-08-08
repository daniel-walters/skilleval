package cost_test

import (
	"math"
	"testing"

	"github.com/daniel-walters/skilleval/cost"
	"github.com/daniel-walters/skilleval/result"
)

func TestUSDComposerEstimate(t *testing.T) {
	// composer-2.5: in 0.5, out 2.5, cacheRead 0.2, cacheWrite 0
	got := cost.USD("cursor", "composer-2.5", result.Usage{
		InputTokens:      100_000,
		OutputTokens:     5_000,
		CacheReadTokens:  500_000,
		CacheWriteTokens: 10_000,
	})
	if got == nil {
		t.Fatal("USD = nil, want estimate")
	}
	// (100000*0.5 + 5000*2.5 + 500000*0.2 + 10000*0) / 1e6
	// = (50000 + 12500 + 100000) / 1e6 = 0.1625
	want := 0.1625
	if math.Abs(*got-want) > 1e-12 {
		t.Fatalf("USD = %g, want %g", *got, want)
	}
}

func TestUSDComposer2AliasMatches25(t *testing.T) {
	usage := result.Usage{
		InputTokens:      100_000,
		OutputTokens:     5_000,
		CacheReadTokens:  500_000,
		CacheWriteTokens: 10_000,
	}
	a := cost.USD("cursor", "composer-2", usage)
	b := cost.USD("cursor", "composer-2.5", usage)
	if a == nil || b == nil {
		t.Fatalf("USD = %v / %v, want estimates", a, b)
	}
	if *a != *b {
		t.Fatalf("composer-2 = %g, composer-2.5 = %g, want equal", *a, *b)
	}
}

func TestUSDUnknownModelNil(t *testing.T) {
	got := cost.USD("cursor", "not-a-real-model", result.Usage{
		InputTokens:  1000,
		OutputTokens: 100,
		TotalTokens:  1100,
	})
	if got != nil {
		t.Fatalf("USD = %v, want nil", *got)
	}
}

func TestUSDEmptyModelNil(t *testing.T) {
	got := cost.USD("cursor", "  ", result.Usage{InputTokens: 1})
	if got != nil {
		t.Fatalf("USD = %v, want nil", *got)
	}
}

func TestUSDEmptyProviderNil(t *testing.T) {
	got := cost.USD("  ", "composer-2.5", result.Usage{InputTokens: 1})
	if got != nil {
		t.Fatalf("USD = %v, want nil", *got)
	}
}

func TestUSDUnknownProviderNil(t *testing.T) {
	got := cost.USD("nope", "composer-2.5", result.Usage{InputTokens: 1})
	if got != nil {
		t.Fatalf("USD = %v, want nil", *got)
	}
}

func TestUSDNoCrossProviderLeak(t *testing.T) {
	// Cursor-catalog model must not resolve under anthropic.
	got := cost.USD("anthropic", "claude-4.5-sonnet", result.Usage{
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	})
	if got != nil {
		t.Fatalf("USD = %v, want nil (no cross-provider leak)", *got)
	}
}

func TestUSDZeroTokensIsZero(t *testing.T) {
	got := cost.USD("cursor", "composer-2.5", result.Usage{})
	if got == nil {
		t.Fatal("USD = nil, want 0")
	}
	if *got != 0 {
		t.Fatalf("USD = %g, want 0", *got)
	}
}

func TestUSDClaudeWithCacheWrite(t *testing.T) {
	// claude-4.5-sonnet under cursor: 3 / 15 / 0.3 / 3.75
	got := cost.USD("cursor", "claude-4.5-sonnet", result.Usage{
		InputTokens:      1_000_000,
		OutputTokens:     1_000_000,
		CacheReadTokens:  1_000_000,
		CacheWriteTokens: 1_000_000,
	})
	if got == nil {
		t.Fatal("USD = nil, want estimate")
	}
	want := 3 + 15 + 0.3 + 3.75 // 22.05
	if math.Abs(*got-want) > 1e-12 {
		t.Fatalf("USD = %g, want %g", *got, want)
	}
}

func TestProviderForRunner(t *testing.T) {
	cases := []struct {
		runner string
		want   string
	}{
		{"cursor", "cursor"},
		{"Claude", "anthropic"},
		{"claude", "anthropic"},
		{"", ""},
		{"other", ""},
	}
	for _, tc := range cases {
		if got := cost.ProviderForRunner(tc.runner); got != tc.want {
			t.Fatalf("ProviderForRunner(%q) = %q, want %q", tc.runner, got, tc.want)
		}
	}
}
