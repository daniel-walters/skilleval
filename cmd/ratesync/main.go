package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daniel-walters/skilleval/internal/ratesync"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("ratesync", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	ratesPath := fs.String("rates", "cost/rates.json", "path to rates.json")
	aliasesPath := fs.String("aliases", "cost/cursor_aliases.json", "path to cursor display-name aliases")
	sourceURL := fs.String("url", ratesync.DefaultSourceURL, "Cursor pricing markdown URL")
	asOf := fs.String("asOf", "", "asOf date (YYYY-MM-DD); default UTC today when changed")
	write := fs.Bool("write", false, "write rates.json when Cursor rates change")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Resolve relative paths from repo root when invoked from elsewhere.
	rates := *ratesPath
	aliases := *aliasesPath
	if !filepath.IsAbs(rates) {
		if abs, err := filepath.Abs(rates); err == nil {
			rates = abs
		}
	}
	if !filepath.IsAbs(aliases) {
		if abs, err := filepath.Abs(aliases); err == nil {
			aliases = abs
		}
	}

	res, err := ratesync.Sync(context.Background(), ratesync.Options{
		RatesPath:   rates,
		AliasesPath: aliases,
		SourceURL:   *sourceURL,
		AsOf:        *asOf,
		Write:       *write,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	fmt.Print(ratesync.FormatSummary(res))
	return 0
}
