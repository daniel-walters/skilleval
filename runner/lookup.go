package runner

import (
	"fmt"
	"strings"
)

// LookupAgent returns the agent implementation for a runner name.
// Empty name defaults to Cursor.
func LookupAgent(name string) (Agent, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "cursor":
		return &CursorAgent{}, nil
	case "claude":
		return &ClaudeAgent{}, nil
	default:
		return nil, fmt.Errorf("unknown runner %q (want cursor or claude)", name)
	}
}
