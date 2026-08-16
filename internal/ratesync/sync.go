package ratesync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DefaultSourceURL is the Cursor teams pricing markdown page.
const DefaultSourceURL = "https://cursor.com/docs/account/teams/pricing.md"

// Options configures Sync.
type Options struct {
	RatesPath   string
	AliasesPath string
	SourceURL   string
	AsOf        string // optional; defaults to UTC today when catalog changes
	Write       bool
	HTTPClient  *http.Client
	// Markdown, when non-empty, skips the HTTP fetch (tests / offline).
	Markdown string
}

// Sync fetches (or uses) pricing markdown, maps aliases, merges into rates.json,
// and optionally writes the result. Parse/unmapped failures return an error
// without writing.
func Sync(ctx context.Context, opts Options) (Result, error) {
	if opts.RatesPath == "" {
		return Result{}, fmt.Errorf("ratesync: RatesPath is required")
	}
	if opts.AliasesPath == "" {
		return Result{}, fmt.Errorf("ratesync: AliasesPath is required")
	}
	url := opts.SourceURL
	if url == "" {
		url = DefaultSourceURL
	}

	md := opts.Markdown
	if md == "" {
		var err error
		md, err = fetchMarkdown(ctx, opts.HTTPClient, url)
		if err != nil {
			return Result{}, err
		}
	}

	rows, err := ParsePricingMarkdown(md)
	if err != nil {
		return Result{}, err
	}

	aliases, err := LoadAliases(opts.AliasesPath)
	if err != nil {
		return Result{}, err
	}

	docsRates, err := MapRows(rows, aliases)
	if err != nil {
		return Result{}, err
	}

	file, err := loadRatesFile(opts.RatesPath)
	if err != nil {
		return Result{}, err
	}

	// Human-facing source URL without .md suffix when using the default markdown endpoint.
	source := stripMarkdownSuffix(url)
	res, err := MergeCursor(file, docsRates, opts.AsOf, source)
	if err != nil {
		return Result{}, err
	}

	if opts.Write && res.Changed {
		if err := writeRatesFile(opts.RatesPath, file); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

const fetchAttempts = 3

// fetchRetryDelay is the pause after a retryable failure. Tests set this to 0.
var fetchRetryDelay = func(attempt int) time.Duration {
	return time.Duration(attempt) * 400 * time.Millisecond
}

type retryableError struct{ error }

func (e retryableError) Unwrap() error { return e.error }

func retryableStatus(code int) bool {
	switch code {
	case http.StatusNotFound, http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func fetchMarkdown(ctx context.Context, client *http.Client, url string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	var lastErr error
	for attempt := 1; attempt <= fetchAttempts; attempt++ {
		body, err := fetchMarkdownOnce(ctx, client, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if attempt == fetchAttempts || !errors.As(err, new(retryableError)) {
			break
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("ratesync: fetch %s: %w", url, ctx.Err())
		case <-time.After(fetchRetryDelay(attempt)):
		}
	}
	return "", lastErr
}

func fetchMarkdownOnce(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("ratesync: build request: %w", err)
	}
	req.Header.Set("Accept", "text/markdown, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; skilleval-ratesync/1.0; +https://github.com/daniel-walters/skilleval)")

	resp, err := client.Do(req)
	if err != nil {
		return "", retryableError{fmt.Errorf("ratesync: fetch %s: %w", url, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		err := fmt.Errorf("ratesync: fetch %s: HTTP %d: %s", url, resp.StatusCode, bytes.TrimSpace(body))
		if retryableStatus(resp.StatusCode) {
			return "", retryableError{err}
		}
		return "", err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ratesync: read body: %w", err)
	}
	return string(b), nil
}

func loadRatesFile(path string) (*ratesFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ratesync: read rates: %w", err)
	}
	var f ratesFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("ratesync: decode rates: %w", err)
	}
	return &f, nil
}

func writeRatesFile(path string, f *ratesFile) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(f); err != nil {
		return fmt.Errorf("ratesync: encode rates: %w", err)
	}
	// Encoder adds a trailing newline; keep stable formatting.
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("ratesync: write rates: %w", err)
	}
	return nil
}

func stripMarkdownSuffix(url string) string {
	if len(url) > 3 && url[len(url)-3:] == ".md" {
		return url[:len(url)-3]
	}
	return url
}

// FormatSummary returns a human-readable Sync result for CLI / PR bodies.
func FormatSummary(res Result) string {
	var b bytes.Buffer
	if res.Unchanged {
		fmt.Fprintf(&b, "No Cursor rate changes (asOf=%s).\n", res.AsOf)
	} else {
		fmt.Fprintf(&b, "Cursor rates updated (asOf=%s).\n", res.AsOf)
	}
	if len(res.Updated) > 0 {
		fmt.Fprintf(&b, "Updated (%d): %v\n", len(res.Updated), res.Updated)
	}
	if len(res.Added) > 0 {
		fmt.Fprintf(&b, "Added (%d): %v\n", len(res.Added), res.Added)
	}
	if len(res.Stale) > 0 {
		fmt.Fprintf(&b, "Stale candidates kept (%d): %v\n", len(res.Stale), res.Stale)
		fmt.Fprintf(&b, "(Removals require human confirmation; not deleted.)\n")
	}
	fmt.Fprintf(&b, "Anthropic catalog left untouched.\n")
	return b.String()
}
