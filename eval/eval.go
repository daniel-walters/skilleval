// Package eval defines the skilleval eval + expectations contract (v1):
// a lean YAML document for one eval (prompt, skill, fixtures, expects).
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniel-walters/skilleval/result"
	"gopkg.in/yaml.v3"
)

// SchemaVersion is the current eval YAML schema version.
const SchemaVersion = 1

// Eval is one eval document: what to run and what must be true.
type Eval struct {
	SchemaVersion int    `yaml:"schemaVersion"`
	Name          string `yaml:"name"`
	Prompt        string `yaml:"prompt"`
	// Replies are optional ordered follow-up user messages sent after the
	// initial prompt finishes each agent leg (interactive skill testing).
	Replies []string `yaml:"replies,omitempty"`
	// Skill is a filesystem path to a skill directory containing SKILL.md
	// (relative to the eval YAML, or absolute).
	Skill string `yaml:"skill"`
	Input string `yaml:"input,omitempty"`
	// MCP is an optional filesystem path to a native MCP JSON config file
	// (relative to the eval YAML, or absolute). The runner seeds it into the
	// attempt workspace as Cursor .cursor/mcp.json or Claude .mcp.json.
	MCP string `yaml:"mcp,omitempty"`
	// Attempts is how many times to run this eval (default 1 when omitted or <= 0).
	Attempts int `yaml:"attempts,omitempty"`
	// PassRate optionally gates a multi-attempt batch on aggregate pass rate.
	PassRate *PassRateExpect `yaml:"passRate,omitempty"`
	Expects  Expects         `yaml:"expects,omitempty"`
}

// PassRateExpect is a batch-level minimum pass rate across attempts.
type PassRateExpect struct {
	Min *float64 `yaml:"min,omitempty"`
}

// Expects are deterministic predicates over a Result (and workspace files).
type Expects struct {
	Turns        *TurnsExpect          `yaml:"turns,omitempty"`
	DurationMs   *TurnsExpect          `yaml:"durationMs,omitempty"`
	ToolCalls    *ToolCallsExpect      `yaml:"toolCalls,omitempty"`
	Usage        *UsageExpect          `yaml:"usage,omitempty"`
	CostUSD      *CostExpect           `yaml:"costUSD,omitempty"`
	ToolsUsed    *ToolsUsedExpect      `yaml:"toolsUsed,omitempty"`
	Skills       *SkillsExpect         `yaml:"skills,omitempty"`
	Files        map[string]FileExpect `yaml:"files,omitempty"`
	FinalMessage *TextExpect           `yaml:"finalMessage,omitempty"`
}

// TurnsExpect bounds run turn count.
// All fields are optional; each set bound is checked independently.
// Semantics: min ≥, max ≤, gt >, lt <, eq ==.
type TurnsExpect struct {
	Min *int `yaml:"min,omitempty"`
	Max *int `yaml:"max,omitempty"`
	Gt  *int `yaml:"gt,omitempty"`
	Lt  *int `yaml:"lt,omitempty"`
	Eq  *int `yaml:"eq,omitempty"`
}

// ToolCallsExpect bounds total tool call count and optionally checks
// ordered subsequences and per-name counts.
type ToolCallsExpect struct {
	Min   *int                    `yaml:"min,omitempty"`
	Max   *int                    `yaml:"max,omitempty"`
	Gt    *int                    `yaml:"gt,omitempty"`
	Lt    *int                    `yaml:"lt,omitempty"`
	Eq    *int                    `yaml:"eq,omitempty"`
	Order []ToolCallStepExpect    `yaml:"order,omitempty"`
	Named map[string]*TurnsExpect `yaml:"named,omitempty"`
}

// ToolCallStepExpect matches one call in an ordered subsequence.
type ToolCallStepExpect struct {
	Name string               `yaml:"name"`
	Args map[string]ArgExpect `yaml:"args,omitempty"`
}

// ArgExpect checks a single tool-call arg value (stringified if non-string).
type ArgExpect struct {
	Contains StringMatch `yaml:"contains,omitempty"`
	Equals   StringMatch `yaml:"equals,omitempty"`
}

