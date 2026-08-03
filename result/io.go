package result

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads path as a Result JSON document, validates schemaVersion, and returns it.
func Load(path string) (*Result, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("result: read %s: %w", path, err)
	}
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("result: decode %s: %w", path, err)
	}
	if r.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("result: %s: unsupported schemaVersion %d (want %d)", path, r.SchemaVersion, SchemaVersion)
	}
	return &r, nil
}

// Write encodes r as pretty JSON to path.
func Write(path string, r *Result) error {
	if r == nil {
		return fmt.Errorf("result: write %s: result is nil", path)
	}
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("result: write %s: unsupported schemaVersion %d (want %d)", path, r.SchemaVersion, SchemaVersion)
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("result: encode %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("result: write %s: %w", path, err)
	}
	return nil
}
