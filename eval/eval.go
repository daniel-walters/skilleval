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
	SchemaVersion int     `yaml:"schemaVersion"`
	Name          string  `yaml:"name"`
	Prompt        string  `yaml:"prompt"`
	Skill         string  `yaml:"skill"`
	Input         string  `yaml:"input,omitempty"`
	Expects       Expects `yaml:"expects,omitempty"`
}

// Expects are deterministic predicates over a Result (and workspace files).
type Expects struct {
	Turns        *TurnsExpect          `yaml:"turns,omitempty"`
	CostUSD      *CostExpect           `yaml:"costUSD,omitempty"`
	ToolsUsed    *ToolsUsedExpect      `yaml:"toolsUsed,omitempty"`
	Skills       *SkillsExpect         `yaml:"skills,omitempty"`
	Files        map[string]FileExpect `yaml:"files,omitempty"`
	FinalMessage *TextExpect           `yaml:"finalMessage,omitempty"`
}

// TurnsExpect bounds run turn count.
type TurnsExpect struct {
	Max *int `yaml:"max,omitempty"`
}

// CostExpect bounds run cost in USD.
type CostExpect struct {
	Max *float64 `yaml:"max,omitempty"`
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

// StringSetExpect requires listed values to be present.
type StringSetExpect struct {
	Includes []string `yaml:"includes,omitempty"`
}

// FileExpect checks a path's outcome status and/or content.
type FileExpect struct {
	Status   result.FileStatus `yaml:"status,omitempty"`
	Contains string            `yaml:"contains,omitempty"`
	Equals   string            `yaml:"equals,omitempty"`
}

// TextExpect checks finalMessage (or similar text fields).
type TextExpect struct {
	Contains string `yaml:"contains,omitempty"`
	Equals   string `yaml:"equals,omitempty"`
}

// Load reads path as an eval YAML document, validates it, and returns the Eval.
// When Input is set, it is interpreted relative to the YAML file's directory
// and must name an existing directory.
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

	for filePath, fe := range e.Expects.Files {
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

	if e.Input == "" {
		return nil
	}
	inputPath := e.Input
	if !filepath.IsAbs(inputPath) {
		inputPath = filepath.Join(filepath.Dir(path), e.Input)
	}
	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("eval: %s: input %q: %w", path, e.Input, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("eval: %s: input %q is not a directory", path, e.Input)
	}
	return nil
}
