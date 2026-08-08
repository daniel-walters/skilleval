// Package checker evaluates a Result against eval expects and returns a Verdict.
package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
)

// Failure is one expectation that did not hold.
type Failure struct {
	// Path identifies the expect (e.g. "turns.max", "files[src/foo.go].contains").
	Path string
	// Reason is a short human-readable explanation.
	Reason string
}

// Verdict is the pass/fail outcome of Check.
type Verdict struct {
	Passed   bool
	Failures []Failure
}

// Check evaluates expects against a finished Result and attempt workspace.
// When result status is not finished, expects are not evaluated.
func Check(r *result.Result, expects eval.Expects, workspace string) Verdict {
	if r == nil {
		return fail("run", "result is nil")
	}
	if r.Status != result.StatusFinished {
		return fail("run.status", fmt.Sprintf("status is %q, want %q", r.Status, result.StatusFinished))
	}

	var failures []Failure
	failures = append(failures, checkTurns(r, expects.Turns)...)
	failures = append(failures, checkDurationMs(r, expects.DurationMs)...)
	failures = append(failures, checkToolCalls(r, expects.ToolCalls)...)
	failures = append(failures, checkUsage(r, expects.Usage)...)
	failures = append(failures, checkCost(r, expects.CostUSD)...)
	failures = append(failures, checkToolsUsed(r, expects.ToolsUsed)...)
	failures = append(failures, checkSkills(r, expects.Skills)...)
	failures = append(failures, checkFiles(r, expects.Files, workspace)...)
	failures = append(failures, checkFinalMessage(r, expects.FinalMessage)...)

	return Verdict{Passed: len(failures) == 0, Failures: failures}
}

func fail(path, reason string) Verdict {
	return Verdict{
		Passed:   false,
		Failures: []Failure{{Path: path, Reason: reason}},
	}
}

func checkTurns(r *result.Result, e *eval.TurnsExpect) []Failure {
	if e == nil {
		return nil
	}
	return checkIntBounds("turns", r.Metrics.Turns, intBounds{
		Min: e.Min,
		Max: e.Max,
		Gt:  e.Gt,
		Lt:  e.Lt,
		Eq:  e.Eq,
	})
}

func checkDurationMs(r *result.Result, e *eval.TurnsExpect) []Failure {
	if e == nil {
		return nil
	}
	return checkIntBounds("durationMs", int(r.Metrics.DurationMs), intBounds{
		Min: e.Min,
		Max: e.Max,
		Gt:  e.Gt,
		Lt:  e.Lt,
		Eq:  e.Eq,
	})
}

func checkToolCalls(r *result.Result, e *eval.TurnsExpect) []Failure {
	if e == nil {
		return nil
	}
	return checkIntBounds("toolCalls", len(r.Metrics.ToolCalls), intBounds{
		Min: e.Min,
		Max: e.Max,
		Gt:  e.Gt,
		Lt:  e.Lt,
		Eq:  e.Eq,
	})
}

func checkUsage(r *result.Result, e *eval.UsageExpect) []Failure {
	if e == nil {
		return nil
	}
	var failures []Failure
	failures = append(failures, checkUsageField("usage.inputTokens", r.Metrics.Usage.InputTokens, e.InputTokens)...)
	failures = append(failures, checkUsageField("usage.outputTokens", r.Metrics.Usage.OutputTokens, e.OutputTokens)...)
	failures = append(failures, checkUsageField("usage.cacheReadTokens", r.Metrics.Usage.CacheReadTokens, e.CacheReadTokens)...)
	failures = append(failures, checkUsageField("usage.cacheWriteTokens", r.Metrics.Usage.CacheWriteTokens, e.CacheWriteTokens)...)
	failures = append(failures, checkUsageField("usage.totalTokens", r.Metrics.Usage.TotalTokens, e.TotalTokens)...)
	return failures
}

func checkUsageField(prefix string, actual int, e *eval.TurnsExpect) []Failure {
	if e == nil {
		return nil
	}
	return checkIntBounds(prefix, actual, intBounds{
		Min: e.Min,
		Max: e.Max,
		Gt:  e.Gt,
		Lt:  e.Lt,
		Eq:  e.Eq,
	})
}

func checkCost(r *result.Result, e *eval.CostExpect) []Failure {
	if e == nil {
		return nil
	}
	return checkFloatBounds("costUSD", r.Metrics.CostUSD, floatBounds{
		Min: e.Min,
		Max: e.Max,
		Gt:  e.Gt,
		Lt:  e.Lt,
		Eq:  e.Eq,
	})
}

