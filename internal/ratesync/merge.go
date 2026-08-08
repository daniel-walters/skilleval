package ratesync

import (
	"fmt"
	"sort"
	"time"
)

const cursorProvider = "cursor"

// MergeCursor updates only the cursor provider models from docsRates.
// Known catalog models missing from docs are preserved (listed as Stale).
// Anthropic and other providers are copied through unchanged.
func MergeCursor(file *ratesFile, docsRates map[string]Rates, asOf string, source string) (Result, error) {
	if file == nil {
		return Result{}, fmt.Errorf("ratesync: nil rates file")
	}
	if file.Providers == nil {
		return Result{}, fmt.Errorf("ratesync: rates file has no providers")
	}
	cursor, ok := file.Providers[cursorProvider]
	if !ok {
		return Result{}, fmt.Errorf("ratesync: rates file missing %q provider", cursorProvider)
	}
	if cursor.Models == nil {
		cursor.Models = map[string]Rates{}
	}

	// Preserve anthropic (and any other providers) by only mutating cursor.
	updated := make([]string, 0)
	added := make([]string, 0)
	seen := make(map[string]bool, len(docsRates))

	for id, rates := range docsRates {
		seen[id] = true
		prev, exists := cursor.Models[id]
		if !exists {
			cursor.Models[id] = rates
			added = append(added, id)
			continue
		}
		if prev != rates {
			cursor.Models[id] = rates
			updated = append(updated, id)
		}
	}

	stale := make([]string, 0)
	for id := range cursor.Models {
		if !seen[id] {
			stale = append(stale, id)
		}
	}

	sort.Strings(updated)
	sort.Strings(added)
	sort.Strings(stale)

	changed := len(updated) > 0 || len(added) > 0
	if source != "" && cursor.Source != source {
		cursor.Source = source
		changed = true
	}
	file.Providers[cursorProvider] = cursor

	if asOf == "" {
		asOf = time.Now().UTC().Format("2006-01-02")
	}
	if changed {
		file.AsOf = asOf
	}

	return Result{
		Changed:   changed,
		AsOf:      file.AsOf,
		Updated:   updated,
		Added:     added,
		Stale:     stale,
		Unchanged: !changed,
	}, nil
}
