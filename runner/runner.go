// Package runner executes one eval attempt and produces a Result v1.
package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daniel-walters/skilleval/cost"
	"github.com/daniel-walters/skilleval/eval"
	"github.com/daniel-walters/skilleval/result"
	"github.com/daniel-walters/skilleval/skill"
)

// Options configure a single attempt run.
type Options struct {
	// Model is required for the default Cursor agent.
	Model string
	// Attempt is the 1-based try index (default 1).
	Attempt int
	// TotalAttempts is optional bookkeeping for Result.eval.
	TotalAttempts int
	// WorkDir is the parent directory for the attempt workspace (default: os.TempDir()).
	WorkDir string
	// Agent runs the model against the workspace. When nil, CursorAgent is used.
	Agent Agent
}

// AgentRequest is what the agent needs to execute one attempt.
type AgentRequest struct {
	Workspace string
	Prompt    string
	Model     string
	SkillName string
}

// AgentObservables are harness-normalized signals from one agent run.
type AgentObservables struct {
	ID           string
	Status       result.Status
	FinalMessage string
	Error        *string
	DurationMs   int64
	Turns        int
	ToolsUsed    []string
	ToolCalls    []result.ToolCall
	Usage        result.Usage
	Skills       result.Skills
	// CostUSD, when set, is used for Result.metrics.costUSD instead of rates.json.
	CostUSD *float64
}

// Agent executes a prompt against a workspace and returns observables.
// Each implementation owns skill placement, MCP seeding, runner identity, and outcome path skips.
type Agent interface {
	// RunnerID is recorded on Result.eval.runner (e.g. "cursor", "claude").
	RunnerID() string
	// PrepareWorkspace places the loaded skill for this backend.
	PrepareWorkspace(workspace string, sk *skill.Skill) error
	// SeedMCP writes native project MCP config from srcJSON into the workspace
	// (Cursor: .cursor/mcp.json; Claude: .mcp.json).
	SeedMCP(workspace, srcJSON string) error
	// IgnoreOutcomePath excludes agent-private trees from file outcomes.
	IgnoreOutcomePath(rel string) bool
	Run(ctx context.Context, req AgentRequest) (AgentObservables, error)
}

