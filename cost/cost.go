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
	AsOf      string                   `json:"asOf"`
	Providers map[string]providerRates `json:"providers"`
}

// providerRates is one billing provider's catalog.
type providerRates struct {
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
	catalog  map[string]map[string]rates
	loadErr  error
)

func load() {
	loadOnce.Do(func() {
		var f ratesFile
		if err := json.Unmarshal(ratesJSON, &f); err != nil {
			loadErr = fmt.Errorf("cost: decode rates.json: %w", err)
			return
		}
		if len(f.Providers) == 0 {
			loadErr = fmt.Errorf("cost: rates.json has no providers")
			return
		}
		anyModels := false
		out := make(map[string]map[string]rates, len(f.Providers))
		for name, p := range f.Providers {
			models := p.Models
			if models == nil {
				models = map[string]rates{}
			}
			if len(models) > 0 {
				anyModels = true
			}
			out[name] = models
		}
		if !anyModels {
			loadErr = fmt.Errorf("cost: rates.json has no models")
			return
		}
		catalog = out
	})
}

// ProviderForRunner maps a runner id to a billing provider key used in rates.json.
// Unknown runners return "" (no rate estimate).
func ProviderForRunner(runnerID string) string {
	switch strings.ToLower(strings.TrimSpace(runnerID)) {
	case "cursor":
		return "cursor"
	case "claude":
		return "anthropic"
	default:
		return ""
	}
}

// USD returns estimated run cost in USD for provider+model and usage, or nil when
// the provider/model is unknown or empty. A known model with zero tokens yields 0, not nil.
func USD(provider, model string, usage result.Usage) *float64 {
	load()
	if loadErr != nil {
		return nil
	}
	prov := strings.TrimSpace(provider)
	id := strings.TrimSpace(model)
	if prov == "" || id == "" {
		return nil
	}
	models, ok := catalog[prov]
	if !ok {
		return nil
	}
	r, ok := models[id]
	if !ok {
		return nil
	}
	total := (float64(usage.InputTokens)*r.Input +
		float64(usage.OutputTokens)*r.Output +
		float64(usage.CacheReadTokens)*r.CacheRead +
		float64(usage.CacheWriteTokens)*r.CacheWrite) / 1_000_000
	return &total
}
