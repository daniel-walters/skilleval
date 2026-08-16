package eval

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Shell tool names that may carry an exit code (exact, runner-specific).
const (
	ShellToolName      = "shell"
	ClaudeBashToolName = "Bash"
)

// IsShellToolName reports whether name is shell or Bash.
func IsShellToolName(name string) bool {
	return name == ShellToolName || name == ClaudeBashToolName
}

// ExitCodeExpect is one or more process exit codes for an order step.
// A call matches if its exit code equals one of these integers.
type ExitCodeExpect []int

// Match reports whether code equals one of the expected integers.
func (e ExitCodeExpect) Match(code *int) bool {
	if code == nil || len(e) == 0 {
		return false
	}
	for _, want := range e {
		if *code == want {
			return true
		}
	}
	return false
}

func (e ExitCodeExpect) validate() error {
	if len(e) == 0 {
		return fmt.Errorf("exitCode must not be empty")
	}
	return nil
}

// UnmarshalYAML decodes a YAML integer or a list of integers.
func (e *ExitCodeExpect) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		n, err := decodeYAMLInt(value)
		if err != nil {
			return err
		}
		*e = ExitCodeExpect{n}
	case yaml.SequenceNode:
		out := make(ExitCodeExpect, 0, len(value.Content))
		for _, node := range value.Content {
			n, err := decodeYAMLInt(node)
			if err != nil {
				return err
			}
			out = append(out, n)
		}
		*e = out
	default:
		return fmt.Errorf("exitCode must be an integer or a list of integers")
	}
	return nil
}

// MarshalYAML encodes one code as a scalar and several codes as a sequence.
func (e ExitCodeExpect) MarshalYAML() (interface{}, error) {
	if len(e) == 1 {
		return e[0], nil
	}
	return []int(e), nil
}

func decodeYAMLInt(n *yaml.Node) (int, error) {
	if n == nil || n.Kind != yaml.ScalarNode {
		return 0, fmt.Errorf("exitCode must be an integer")
	}
	i, err := strconv.Atoi(n.Value)
	if err != nil {
		return 0, fmt.Errorf("exitCode must be an integer")
	}
	return i, nil
}