// Run executes one attempt for ev (loaded from evalPath) and returns the Result
// plus the attempt workspace path. The workspace is left on disk for the caller.
func Run(ctx context.Context, ev *eval.Eval, evalPath string, opts Options) (*result.Result, string, error) {
	if ev == nil {
		return nil, "", fmt.Errorf("runner: eval is nil")
	}
	if strings.TrimSpace(evalPath) == "" {
		return nil, "", fmt.Errorf("runner: evalPath is required")
	}

	attempt := opts.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	agent := opts.Agent
	if agent == nil {
		agent = DefaultAgent()
	}

	skillPath := eval.ResolvePath(evalPath, ev.Skill)
	sk, err := skill.Load(skillPath)
	if err != nil {
		return nil, "", fmt.Errorf("runner: %w", err)
	}

	parent := opts.WorkDir
	if parent == "" {
		parent = os.TempDir()
	}
	workspace, err := os.MkdirTemp(parent, "skilleval-*")
	if err != nil {
		return nil, "", fmt.Errorf("runner: create workspace: %w", err)
	}

	if ev.Input != "" {
		inputPath := eval.ResolvePath(evalPath, ev.Input)
		if err := copyTree(inputPath, workspace); err != nil {
			return nil, workspace, fmt.Errorf("runner: seed input: %w", err)
		}
	}

	if err := agent.PrepareWorkspace(workspace, sk); err != nil {
		return nil, workspace, fmt.Errorf("runner: place skill: %w", err)
	}

	if ev.MCP != "" {
		mcpPath := eval.ResolvePath(evalPath, ev.MCP)
		if err := agent.SeedMCP(workspace, mcpPath); err != nil {
			return nil, workspace, fmt.Errorf("runner: seed mcp: %w", err)
		}
	}

	before, err := snapshot(workspace, agent.IgnoreOutcomePath)
	if err != nil {
		return nil, workspace, fmt.Errorf("runner: snapshot: %w", err)
	}

	startedAt := time.Now().UTC()
	obs, err := agent.Run(ctx, AgentRequest{
		Workspace: workspace,
		Prompt:    ev.Prompt,
		Model:     opts.Model,
		SkillName: sk.Name,
	})
	finishedAt := time.Now().UTC()
	if err != nil {
		return nil, workspace, fmt.Errorf("runner: agent: %w", err)
	}

	after, err := snapshot(workspace, agent.IgnoreOutcomePath)
	if err != nil {
		return nil, workspace, fmt.Errorf("runner: snapshot after: %w", err)
	}
	files := diffSnapshots(before, after)

	if obs.Status == "" {
		obs.Status = result.StatusFinished
	}
	duration := obs.DurationMs
	if duration == 0 {
		duration = finishedAt.Sub(startedAt).Milliseconds()
	}

	toolsUsed := obs.ToolsUsed
	if toolsUsed == nil {
		toolsUsed = []string{}
	}
	toolCalls := obs.ToolCalls
	if toolCalls == nil {
		toolCalls = []result.ToolCall{}
	}
	activated := obs.Skills.Activated
	if activated == nil {
		activated = []string{}
	}
	if files == nil {
		files = map[string]result.FileOutcome{}
	}

	id := obs.ID
	if id == "" {
		id = fmt.Sprintf("run_%d", startedAt.UnixNano())
	}

	costUSD := obs.CostUSD
	if costUSD == nil {
		costUSD = cost.USD(opts.Model, obs.Usage)
	}

	r := &result.Result{
		SchemaVersion: result.SchemaVersion,
		ID:            id,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
		Eval: result.Eval{
			Name:          ev.Name,
			Prompt:        ev.Prompt,
			Skill:         sk.Name,
			Model:         opts.Model,
			Runner:        agent.RunnerID(),
			Attempt:       attempt,
			TotalAttempts: opts.TotalAttempts,
		},
		Status: obs.Status,
		Metrics: result.Metrics{
			Turns:      obs.Turns,
			DurationMs: duration,
			ToolsUsed:  toolsUsed,
			ToolCalls:  toolCalls,
			Usage:      obs.Usage,
			CostUSD:    costUSD,
		},
		Skills: result.Skills{
			Activated: activated,
		},
		Outcomes: result.Outcomes{
			Files: files,
		},
		Error:        obs.Error,
		FinalMessage: obs.FinalMessage,
	}
	return r, workspace, nil
}

type fileMeta struct {
	hash string
}

func snapshot(root string, ignore func(rel string) bool) (map[string]fileMeta, error) {
	out := make(map[string]fileMeta)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if ignore != nil && ignore(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		sum, err := hashFile(path)
		if err != nil {
			return err
		}
		out[rel] = fileMeta{hash: sum}
		return nil
	})
	return out, err
}

// PlaceSkillUnder copies sk into workspace/<skillsRel...>/<name>/.
func PlaceSkillUnder(workspace string, skillsRel []string, sk *skill.Skill) error {
	if sk == nil {
		return fmt.Errorf("skill is nil")
	}
	parts := append([]string{workspace}, skillsRel...)
	skillsRoot := filepath.Join(parts...)
	skillDest, err := skillDestUnder(skillsRoot, sk.Name)
	if err != nil {
		return err
	}
	return copyTree(sk.Dir, skillDest)
}

func diffSnapshots(before, after map[string]fileMeta) map[string]result.FileOutcome {
	files := make(map[string]result.FileOutcome)
	for path, meta := range after {
		prev, ok := before[path]
		if !ok {
			files[path] = result.FileOutcome{Status: result.FileCreated}
			continue
		}
		if prev.hash != meta.hash {
			files[path] = result.FileOutcome{Status: result.FileModified}
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			files[path] = result.FileOutcome{Status: result.FileDeleted}
		}
	}
	return files
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// skillDestUnder joins skillsRoot and name, ensuring the result stays under skillsRoot.
func skillDestUnder(skillsRoot, name string) (string, error) {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, `\`) || name == ".." || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid skill name %q", name)
	}
	root, err := filepath.Abs(skillsRoot)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(root, name)
	dest, err = filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("skill name %q escapes skills directory", name)
	}
	if rel != name {
		return "", fmt.Errorf("skill name %q escapes skills directory", name)
	}
	return dest, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
