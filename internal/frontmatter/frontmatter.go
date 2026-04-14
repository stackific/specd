// Package frontmatter handles YAML frontmatter parsing and rendering for
// specd markdown files. It supports spec and task frontmatter schemas,
// round-trip encoding, and extraction of acceptance criteria checkboxes
// from the "## Acceptance criteria" body section.
package frontmatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SpecFrontmatter represents the YAML frontmatter of a spec file.
type SpecFrontmatter struct {
	Title       string        `yaml:"title"`
	Type        string        `yaml:"type"`
	Summary     string        `yaml:"summary"`
	LinkedSpecs []string      `yaml:"linked_specs,omitempty"`
	Cites       []CitationRef `yaml:"cites,omitempty"`
}

// TaskFrontmatter represents the YAML frontmatter of a task file.
type TaskFrontmatter struct {
	Title       string        `yaml:"title"`
	Status      string        `yaml:"status"`
	Summary     string        `yaml:"summary"`
	LinkedTasks []string      `yaml:"linked_tasks,omitempty"`
	DependsOn   []string      `yaml:"depends_on,omitempty"`
	Cites       []CitationRef `yaml:"cites,omitempty"`
}

// CitationRef is a reference to a KB doc and optional chunk positions.
type CitationRef struct {
	KB     string `yaml:"kb"`
	Chunks []int  `yaml:"chunks,omitempty"`
}

// ParsedDocument holds the frontmatter and body of a markdown file.
type ParsedDocument struct {
	RawFrontmatter string
	Body           string
}

// Parse splits a markdown file into frontmatter (YAML string) and body.
// Returns an error if the file has no valid frontmatter delimiters.
func Parse(content string) (*ParsedDocument, error) {
	// Trim leading BOM or whitespace.
	content = strings.TrimLeft(content, "\xef\xbb\xbf")

	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("missing opening frontmatter delimiter")
	}

	// Find closing delimiter.
	rest := content[3:]
	// Skip the newline after opening ---
	if idx := strings.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		return nil, fmt.Errorf("malformed frontmatter: no newline after opening delimiter")
	}

	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return nil, fmt.Errorf("missing closing frontmatter delimiter")
	}

	rawFM := rest[:endIdx]
	body := rest[endIdx+4:] // skip \n---
	// Strip leading newline from body.
	body = strings.TrimLeft(body, "\n")

	return &ParsedDocument{
		RawFrontmatter: rawFM,
		Body:           body,
	}, nil
}

// DecodeSpec parses YAML frontmatter into SpecFrontmatter.
func DecodeSpec(raw string) (*SpecFrontmatter, error) {
	var fm SpecFrontmatter
	if err := yaml.Unmarshal([]byte(raw), &fm); err != nil {
		return nil, fmt.Errorf("decode spec frontmatter: %w", err)
	}
	return &fm, nil
}

// DecodeTask parses YAML frontmatter into TaskFrontmatter.
func DecodeTask(raw string) (*TaskFrontmatter, error) {
	var fm TaskFrontmatter
	if err := yaml.Unmarshal([]byte(raw), &fm); err != nil {
		return nil, fmt.Errorf("decode task frontmatter: %w", err)
	}
	return &fm, nil
}

// EncodeSpec renders SpecFrontmatter back to YAML.
func EncodeSpec(fm *SpecFrontmatter) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// EncodeTask renders TaskFrontmatter back to YAML.
func EncodeTask(fm *TaskFrontmatter) (string, error) {
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RenderSpec produces a complete markdown file from frontmatter and body.
func RenderSpec(fm *SpecFrontmatter, body string) (string, error) {
	yml, err := EncodeSpec(fm)
	if err != nil {
		return "", err
	}
	return "---\n" + yml + "---\n\n" + body, nil
}

// RenderTask produces a complete markdown file from frontmatter and body.
func RenderTask(fm *TaskFrontmatter, body string) (string, error) {
	yml, err := EncodeTask(fm)
	if err != nil {
		return "", err
	}
	return "---\n" + yml + "---\n\n" + body, nil
}

// ParseCriteria extracts acceptance criteria from the body's
// "## Acceptance criteria" section. Returns a list of (text, checked) pairs.
func ParseCriteria(body string) []Criterion {
	var criteria []Criterion
	inSection := false

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## Acceptance criteria") {
			inSection = true
			continue
		}

		// End of section on next heading.
		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}

		if !inSection {
			continue
		}

		if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
			criteria = append(criteria, Criterion{
				Text:    strings.TrimSpace(trimmed[6:]),
				Checked: true,
			})
		} else if strings.HasPrefix(trimmed, "- [ ] ") {
			criteria = append(criteria, Criterion{
				Text:    strings.TrimSpace(trimmed[6:]),
				Checked: false,
			})
		}
	}

	return criteria
}

// Criterion represents a single acceptance criterion.
type Criterion struct {
	Text    string
	Checked bool
}
