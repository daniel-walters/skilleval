// Package skill loads SKILL.md packages from a filesystem path.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// namePattern restricts skill names to a single safe path segment so placement
// under a runner skills directory cannot escape the attempt workspace.
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Skill is a loaded skill package.
type Skill struct {
	// Name is the frontmatter name (used in Result and expects).
	Name string
	// Description is the optional frontmatter description.
	Description string
	// Dir is the absolute path to the skill directory (contains SKILL.md).
	Dir string
	// Body is the markdown body after frontmatter.
	Body string
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Load reads a skill from path. Path may be a skill directory or a SKILL.md file.
func Load(path string) (*Skill, error) {
	dir, skillMD, err := resolve(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(skillMD)
	if err != nil {
		return nil, fmt.Errorf("skill: read %s: %w", skillMD, err)
	}
	fm, body, err := parseFrontmatter(string(raw))
	if err != nil {
		return nil, fmt.Errorf("skill: %s: %w", skillMD, err)
	}
	name := strings.TrimSpace(fm.Name)
	if name == "" {
		return nil, fmt.Errorf("skill: %s: name is required in frontmatter", skillMD)
	}
	if !namePattern.MatchString(name) || strings.Contains(name, "..") {
		return nil, fmt.Errorf("skill: %s: name %q must match %s", skillMD, name, namePattern.String())
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("skill: abs %s: %w", dir, err)
	}
	return &Skill{
		Name:        name,
		Description: strings.TrimSpace(fm.Description),
		Dir:         absDir,
		Body:        body,
	}, nil
}

func resolve(path string) (dir, skillMD string, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("skill: %s: %w", path, err)
	}
	if info.IsDir() {
		dir = path
		skillMD = filepath.Join(path, "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			return "", "", fmt.Errorf("skill: %s: missing SKILL.md: %w", path, err)
		}
		return dir, skillMD, nil
	}
	if filepath.Base(path) != "SKILL.md" {
		return "", "", fmt.Errorf("skill: %s: path must be a skill directory or SKILL.md", path)
	}
	return filepath.Dir(path), path, nil
}

func parseFrontmatter(content string) (frontmatter, string, error) {
	const delim = "---"
	trimmed := strings.TrimLeft(content, "\ufeff \t\r\n")
	if !strings.HasPrefix(trimmed, delim) {
		return frontmatter{}, "", fmt.Errorf("missing YAML frontmatter")
	}
	rest := trimmed[len(delim):]
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n"+delim)
	if end < 0 {
		// allow ---\n...\n--- at end without trailing newline body
		if strings.HasSuffix(strings.TrimRight(rest, "\r\n"), delim) {
			yamlPart := strings.TrimSuffix(strings.TrimRight(rest, "\r\n"), delim)
			yamlPart = strings.TrimRight(yamlPart, "\r\n")
			var fm frontmatter
			if err := yaml.Unmarshal([]byte(yamlPart), &fm); err != nil {
				return frontmatter{}, "", fmt.Errorf("decode frontmatter: %w", err)
			}
			return fm, "", nil
		}
		return frontmatter{}, "", fmt.Errorf("unclosed YAML frontmatter")
	}
	yamlPart := rest[:end]
	body := rest[end+len("\n"+delim):]
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimLeft(body, "\r\n")

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlPart), &fm); err != nil {
		return frontmatter{}, "", fmt.Errorf("decode frontmatter: %w", err)
	}
	return fm, body, nil
}
