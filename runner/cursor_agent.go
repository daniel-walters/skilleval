package runner

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/daniel-walters/skilleval/result"
	"github.com/daniel-walters/skilleval/skill"
)

//go:embed cursoragent/run.mjs cursoragent/turns.mjs cursoragent/skills.mjs cursoragent/package.json cursoragent/package-lock.json
var cursorAssets embed.FS

// CursorAgent invokes the embedded Node helper with @cursor/sdk.
type CursorAgent struct {
	// Node is the node executable (default: "node").
	Node string
	// HelperDir overrides where run.mjs and node_modules live.
	// When empty, a temp dir is prepared from embedded assets once.
	HelperDir string

	mu         sync.Mutex
	prepared   string
	prepareErr error
}

// DefaultAgent returns the Cursor Node @cursor/sdk agent adapter.
func DefaultAgent() Agent {
	return &CursorAgent{}
}

// RunnerID implements Agent.
func (a *CursorAgent) RunnerID() string { return "cursor" }

// PrepareWorkspace places the skill under .cursor/skills/<name>/.
func (a *CursorAgent) PrepareWorkspace(workspace string, sk *skill.Skill) error {
	return PlaceSkillUnder(workspace, []string{".cursor", "skills"}, sk)
}

// SeedMCP writes srcJSON to workspace/.cursor/mcp.json.
func (a *CursorAgent) SeedMCP(workspace, srcJSON string) error {
	return copyFile(srcJSON, filepath.Join(workspace, ".cursor", "mcp.json"))
}

// IgnoreOutcomePath skips Cursor's private skill tree from file outcomes.
func (a *CursorAgent) IgnoreOutcomePath(rel string) bool {
	return rel == ".cursor" || strings.HasPrefix(rel, ".cursor/")
}

// Run executes one Cursor agent attempt and returns normalized observables.
func (a *CursorAgent) Run(ctx context.Context, req AgentRequest) (AgentObservables, error) {
	if req.Workspace == "" {
		return AgentObservables{}, fmt.Errorf("cursoragent: workspace is required")
	}
	if req.Model == "" {
		return AgentObservables{}, fmt.Errorf("cursoragent: model is required")
	}
	if req.Prompt == "" {
		return AgentObservables{}, fmt.Errorf("cursoragent: prompt is required")
	}

	helperDir, err := a.ensureHelper()
	if err != nil {
		return AgentObservables{}, err
	}

	node := a.Node
	if node == "" {
		node = "node"
	}

	cmd := exec.CommandContext(ctx, node, filepath.Join(helperDir, "run.mjs"),
		"--cwd", req.Workspace,
		"--model", req.Model,
		"--prompt", req.Prompt,
	)
	cmd.Dir = helperDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return AgentObservables{}, fmt.Errorf("cursoragent: node helper: %w\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	payload, err := extractJSONObject(stdout.Bytes())
	if err != nil {
		return AgentObservables{}, fmt.Errorf("cursoragent: decode helper output: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	var raw helperOutput
	if err := json.Unmarshal(payload, &raw); err != nil {
		return AgentObservables{}, fmt.Errorf("cursoragent: decode helper output: %w\nstdout: %s", err, stdout.String())
	}
	return mapHelperOutput(raw), nil
}

// extractJSONObject returns the last top-level JSON object in b.
// The Node helper prints one JSON blob to stdout, but @cursor/sdk may
// also emit log lines on stdout before it.
func extractJSONObject(b []byte) ([]byte, error) {
	lines := bytes.Split(b, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		if json.Valid(line) {
			return line, nil
		}
	}
	// Fallback: last '{' … end (multiline JSON, no trailing logs after).
	start := bytes.LastIndexByte(b, '{')
	if start < 0 {
		return nil, fmt.Errorf("no JSON object found")
	}
	candidate := bytes.TrimSpace(b[start:])
	if json.Valid(candidate) {
		return candidate, nil
	}
	return nil, fmt.Errorf("no valid JSON object found")
}

type helperOutput struct {
	ID           string           `json:"id"`
	Status       string           `json:"status"`
	FinalMessage string           `json:"finalMessage"`
	Error        *string          `json:"error"`
	DurationMs   int64            `json:"durationMs"`
	Turns        int              `json:"turns"`
	ToolsUsed    []string         `json:"toolsUsed"`
	ToolCalls    []helperToolCall `json:"toolCalls"`
	Usage        result.Usage     `json:"usage"`
	Skills       result.Skills    `json:"skills"`
	CostUSD      *float64         `json:"costUSD"`
}

type helperToolCall struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func mapHelperOutput(raw helperOutput) AgentObservables {
	status := result.Status(raw.Status)
	switch status {
	case result.StatusFinished, result.StatusError, result.StatusCancelled:
	default:
		// Unknown/future SDK statuses are failures, not successful completion.
		status = result.StatusError
	}
	fallback := result.ToolCallCompleted
	if status == result.StatusError || status == result.StatusCancelled {
		fallback = result.ToolCallError
	}
	calls := make([]result.ToolCall, 0, len(raw.ToolCalls))
	for _, c := range raw.ToolCalls {
		st := result.ToolCallStatus(c.Status)
		switch st {
		case result.ToolCallRunning:
			st = fallback
		case result.ToolCallCompleted, result.ToolCallError:
		default:
			st = result.ToolCallCompleted
		}
		calls = append(calls, result.ToolCall{Name: c.Name, Status: st})
	}
	tools := raw.ToolsUsed
	if tools == nil {
		tools = []string{}
	}
	activated := raw.Skills.Activated
	if activated == nil {
		activated = []string{}
	}
	return AgentObservables{
		ID:           raw.ID,
		Status:       status,
		FinalMessage: raw.FinalMessage,
		Error:        raw.Error,
		DurationMs:   raw.DurationMs,
		Turns:        raw.Turns,
		ToolsUsed:    tools,
		ToolCalls:    calls,
		Usage:        raw.Usage,
		Skills: result.Skills{
			Activated: activated,
		},
		CostUSD: raw.CostUSD,
	}
}

func (a *CursorAgent) ensureHelper() (string, error) {
	if a.HelperDir != "" {
		return a.HelperDir, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.prepared != "" || a.prepareErr != nil {
		return a.prepared, a.prepareErr
	}
	dir, err := prepareCursorHelperDir()
	if err != nil {
		a.prepareErr = err
		return "", err
	}
	a.prepared = dir
	return dir, nil
}

func prepareCursorHelperDir() (string, error) {
	dir, err := os.MkdirTemp("", "skilleval-cursoragent-*")
	if err != nil {
		return "", fmt.Errorf("cursoragent: temp dir: %w", err)
	}
	for _, name := range []string{"run.mjs", "turns.mjs", "skills.mjs", "package.json", "package-lock.json"} {
		data, err := cursorAssets.ReadFile("cursoragent/" + name)
		if err != nil {
			return "", fmt.Errorf("cursoragent: embed %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return "", fmt.Errorf("cursoragent: write %s: %w", name, err)
		}
	}
	cmd := exec.Command("npm", "ci", "--omit=dev", "--no-fund", "--no-audit")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cursoragent: npm ci: %w\nstderr: %s", err, stderr.String())
	}
	return dir, nil
}