type intBounds struct {
	Min, Max, Gt, Lt, Eq *int
}

type floatBounds struct {
	Min, Max, Gt, Lt, Eq *float64
}

func checkIntBounds(prefix string, actual int, b intBounds) []Failure {
	var failures []Failure
	if b.Min != nil && actual < *b.Min {
		failures = append(failures, Failure{
			Path:   prefix + ".min",
			Reason: fmt.Sprintf("%s %d below min %d", prefix, actual, *b.Min),
		})
	}
	if b.Max != nil && actual > *b.Max {
		failures = append(failures, Failure{
			Path:   prefix + ".max",
			Reason: fmt.Sprintf("%s %d exceeds max %d", prefix, actual, *b.Max),
		})
	}
	if b.Gt != nil && actual <= *b.Gt {
		failures = append(failures, Failure{
			Path:   prefix + ".gt",
			Reason: fmt.Sprintf("%s %d not greater than %d", prefix, actual, *b.Gt),
		})
	}
	if b.Lt != nil && actual >= *b.Lt {
		failures = append(failures, Failure{
			Path:   prefix + ".lt",
			Reason: fmt.Sprintf("%s %d not less than %d", prefix, actual, *b.Lt),
		})
	}
	if b.Eq != nil && actual != *b.Eq {
		failures = append(failures, Failure{
			Path:   prefix + ".eq",
			Reason: fmt.Sprintf("%s %d not equal to %d", prefix, actual, *b.Eq),
		})
	}
	return failures
}

func checkFloatBounds(prefix string, actual *float64, b floatBounds) []Failure {
	ops := []struct {
		name string
		set  bool
	}{
		{"min", b.Min != nil},
		{"max", b.Max != nil},
		{"gt", b.Gt != nil},
		{"lt", b.Lt != nil},
		{"eq", b.Eq != nil},
	}
	anySet := false
	for _, op := range ops {
		if op.set {
			anySet = true
			break
		}
	}
	if !anySet {
		return nil
	}
	if actual == nil {
		var failures []Failure
		for _, op := range ops {
			if !op.set {
				continue
			}
			failures = append(failures, Failure{
				Path:   prefix + "." + op.name,
				Reason: fmt.Sprintf("%s is unknown (nil), cannot satisfy %s bound", prefix, op.name),
			})
		}
		return failures
	}
	v := *actual
	var failures []Failure
	if b.Min != nil && v < *b.Min {
		failures = append(failures, Failure{
			Path:   prefix + ".min",
			Reason: fmt.Sprintf("%s %g below min %g", prefix, v, *b.Min),
		})
	}
	if b.Max != nil && v > *b.Max {
		failures = append(failures, Failure{
			Path:   prefix + ".max",
			Reason: fmt.Sprintf("%s %g exceeds max %g", prefix, v, *b.Max),
		})
	}
	if b.Gt != nil && v <= *b.Gt {
		failures = append(failures, Failure{
			Path:   prefix + ".gt",
			Reason: fmt.Sprintf("%s %g not greater than %g", prefix, v, *b.Gt),
		})
	}
	if b.Lt != nil && v >= *b.Lt {
		failures = append(failures, Failure{
			Path:   prefix + ".lt",
			Reason: fmt.Sprintf("%s %g not less than %g", prefix, v, *b.Lt),
		})
	}
	if b.Eq != nil && v != *b.Eq {
		failures = append(failures, Failure{
			Path:   prefix + ".eq",
			Reason: fmt.Sprintf("%s %g not equal to %g", prefix, v, *b.Eq),
		})
	}
	return failures
}

func checkToolsUsed(r *result.Result, e *eval.ToolsUsedExpect) []Failure {
	if e == nil {
		return nil
	}
	used := stringSet(r.Metrics.ToolsUsed)
	var failures []Failure
	for _, tool := range e.Includes {
		if !used[tool] {
			failures = append(failures, Failure{
				Path:   "toolsUsed.includes",
				Reason: fmt.Sprintf("missing required tool %q", tool),
			})
		}
	}
	for _, tool := range e.Excludes {
		if used[tool] {
			failures = append(failures, Failure{
				Path:   "toolsUsed.excludes",
				Reason: fmt.Sprintf("forbidden tool %q was used", tool),
			})
		}
	}
	return failures
}