// UsageExpect bounds token usage fields with the same int ops as turns.
type UsageExpect struct {
	InputTokens      *TurnsExpect `yaml:"inputTokens,omitempty"`
	OutputTokens     *TurnsExpect `yaml:"outputTokens,omitempty"`
	CacheReadTokens  *TurnsExpect `yaml:"cacheReadTokens,omitempty"`
	CacheWriteTokens *TurnsExpect `yaml:"cacheWriteTokens,omitempty"`
	TotalTokens      *TurnsExpect `yaml:"totalTokens,omitempty"`
}

// CostExpect bounds run cost in USD.
// All fields are optional; each set bound is checked independently.
// Semantics: min ≥, max ≤, gt >, lt <, eq ==.
type CostExpect struct {
	Min *float64 `yaml:"min,omitempty"`
	Max *float64 `yaml:"max,omitempty"`
	Gt  *float64 `yaml:"gt,omitempty"`
	Lt  *float64 `yaml:"lt,omitempty"`
	Eq  *float64 `yaml:"eq,omitempty"`
}

// ToolsUsedExpect checks toolsUsed membership.
type ToolsUsedExpect struct {
	Includes []string `yaml:"includes,omitempty"`
	Excludes []string `yaml:"excludes,omitempty"`
}

// SkillsExpect checks skill activation signals.
type SkillsExpect struct {
	Activated *StringSetExpect `yaml:"activated,omitempty"`
}

// StringSetExpect checks membership of listed values.
type StringSetExpect struct {
	Includes []string `yaml:"includes,omitempty"`
	Excludes []string `yaml:"excludes,omitempty"`
}

// FileExpect checks a path's outcome status and/or content.
type FileExpect struct {
	Status   result.FileStatus `yaml:"status,omitempty"`
	Contains StringMatch       `yaml:"contains,omitempty"`
	Equals   StringMatch       `yaml:"equals,omitempty"`
}

// TextExpect checks finalMessage (or similar text fields).
type TextExpect struct {
	Contains StringMatch `yaml:"contains,omitempty"`
	Equals   StringMatch `yaml:"equals,omitempty"`
}

// Load reads path as an eval YAML document, validates it, and returns the Eval.
// Skill, optional Input, and optional MCP are interpreted relative to the YAML file's directory.
// Skill must name an existing directory that contains SKILL.md.
// Input, when set, must name an existing directory.
// MCP, when set, must name an existing file (native MCP JSON).
func Load(path string) (*Eval, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("eval: read %s: %w", path, err)
	}

	var e Eval
	if err := yaml.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("eval: decode %s: %w", path, err)
	}
	if err := validate(&e, path); err != nil {
		return nil, err
	}
	return &e, nil
}

// ResolvePath joins rel with the eval YAML directory when rel is not absolute.
func ResolvePath(evalPath, rel string) string {
	if rel == "" || filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(filepath.Dir(evalPath), rel)
}

