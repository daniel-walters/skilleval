// Package history retains summary Reports under a durable on-disk layout.
package history

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/daniel-walters/skilleval/summary"
)

// Retain writes r into dir under evalName as a timestamped JSON file and
// updates latest.json to the same Report. It returns the timestamped path.
//
// Layout: dir/<evalName>/<UTC-timestamp>.json and dir/<evalName>/latest.json
func Retain(dir, evalName string, r summary.Report) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("history: retain: dir is empty")
	}
	if evalName == "" {
		return "", fmt.Errorf("history: retain: eval name is empty")
	}

	evalDir := filepath.Join(dir, evalName)
	if err := os.MkdirAll(evalDir, 0o755); err != nil {
		return "", fmt.Errorf("history: mkdir %s: %w", evalDir, err)
	}

	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	tsPath := filepath.Join(evalDir, stamp+".json")
	if err := summary.Write(tsPath, r); err != nil {
		return "", err
	}
	latestPath := filepath.Join(evalDir, "latest.json")
	if err := summary.Write(latestPath, r); err != nil {
		return "", err
	}
	return tsPath, nil
}
