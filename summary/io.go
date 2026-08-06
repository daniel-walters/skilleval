package summary

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads path as a Report JSON document.
func Load(path string) (*Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("summary: read %s: %w", path, err)
	}
	var r Report
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("summary: decode %s: %w", path, err)
	}
	return &r, nil
}

// Write encodes r as pretty JSON to path.
func Write(path string, r Report) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("summary: encode %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("summary: write %s: %w", path, err)
	}
	return nil
}
