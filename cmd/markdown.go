// markdown.go renders markdown source to safe HTML for the Web UI. Used by
// the spec/task/KB detail templates via the `markdown` template func.
package cmd

import (
	"bytes"
	"html/template"
	"log/slog"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// markdownRenderer is built once; goldmark's renderer is safe for concurrent
// use and reusing it avoids per-request allocation of the parser/render trees.
var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM, // tables, strikethrough, task lists, autolinks
		extension.DefinitionList,
		extension.Footnote,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		// HardWraps keeps single newlines as line breaks, matching how spec
		// authors usually write markdown in editors.
		html.WithHardWraps(),
		// Default goldmark escapes raw <script>/<iframe>; do NOT enable
		// WithUnsafe — spec bodies are user content and can contain anything.
	),
)

// RenderMarkdown converts CommonMark source to sanitized HTML for embedding
// in a template. Errors fall back to the escaped source so the page still
// renders (and the failure is logged).
func RenderMarkdown(src string) template.HTML {
	if src == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(src), &buf); err != nil {
		slog.Error("markdown render", "error", err)
		// Escape the source so we don't accidentally inject raw HTML on the
		// failure path. template.HTMLEscapeString returns string; wrapping in
		// HTML is safe because the value is escaped.
		return template.HTML(template.HTMLEscapeString(src)) //nolint:gosec // input is escaped via HTMLEscapeString
	}
	return template.HTML(buf.String()) //nolint:gosec // goldmark default escapes raw HTML; WithUnsafe is not set above
}

// StripAcceptanceCriteria removes the `## Acceptance Criteria` section from a
// markdown body. The section runs from its H2 line up to (but not including)
// the next H1/H2 heading or end of file. Used to avoid double-rendering the
// criteria, which the detail page already shows as a structured table from
// the parsed claims. Match is case-insensitive on the heading text.
func StripAcceptanceCriteria(body string) string {
	if body == "" {
		return body
	}
	// Normalise CRLF so heading detection is consistent on Windows-authored
	// files.
	normalised := strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(normalised, "\n")

	out := make([]string, 0, len(lines))
	skip := false
	for _, ln := range lines {
		if isAcceptanceH2(ln) {
			skip = true
			continue
		}
		if skip && isH1OrH2(ln) {
			// Reached the next top-level section — stop skipping and keep
			// this heading.
			skip = false
		}
		if !skip {
			out = append(out, ln)
		}
	}
	// Trim leading/trailing blank lines that the strip may have left behind.
	return strings.Trim(strings.Join(out, "\n"), "\n")
}

// isAcceptanceH2 reports whether ln is `## Acceptance Criteria` (any case,
// any trailing whitespace, optional leading spaces).
func isAcceptanceH2(ln string) bool {
	t := strings.TrimSpace(ln)
	if !strings.HasPrefix(t, "## ") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(t[3:]), "acceptance criteria")
}

// isH1OrH2 reports whether ln starts a level-1 or level-2 ATX heading.
func isH1OrH2(ln string) bool {
	t := strings.TrimSpace(ln)
	return strings.HasPrefix(t, "# ") || strings.HasPrefix(t, "## ")
}
