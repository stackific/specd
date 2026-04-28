package cmd

import (
	"strings"
	"testing"
)

func TestStripAcceptanceCriteriaRemovesSection(t *testing.T) {
	in := `## Overview

Some intro.

## Acceptance Criteria

- A
- B
- C

## Requirements

Continue.`
	out := StripAcceptanceCriteria(in)
	if strings.Contains(out, "Acceptance Criteria") {
		t.Errorf("expected acceptance heading removed, got:\n%s", out)
	}
	if strings.Contains(out, "- A") || strings.Contains(out, "- B") {
		t.Errorf("expected criteria bullets removed, got:\n%s", out)
	}
	if !strings.Contains(out, "## Overview") || !strings.Contains(out, "## Requirements") {
		t.Errorf("expected sibling H2s preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "Continue.") {
		t.Errorf("expected post-criteria content preserved, got:\n%s", out)
	}
}

func TestStripAcceptanceCriteriaCaseInsensitive(t *testing.T) {
	in := "## acceptance criteria\n\n- a\n\n## Notes\n\nKeep."
	out := StripAcceptanceCriteria(in)
	if strings.Contains(out, "acceptance criteria") {
		t.Errorf("case-insensitive match failed: %s", out)
	}
	if !strings.Contains(out, "## Notes") || !strings.Contains(out, "Keep.") {
		t.Errorf("expected following section preserved, got:\n%s", out)
	}
}

func TestStripAcceptanceCriteriaAtEndOfBody(t *testing.T) {
	in := "## Overview\n\nIntro.\n\n## Acceptance Criteria\n\n- only one\n"
	out := StripAcceptanceCriteria(in)
	if strings.Contains(out, "Acceptance Criteria") || strings.Contains(out, "only one") {
		t.Errorf("expected trailing criteria section removed, got:\n%s", out)
	}
	if !strings.Contains(out, "Intro.") {
		t.Errorf("expected preceding content kept, got:\n%s", out)
	}
}

func TestStripAcceptanceCriteriaStopsAtH1(t *testing.T) {
	// A bare `# Heading` should also terminate the skip.
	in := "## Acceptance Criteria\n\n- A\n\n# Top\n\nKeep."
	out := StripAcceptanceCriteria(in)
	if !strings.Contains(out, "# Top") || !strings.Contains(out, "Keep.") {
		t.Errorf("expected H1 to terminate skip, got:\n%s", out)
	}
}

func TestStripAcceptanceCriteriaNoMatch(t *testing.T) {
	in := "## Overview\n\nNo criteria heading here.\n"
	out := StripAcceptanceCriteria(in)
	if out == "" {
		t.Errorf("expected unchanged content, got empty string")
	}
	if !strings.Contains(out, "Overview") || !strings.Contains(out, "No criteria") {
		t.Errorf("expected content preserved verbatim, got:\n%s", out)
	}
}

func TestStripAcceptanceCriteriaCRLF(t *testing.T) {
	in := "## Acceptance Criteria\r\n\r\n- A\r\n\r\n## Next\r\n\r\nKeep.\r\n"
	out := StripAcceptanceCriteria(in)
	if strings.Contains(out, "Acceptance Criteria") {
		t.Errorf("CRLF input not handled: %s", out)
	}
	if !strings.Contains(out, "## Next") {
		t.Errorf("expected next H2 preserved, got:\n%s", out)
	}
}

func TestStripAcceptanceCriteriaEmpty(t *testing.T) {
	if got := StripAcceptanceCriteria(""); got != "" {
		t.Errorf("expected empty in/out, got %q", got)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	if got := RenderMarkdown(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRenderMarkdownConvertsHeadings(t *testing.T) {
	got := string(RenderMarkdown("## Overview\n\nHello."))
	if !strings.Contains(got, "<h2") || !strings.Contains(got, "Overview") {
		t.Errorf("expected h2 in output, got: %s", got)
	}
	if !strings.Contains(got, "<p>Hello.") {
		t.Errorf("expected paragraph, got: %s", got)
	}
}
