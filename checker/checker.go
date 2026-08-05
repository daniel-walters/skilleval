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
	if e == nil || e.Max == nil {
		return nil
	}
	if r.Metrics.Turns > *e.Max {
		return []Failure{{
			Path:   "turns.max",
			Reason: fmt.Sprintf("turns %d exceeds max %d", r.Metrics.Turns, *e.Max),
		}}
	}
	return nil
}

func checkCost(r *result.Result, e *eval.CostExpect) []Failure {
	if e == nil || e.Max == nil {
		return nil
	}
	if r.Metrics.CostUSD == nil {
		return []Failure{{
			Path:   "costUSD.max",
			Reason: "costUSD is unknown (nil), cannot satisfy max bound",
		}}
	}
	if *r.Metrics.CostUSD > *e.Max {
		return []Failure{{
			Path:   "costUSD.max",
			Reason: fmt.Sprintf("costUSD %g exceeds max %g", *r.Metrics.CostUSD, *e.Max),
		}}
	}
	return nil
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
		if fe.Status == result.FileDeleted || (ok && outcome.Status == result.FileDeleted) {
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
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("path %q must be relative to workspace", rel)
	}
	full := filepath.Join(workspace, filepath.FromSlash(rel))
	raw, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", rel, err)
	}
	return string(raw), nil
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}
