// Package result defines the skilleval Result contract (v1):
// a lean JSON artifact for one attempt of one eval.
package result

import "time"

// SchemaVersion is the current Result JSON schema version.
const SchemaVersion = 1

// Status is the agent run completion state (not expectation pass/fail).
type Status string

const (
	StatusFinished  Status = "finished"
	StatusError     Status = "error"
	StatusCancelled Status = "cancelled"
)

// FileStatus is how a path changed in the attempt workspace.
type FileStatus string

const (
	FileCreated  FileStatus = "created"
	FileModified FileStatus = "modified"
	FileDeleted  FileStatus = "deleted"
)

// ToolCallStatus is the completion state of a single tool invocation.
type ToolCallStatus string

const (
	ToolCallRunning   ToolCallStatus = "running"
	ToolCallCompleted ToolCallStatus = "completed"
	ToolCallError     ToolCallStatus = "error"
)

// Result is one attempt of one eval.
type Result struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	StartedAt     time.Time `json:"startedAt"`
	FinishedAt    time.Time `json:"finishedAt"`

	Eval     Eval     `json:"eval"`
	Status   Status   `json:"status"`
	Metrics  Metrics  `json:"metrics"`
	Skills   Skills   `json:"skills"`
	Outcomes Outcomes `json:"outcomes"`

	// Error is set when Status is StatusError.
	Error *string `json:"error"`

	// FinalMessage is the last assistant text from the run.
	FinalMessage string `json:"finalMessage"`
}

// Eval identifies the eval attempt and how it was run.
type Eval struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
	Skill  string `json:"skill"`
	Model  string `json:"model"`
	Runner string `json:"runner"`

	// Attempt is the 1-based try index for this Result.
	Attempt int `json:"attempt"`
	// TotalAttempts is how many tries were requested; omitted when unknown (0).
	TotalAttempts int `json:"totalAttempts,omitempty"`
}

// Metrics are observable run stats for deterministic checks.
type Metrics struct {
	Turns      int        `json:"turns"`
	DurationMs int64      `json:"durationMs"`
	ToolsUsed  []string   `json:"toolsUsed"`
	ToolCalls  []ToolCall `json:"toolCalls"`
	Usage      Usage      `json:"usage"`
	// CostUSD is nil when cost is unknown / not computed.
	CostUSD *float64 `json:"costUSD"`
}

// ToolCall is one tool invocation in order.
type ToolCall struct {
	Name   string         `json:"name"`
	Status ToolCallStatus `json:"status"`
	// Args are lean pre-call arguments (path, command, and other small scalars).
	// Large body fields are stripped by the runner. Omitted when empty.
	Args map[string]any `json:"args,omitempty"`
}

// Usage is token accounting for the run.
type Usage struct {
	InputTokens      int `json:"inputTokens"`
	OutputTokens     int `json:"outputTokens"`
	CacheReadTokens  int `json:"cacheReadTokens"`
	CacheWriteTokens int `json:"cacheWriteTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// Skills records activation when the runner can observe it.
// For Cursor, a skill is activated when the agent completes a read of its SKILL.md.
type Skills struct {
	Activated []string `json:"activated"`
}

// Outcomes holds workspace change metadata (no file bodies).
type Outcomes struct {
	// Files maps relative paths to change status. Untouched paths are omitted.
	Files map[string]FileOutcome `json:"files"`
}

// FileOutcome is the change status for a single path.
type FileOutcome struct {
	Status FileStatus `json:"status"`
}