func validate(e *Eval, path string) error {
	if e.SchemaVersion != SchemaVersion {
		return fmt.Errorf("eval: %s: unsupported schemaVersion %d (want %d)", path, e.SchemaVersion, SchemaVersion)
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("eval: %s: name is required", path)
	}
	if strings.TrimSpace(e.Prompt) == "" {
		return fmt.Errorf("eval: %s: prompt is required", path)
	}
	if strings.TrimSpace(e.Skill) == "" {
		return fmt.Errorf("eval: %s: skill is required", path)
	}
	if len(e.Replies) == 0 {
		e.Replies = nil
	} else {
		for i, r := range e.Replies {
			if strings.TrimSpace(r) == "" {
				return fmt.Errorf("eval: %s: replies[%d] must be non-empty", path, i)
			}
		}
	}
	if e.Attempts <= 0 {
		e.Attempts = 1
	}
	if e.PassRate != nil {
		if e.PassRate.Min == nil {
			return fmt.Errorf("eval: %s: passRate.min is required when passRate is set", path)
		}
		min := *e.PassRate.Min
		if min < 0 || min > 1 {
			return fmt.Errorf("eval: %s: passRate.min must be between 0 and 1 (got %g)", path, min)
		}
	}

	skillPath := ResolvePath(path, e.Skill)
	info, err := os.Stat(skillPath)
	if err != nil {
		return fmt.Errorf("eval: %s: skill %q: %w", path, e.Skill, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("eval: %s: skill %q is not a directory", path, e.Skill)
	}
	skillMD := filepath.Join(skillPath, "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		return fmt.Errorf("eval: %s: skill %q: missing SKILL.md: %w", path, e.Skill, err)
	}

	if err := validateStringMatches(e, path); err != nil {
		return err
	}

	for filePath, fe := range e.Expects.Files {
		if err := validateWorkspaceRelPath(filePath); err != nil {
			return fmt.Errorf("eval: %s: files[%q]: %w", path, filePath, err)
		}
		if fe.Status == "" {
			continue
		}
		switch fe.Status {
		case result.FileCreated, result.FileModified, result.FileDeleted:
			// ok
		default:
			return fmt.Errorf("eval: %s: files[%q].status %q is invalid", path, filePath, fe.Status)
		}
	}

	if e.Input != "" {
		inputPath := ResolvePath(path, e.Input)
		info, err = os.Stat(inputPath)
		if err != nil {
			return fmt.Errorf("eval: %s: input %q: %w", path, e.Input, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("eval: %s: input %q is not a directory", path, e.Input)
		}
	}

	if e.MCP != "" {
		mcpPath := ResolvePath(path, e.MCP)
		info, err = os.Stat(mcpPath)
		if err != nil {
			return fmt.Errorf("eval: %s: mcp %q: %w", path, e.MCP, err)
		}
		if info.IsDir() {
			return fmt.Errorf("eval: %s: mcp %q is a directory (want a JSON file)", path, e.MCP)
		}
	}
	return nil
}

// validateWorkspaceRelPath rejects absolute or ..-escaping keys used under expects.files.
func validateWorkspaceRelPath(rel string) error {
	if rel == "" {
		return fmt.Errorf("path must be relative to workspace")
	}
	if filepath.IsAbs(rel) || filepath.IsAbs(filepath.FromSlash(rel)) {
		return fmt.Errorf("path must be relative to workspace")
	}
	cleaned := filepath.Clean(filepath.FromSlash(rel))
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || filepath.IsAbs(cleaned) {
		return fmt.Errorf("path must be relative to workspace")
	}
	return nil
}

func validateStringMatches(e *Eval, path string) error {
	if e.Expects.FinalMessage != nil {
		fm := e.Expects.FinalMessage
		if err := compileStringMatch(&fm.Contains); err != nil {
			return fmt.Errorf("eval: %s: finalMessage.contains: invalid regex: %w", path, err)
		}
		if err := compileStringMatch(&fm.Equals); err != nil {
			return fmt.Errorf("eval: %s: finalMessage.equals: invalid regex: %w", path, err)
		}
	}
	for filePath, fe := range e.Expects.Files {
		if err := compileStringMatch(&fe.Contains); err != nil {
			return fmt.Errorf("eval: %s: files[%q].contains: invalid regex: %w", path, filePath, err)
		}
		if err := compileStringMatch(&fe.Equals); err != nil {
			return fmt.Errorf("eval: %s: files[%q].equals: invalid regex: %w", path, filePath, err)
		}
		e.Expects.Files[filePath] = fe
	}
	if tc := e.Expects.ToolCalls; tc != nil {
		for i := range tc.Order {
			step := &tc.Order[i]
			if strings.TrimSpace(step.Name) == "" {
				return fmt.Errorf("eval: %s: toolCalls.order[%d]: name is required", path, i)
			}
			for argKey, ae := range step.Args {
				if err := compileStringMatch(&ae.Contains); err != nil {
					return fmt.Errorf("eval: %s: toolCalls.order[%d].args[%q].contains: invalid regex: %w", path, i, argKey, err)
				}
				if err := compileStringMatch(&ae.Equals); err != nil {
					return fmt.Errorf("eval: %s: toolCalls.order[%d].args[%q].equals: invalid regex: %w", path, i, argKey, err)
				}
				step.Args[argKey] = ae
			}
		}
	}
	return nil
}
