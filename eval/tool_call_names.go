package eval

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ToolCallNames is one or more exact tool names for an order step.
type ToolCallNames []string

// Match reports whether callName equals one of the names.
func (n ToolCallNames) Match(callName string) bool {
	for _, name := range n {
		if callName == name {
			return true
		}
	}
	return false
}

// String formats names for error text: one name as %q, several as a JSON list.
func (n ToolCallNames) String() string {
	if len(n) == 1 {
		return fmt.Sprintf("%q", n[0])
	}
	b, err := json.Marshal([]string(n))
	if err != nil {
		return fmt.Sprint([]string(n))
	}
	return string(b)
}

func (n ToolCallNames) validate() error {
	if len(n) == 0 {
		return fmt.Errorf("name is required")
	}
	for _, name := range n {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("name is required")
		}
	}
	return nil
}

// UnmarshalYAML decodes a YAML string or a list of strings.
func (n *ToolCallNames) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*n = ToolCallNames{s}
	case yaml.SequenceNode:
		var ss []string
		if err := value.Decode(&ss); err != nil {
			return err
		}
		*n = ss
	default:
		return fmt.Errorf("name must be a string or a list of strings")
	}
	return nil
}

// MarshalYAML encodes one name as a scalar and several names as a sequence.
func (n ToolCallNames) MarshalYAML() (interface{}, error) {
	if len(n) == 1 {
		return n[0], nil
	}
	return []string(n), nil
}
