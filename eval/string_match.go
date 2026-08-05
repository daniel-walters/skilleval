package eval

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringMatch is a literal string or a slash-delimited regex (/pattern/).
// A value that starts and ends with '/' (length >= 2) is a regex; the pattern
// is the interior. Other strings are literals. A literal that itself looks
// like /.../ cannot be expressed.
type StringMatch struct {
	raw     string
	pattern string
	regex   bool
	re      *regexp.Regexp
}

// IsSet reports whether a non-empty expect value was provided.
func (m StringMatch) IsSet() bool {
	return m.raw != ""
}

// IsZero reports whether the match is unset (for yaml omitempty).
func (m StringMatch) IsZero() bool {
	return m.raw == ""
}

// IsRegex reports whether the value was slash-delimited.
func (m StringMatch) IsRegex() bool {
	return m.regex
}

// String returns the original YAML form (literal or /pattern/).
func (m StringMatch) String() string {
	return m.raw
}

// MatchContains checks substring (literal) or MatchString (regex).
func (m StringMatch) MatchContains(haystack string) bool {
	if !m.IsSet() {
		return true
	}
	if m.regex {
		return m.re.MatchString(haystack)
	}
	return strings.Contains(haystack, m.pattern)
}

// MatchEquals checks exact equality (literal) or full-string regex match.
func (m StringMatch) MatchEquals(haystack string) bool {
	if !m.IsSet() {
		return true
	}
	if m.regex {
		loc := m.re.FindStringIndex(haystack)
		return loc != nil && loc[0] == 0 && loc[1] == len(haystack)
	}
	return haystack == m.pattern
}

// UnmarshalYAML decodes a YAML string into a literal or /regex/ match.
func (m *StringMatch) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	*m = parseStringMatch(s)
	return nil
}

// MarshalYAML encodes the original string form.
func (m StringMatch) MarshalYAML() (interface{}, error) {
	return m.raw, nil
}

func parseStringMatch(s string) StringMatch {
	if len(s) >= 2 && s[0] == '/' && s[len(s)-1] == '/' {
		return StringMatch{raw: s, pattern: s[1 : len(s)-1], regex: true}
	}
	return StringMatch{raw: s, pattern: s, regex: false}
}

func compileStringMatch(m *StringMatch) error {
	if m == nil || !m.regex {
		return nil
	}
	if m.re != nil {
		return nil
	}
	re, err := regexp.Compile(m.pattern)
	if err != nil {
		return err
	}
	m.re = re
	return nil
}
