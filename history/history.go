// Package history retains summary Reports under a durable on-disk layout.
package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daniel-walters/skilleval/summary"
)

// Retain writes r into dir under evalName as a timestamped JSON file and
// updates latest.json to the same Report. It returns the timestamped path.
//
// Layout: dir/<evalName>/<UTC-timestamp>.json and dir/<evalName>/latest.json
// evalName must be a single relative path segment (no separators, no ..).
func Retain(dir, evalName string, r summary.Report) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("history: retain: dir is empty")
	}
	evalDir, err := evalDirUnder(dir, evalName)
	if err != nil {
		return "", err
	}
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

// evalDirUnder joins dir and evalName after rejecting names that escape dir.
func evalDirUnder(dir, evalName string) (string, error) {
	if evalName == "" {
		return "", fmt.Errorf("history: retain: eval name is empty")
	}
	if filepath.IsAbs(evalName) || filepath.IsAbs(filepath.FromSlash(evalName)) {
		return "", fmt.Errorf("history: retain: eval name %q must be a single path segment under the history dir", evalName)
	}
	if strings.Contains(evalName, "/") || strings.Contains(evalName, `\`) {
		return "", fmt.Errorf("history: retain: eval name %q must be a single path segment under the history dir", evalName)
	}
	cleaned := filepath.Clean(evalName)
	if cleaned == "." || cleaned == ".." {
		return "", fmt.Errorf("history: retain: eval name %q must be a single path segment under the history dir", evalName)
	}

	root := filepath.Clean(dir)
	evalDir := filepath.Join(root, cleaned)
	rel, err := filepath.Rel(root, evalDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("history: retain: eval name %q escapes history dir", evalName)
	}
	return evalDir, nil
}
