package ratesync

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// LoadAliases reads display-name → model-id mappings from a JSON object.
func LoadAliases(path string) (map[string][]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ratesync: read aliases: %w", err)
	}
	var raw map[string][]string
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("ratesync: decode aliases: %w", err)
	}
	out := make(map[string][]string, len(raw))
	for name, ids := range raw {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("ratesync: aliases contain empty display name")
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("ratesync: alias %q has no model ids", name)
		}
		cleaned := make([]string, 0, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				return nil, fmt.Errorf("ratesync: alias %q has empty model id", name)
			}
			cleaned = append(cleaned, id)
		}
		out[name] = cleaned
	}
	return out, nil
}

// MapRows resolves doc rows to catalog model ids via aliases.
// Returns an error listing every unmapped display name (does not invent ids).
func MapRows(rows []DocRow, aliases map[string][]string) (map[string]Rates, error) {
	out := make(map[string]Rates)
	var unmapped []string
	seen := make(map[string]bool)
	for _, row := range rows {
		ids, ok := aliases[row.DisplayName]
		if !ok {
			if !seen[row.DisplayName] {
				unmapped = append(unmapped, row.DisplayName)
				seen[row.DisplayName] = true
			}
			continue
		}
		for _, id := range ids {
			out[id] = row.Rates
		}
	}
	if len(unmapped) > 0 {
		sort.Strings(unmapped)
		return nil, fmt.Errorf("ratesync: unmapped display names (add to cost/cursor_aliases.json): %s", strings.Join(unmapped, ", "))
	}
	return out, nil
}
