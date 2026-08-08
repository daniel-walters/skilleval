package ratesync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	linkTextRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	priceRe    = regexp.MustCompile(`^\$?([0-9]+(?:\.[0-9]+)?)$`)
)

// ParsePricingMarkdown extracts Model Pricing table rows from Cursor docs markdown.
func ParsePricingMarkdown(md string) ([]DocRow, error) {
	lines := strings.Split(md, "\n")
	start := -1
	for i, line := range lines {
		if isPricingHeader(line) {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("ratesync: Model Pricing table header not found")
	}

	// Advance to the separator row after the header.
	i := start + 1
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || !isSeparatorRow(lines[i]) {
		return nil, fmt.Errorf("ratesync: Model Pricing table separator not found")
	}
	i++

	var rows []DocRow
	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			break
		}
		row, err := parseTableRow(line)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("ratesync: Model Pricing table has no data rows")
	}
	return rows, nil
}

func isPricingHeader(line string) bool {
	cells := splitCells(line)
	if len(cells) < 6 {
		return false
	}
	return strings.EqualFold(cells[0], "Model") &&
		strings.EqualFold(cells[1], "Provider") &&
		strings.EqualFold(cells[2], "Input") &&
		strings.Contains(strings.ToLower(cells[3]), "cache write") &&
		strings.Contains(strings.ToLower(cells[4]), "cache read") &&
		strings.EqualFold(cells[5], "Output")
}

func isSeparatorRow(line string) bool {
	cells := splitCells(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		t := strings.TrimSpace(c)
		if t == "" {
			continue
		}
		for _, r := range t {
			if r != '-' && r != ':' && r != ' ' {
				return false
			}
		}
	}
	return true
}

func parseTableRow(line string) (DocRow, error) {
	cells := splitCells(line)
	if len(cells) < 6 {
		return DocRow{}, fmt.Errorf("ratesync: expected at least 6 columns, got %d in %q", len(cells), line)
	}
	name := normalizeDisplayName(cells[0])
	if name == "" {
		return DocRow{}, fmt.Errorf("ratesync: empty model display name in %q", line)
	}
	input, err := parsePrice(cells[2])
	if err != nil {
		return DocRow{}, fmt.Errorf("ratesync: input for %q: %w", name, err)
	}
	cacheWrite, err := parsePriceOrDash(cells[3])
	if err != nil {
		return DocRow{}, fmt.Errorf("ratesync: cache write for %q: %w", name, err)
	}
	cacheRead, err := parsePrice(cells[4])
	if err != nil {
		return DocRow{}, fmt.Errorf("ratesync: cache read for %q: %w", name, err)
	}
	output, err := parsePrice(cells[5])
	if err != nil {
		return DocRow{}, fmt.Errorf("ratesync: output for %q: %w", name, err)
	}
	return DocRow{
		DisplayName: name,
		Rates: Rates{
			Input:      input,
			Output:     output,
			CacheRead:  cacheRead,
			CacheWrite: cacheWrite,
		},
	}, nil
}

func splitCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func normalizeDisplayName(cell string) string {
	s := strings.TrimSpace(cell)
	if m := linkTextRe.FindStringSubmatch(s); len(m) == 2 {
		s = m[1]
	}
	return strings.TrimSpace(s)
}

func parsePrice(cell string) (float64, error) {
	s := strings.TrimSpace(cell)
	m := priceRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid price %q", cell)
	}
	return strconv.ParseFloat(m[1], 64)
}

func parsePriceOrDash(cell string) (float64, error) {
	s := strings.TrimSpace(cell)
	if s == "-" || s == "—" || s == "" {
		return 0, nil
	}
	return parsePrice(s)
}
