package ratesync

// ratesFile is the on-disk shape of cost/rates.json.
type ratesFile struct {
	AsOf      string                   `json:"asOf"`
	Providers map[string]providerRates `json:"providers"`
}

// providerRates is one billing provider's catalog.
type providerRates struct {
	Source string           `json:"source"`
	Models map[string]Rates `json:"models"`
}

// Rates are USD per million tokens for one model.
type Rates struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// DocRow is one parsed Model Pricing table row.
type DocRow struct {
	DisplayName string
	Rates       Rates
}

// Result summarizes a successful Sync merge.
type Result struct {
	Changed   bool
	AsOf      string
	Updated   []string
	Added     []string
	Stale     []string
	Unchanged bool
}
