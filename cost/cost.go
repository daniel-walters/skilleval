// Package cost estimates run cost in USD from token usage and model rates.
package cost

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/daniel-walters/skilleval/result"
)

//go:embed rates.json
var ratesJSON []byte

// ratesFile is the on-disk shape of rates.json.
type ratesFile struct {
	AsOf   string           `json:"asOf"`
	Source string           `json:"source"`
	Models map[string]rates `json:"models"`
}

// rates are USD per million tokens for one model.
type rates struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

var (
	loadOnce sync.Once
	catalog  map[string]rates
	loadErr  error
)

func load() {
	loadOnce.Do(func() {
		var f ratesFile
		if err := json.Unmarshal(ratesJSON, &f); err != nil {
			loadErr = fmt.Errorf("cost: decode rates.json: %w", err)
			return
		}
		if len(f.Models) == 0 {
			loadErr = fmt.Errorf("cost: rates.json has no models")
			return
		}
		catalog = f.Models
	})
}

// USD returns estimated run cost in USD for model and usage, or nil when the
// model is unknown / empty. A known model with zero tokens yields 0, not nil.
func USD(model string, usage result.Usage) *float64 {
	load()
	if loadErr != nil {
		return nil
	}
	id := strings.TrimSpace(model)
	if id == "" {
		return nil
	}
	r, ok := catalog[id]
	if !ok {
		return nil
	}
	total := (float64(usage.InputTokens)*r.Input +
		float64(usage.OutputTokens)*r.Output +
		float64(usage.CacheReadTokens)*r.CacheRead +
		float64(usage.CacheWriteTokens)*r.CacheWrite) / 1_000_000
	return &total
}