func checkSkills(r *result.Result, e *eval.SkillsExpect) []Failure {
	if e == nil || e.Activated == nil {
		return nil
	}
	activated := stringSet(r.Skills.Activated)
	var failures []Failure
	for _, skill := range e.Activated.Includes {
		if !activated[skill] {
			failures = append(failures, Failure{
				Path:   "skills.activated.includes",
				Reason: fmt.Sprintf("missing required activated skill %q", skill),
			})
		}
	}
	for _, skill := range e.Activated.Excludes {
		if activated[skill] {
			failures = append(failures, Failure{
				Path:   "skills.activated.excludes",
				Reason: fmt.Sprintf("forbidden activated skill %q was activated", skill),
			})
		}
	}
	return failures
}

func checkFiles(r *result.Result, files map[string]eval.FileExpect, workspace string) []Failure {
	if len(files) == 0 {
		return nil
	}
	outcomes := r.Outcomes.Files
	if outcomes == nil {
		outcomes = map[string]result.FileOutcome{}
	}

	var failures []Failure
	for path, fe := range files {
		prefix := fmt.Sprintf("files[%s]", path)
		outcome, ok := outcomes[path]

		if fe.Status != "" {
			if !ok {
				failures = append(failures, Failure{
					Path:   prefix + ".status",
					Reason: "path missing from outcomes.files",
				})
			} else if outcome.Status != fe.Status {
				failures = append(failures, Failure{
					Path:   prefix + ".status",
					Reason: fmt.Sprintf("status is %q, want %q", outcome.Status, fe.Status),
				})
			}
		}

		if !fe.Contains.IsSet() && !fe.Equals.IsSet() {
			continue
		}
		// Content expects require a recorded file change so pre-seeded input
		// alone cannot satisfy contains/equals.
		if !ok {
			failures = append(failures, Failure{
				Path:   prefix,
				Reason: "path missing from outcomes.files",
			})
			continue
		}
		if fe.Status == result.FileDeleted || outcome.Status == result.FileDeleted {
			failures = append(failures, Failure{
				Path:   prefix,
				Reason: "content expects cannot be checked for a deleted file",
			})
			continue
		}
		body, err := readWorkspaceFile(workspace, path)
		if err != nil {
			failures = append(failures, Failure{
				Path:   prefix,
				Reason: err.Error(),
			})
			continue
		}
		failures = append(failures, checkTextExpects(body, fe.Contains, fe.Equals, prefix, "file", "file content")...)
	}
	return failures
}

func checkFinalMessage(r *result.Result, e *eval.TextExpect) []Failure {
	if e == nil {
		return nil
	}
	return checkTextExpects(r.FinalMessage, e.Contains, e.Equals, "finalMessage", "finalMessage", "finalMessage")
}

func checkTextExpects(haystack string, contains, equals eval.StringMatch, pathPrefix, containsSubject, equalsSubject string) []Failure {
	var failures []Failure
	if contains.IsSet() && !contains.MatchContains(haystack) {
		reason := fmt.Sprintf("%s does not contain %q", containsSubject, contains.String())
		if contains.IsRegex() {
			reason = fmt.Sprintf("%s does not match %s", containsSubject, contains.String())
		}
		failures = append(failures, Failure{Path: pathPrefix + ".contains", Reason: reason})
	}
	if equals.IsSet() && !equals.MatchEquals(haystack) {
		reason := fmt.Sprintf("%s does not equal expected value", equalsSubject)
		if equals.IsRegex() {
			reason = fmt.Sprintf("%s does not match %s", equalsSubject, equals.String())
		}
		failures = append(failures, Failure{Path: pathPrefix + ".equals", Reason: reason})
	}
	return failures
}

func readWorkspaceFile(workspace, rel string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace is required for file content checks")
	}
	full, err := containedPath(workspace, rel)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", rel, err)
	}
	return string(raw), nil
}

// containedPath joins workspace and rel, rejecting absolute or escaping paths.
func containedPath(workspace, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("path %q must be relative to workspace", rel)
	}
	if filepath.IsAbs(rel) || filepath.IsAbs(filepath.FromSlash(rel)) {
		return "", fmt.Errorf("path %q must be relative to workspace", rel)
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace: %w", err)
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	full, err = filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("path %q: %w", rel, err)
	}
	relToRoot, err := filepath.Rel(root, full)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q must be relative to workspace", rel)
	}
	return full, nil
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}
