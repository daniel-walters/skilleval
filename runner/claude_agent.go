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

	"github.com/daniel-walters/skilleval/skill"
)

//go:embed claudeagent/run.mjs claudeagent/skills.mjs claudeagent/package.json claudeagent/package-lock.json toolargs.mjs agentlog.mjs legagg.mjs
var claudeAssets embed.FS

// ClaudeAgent invokes the embedded Node helper with @anthropic-ai/claude-agent-sdk.
type ClaudeAgent struct {
	// Node is the node executable (default: "node").
	Node string
	// HelperDir overrides where run.mjs and node_modules live.
	// When empty, a temp dir is prepared from embedded assets once.
	HelperDir string

	mu         sync.Mutex
	prepared   string
	prepareErr error
}

// RunnerID implements Agent.
func (a *ClaudeAgent) RunnerID() string { return "claude" }

// PrepareWorkspace places the skill under .claude/skills/<name>/.
func (a *ClaudeAgent) PrepareWorkspace(workspace string, sk *skill.Skill) error {
	return PlaceSkillUnder(workspace, []string{".claude", "skills"}, sk)
}

// SeedMCP writes srcJSON to workspace/.mcp.json.
func (a *ClaudeAgent) SeedMCP(workspace, srcJSON string) error {
	return copyFile(srcJSON, filepath.Join(workspace, ".mcp.json"))
}

// IgnoreOutcomePath skips Claude's private skill tree and seeded MCP config
// from file outcomes.
func (a *ClaudeAgent) IgnoreOutcomePath(rel string) bool {
	return rel == ".claude" || strings.HasPrefix(rel, ".claude/") || rel == ".mcp.json"
}

// Run executes one Claude agent attempt and returns normalized observables.
func (a *ClaudeAgent) Run(ctx context.Context, req AgentRequest) (AgentObservables, error) {
	if req.Workspace == "" {
		return AgentObservables{}, fmt.Errorf("claudeagent: workspace is required")
	}
	if req.Model == "" {
		return AgentObservables{}, fmt.Errorf("claudeagent: model is required")
	}
	if req.Prompt == "" {
		return AgentObservables{}, fmt.Errorf("claudeagent: prompt is required")
	}

	helperDir, err := a.ensureHelper()
	if err != nil {
		return AgentObservables{}, err
	}

	node := a.Node
	if node == "" {
		node = "node"
	}

	args := []string{
		filepath.Join(helperDir, "run.mjs"),
		"--cwd", req.Workspace,
		"--model", req.Model,
		"--prompt", req.Prompt,
	}
	args = appendReplyArgs(args, req.Replies)
	if req.SkillName != "" {
		args = append(args, "--skill", req.SkillName)
	}

	cmd := exec.CommandContext(ctx, node, args...)
	cmd.Dir = helperDir
	cmd.Env = os.Environ()
	configureAgentCmd(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return AgentObservables{}, fmt.Errorf("claudeagent: node helper: %w\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	payload, err := extractJSONObject(stdout.Bytes())
	if err != nil {
		return AgentObservables{}, fmt.Errorf("claudeagent: decode helper output: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	var raw helperOutput
	if err := json.Unmarshal(payload, &raw); err != nil {
		return AgentObservables{}, fmt.Errorf("claudeagent: decode helper output: %w\nstdout: %s", err, stdout.String())
	}
	return mapHelperOutput(raw), nil
}

func (a *ClaudeAgent) ensureHelper() (string, error) {
	if a.HelperDir != "" {
		return a.HelperDir, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.prepared != "" || a.prepareErr != nil {
		return a.prepared, a.prepareErr
	}
	dir, err := prepareClaudeHelperDir()
	if err != nil {
		a.prepareErr = err
		return "", err
	}
	a.prepared = dir
	return dir, nil
}

func prepareClaudeHelperDir() (string, error) {
	dir, err := os.MkdirTemp("", "skilleval-claudeagent-*")
	if err != nil {
		return "", fmt.Errorf("claudeagent: temp dir: %w", err)
	}
	for _, name := range []string{"run.mjs", "skills.mjs", "package.json", "package-lock.json", "toolargs.mjs", "agentlog.mjs", "legagg.mjs"} {
		src := "claudeagent/" + name
		if name == "toolargs.mjs" || name == "agentlog.mjs" || name == "legagg.mjs" {
			src = name
		}
		data, err := claudeAssets.ReadFile(src)
		if err != nil {
			return "", fmt.Errorf("claudeagent: embed %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return "", fmt.Errorf("claudeagent: write %s: %w", name, err)
		}
	}
	cmd := exec.Command("npm", "ci", "--omit=dev", "--no-fund", "--no-audit")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claudeagent: npm ci: %w\nstderr: %s", err, stderr.String())
	}
	return dir, nil
}
