package frontmatter

import (
	"testing"
)

const specMD = `---
title: OAuth with GitHub
type: technical
summary: OAuth flow using GitHub as an identity provider
linked_specs: [SPEC-1, SPEC-7]
cites:
  - kb: KB-4
    chunks: [12, 15]
  - kb: KB-9
    chunks: [3]
---

# OAuth with GitHub

Freeform body.
`

const taskMD = `---
title: Protection middleware
status: in_progress
summary: Add auth middleware to routes requiring authentication
linked_tasks: [TASK-2]
depends_on: [TASK-1]
cites:
  - kb: KB-4
    chunks: [12]
---

# Protection middleware

Freeform body.

## Acceptance criteria

- [x] Middleware rejects requests with invalid JWT (401)
- [ ] Middleware rejects requests with no Authorization header (401)
- [ ] Middleware attaches user context on valid JWT
- [ ] Unit tests cover all three cases
`

func TestParseSpec(t *testing.T) {
	doc, err := Parse(specMD)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	fm, err := DecodeSpec(doc.RawFrontmatter)
	if err != nil {
		t.Fatalf("DecodeSpec: %v", err)
	}

	if fm.Title != "OAuth with GitHub" {
		t.Errorf("title = %q", fm.Title)
	}
	if fm.Type != "technical" {
		t.Errorf("type = %q", fm.Type)
	}
	if len(fm.LinkedSpecs) != 2 {
		t.Errorf("linked_specs len = %d", len(fm.LinkedSpecs))
	}
	if len(fm.Cites) != 2 {
		t.Errorf("cites len = %d", len(fm.Cites))
	}
	if fm.Cites[0].KB != "KB-4" || len(fm.Cites[0].Chunks) != 2 {
		t.Errorf("first cite = %+v", fm.Cites[0])
	}

	if doc.Body == "" {
		t.Error("body is empty")
	}
}

func TestParseTask(t *testing.T) {
	doc, err := Parse(taskMD)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	fm, err := DecodeTask(doc.RawFrontmatter)
	if err != nil {
		t.Fatalf("DecodeTask: %v", err)
	}

	if fm.Title != "Protection middleware" {
		t.Errorf("title = %q", fm.Title)
	}
	if fm.Status != "in_progress" {
		t.Errorf("status = %q", fm.Status)
	}
	if len(fm.DependsOn) != 1 || fm.DependsOn[0] != "TASK-1" {
		t.Errorf("depends_on = %v", fm.DependsOn)
	}
}

func TestParseCriteria(t *testing.T) {
	doc, err := Parse(taskMD)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	criteria := ParseCriteria(doc.Body)
	if len(criteria) != 4 {
		t.Fatalf("criteria len = %d, want 4", len(criteria))
	}

	if !criteria[0].Checked {
		t.Error("first criterion should be checked")
	}
	if criteria[1].Checked {
		t.Error("second criterion should be unchecked")
	}
	if criteria[0].Text != "Middleware rejects requests with invalid JWT (401)" {
		t.Errorf("first criterion text = %q", criteria[0].Text)
	}
}

func TestRenderSpecRoundtrip(t *testing.T) {
	fm := &SpecFrontmatter{
		Title:   "Test Spec",
		Type:    "technical",
		Summary: "A test spec",
	}
	body := "# Test Spec\n\nBody content.\n"

	rendered, err := RenderSpec(fm, body)
	if err != nil {
		t.Fatalf("RenderSpec: %v", err)
	}

	doc, err := Parse(rendered)
	if err != nil {
		t.Fatalf("Parse roundtrip: %v", err)
	}

	fm2, err := DecodeSpec(doc.RawFrontmatter)
	if err != nil {
		t.Fatalf("DecodeSpec roundtrip: %v", err)
	}

	if fm2.Title != fm.Title {
		t.Errorf("title roundtrip: %q != %q", fm2.Title, fm.Title)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	_, err := Parse("# Just a heading\n\nNo frontmatter here.")
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParseMissingClosingDelimiter(t *testing.T) {
	_, err := Parse("---\ntitle: test\nNo closing delimiter")
	if err == nil {
		t.Fatal("expected error for missing closing delimiter")
	}
}
