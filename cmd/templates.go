package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// PageData holds the data passed to every page template.
type PageData struct {
	Title   string // Page title (appears in <title>)
	Active  string // Active nav section (e.g. "docs")
	DevMode bool   // Whether dev mode is enabled (injects livereload script)
	CSSHash string // Content hash for CSS cache busting (e.g. "a1b2c3d4")
	JSHash  string // Content hash for JS cache busting
	Data    any    // Page-specific payload
}

// templateFuncMap provides helpers available in all templates.
var templateFuncMap = template.FuncMap{
	"isActive": func(active, section string) bool {
		return active == section
	},
	"searchResultHref":        searchResultHref,
	"fromSlug":                FromSlug,
	"markdown":                RenderMarkdown,
	"stripAcceptanceCriteria": StripAcceptanceCriteria,
}

// searchResultHref returns the page URL for a search result of the given kind
// and ID. Each kind has its own detail page at /<kind>/{id}.
func searchResultHref(kind, id string) string {
	switch kind {
	case KindSpec:
		return "/specs/" + id
	case KindTask:
		return "/tasks/" + id
	case KindKB:
		return "/kb/" + id
	default:
		return "/search"
	}
}

// parseTemplates parses all templates from the given filesystem and returns
// a map of page name → ready-to-execute template. Each page template includes
// the shared layout and partials so it can render a full page.
func parseTemplates(fsys fs.FS) (map[string]*template.Template, error) {
	// Parse shared templates (layouts + partials) as the base set.
	shared, err := template.New("").Funcs(templateFuncMap).ParseFS(fsys,
		"layouts/*.html",
		"partials/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing shared templates: %w", err)
	}

	// Find all page templates.
	pageFiles, err := fs.Glob(fsys, "pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("globbing page templates: %w", err)
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for _, pf := range pageFiles {
		name := strings.TrimSuffix(filepath.Base(pf), ".html")

		// Clone the shared templates so each page gets its own copy.
		clone, err := shared.Clone()
		if err != nil {
			return nil, fmt.Errorf("cloning shared templates for %s: %w", name, err)
		}

		// Parse the page file into the cloned set.
		_, err = clone.ParseFS(fsys, pf)
		if err != nil {
			return nil, fmt.Errorf("parsing page %s: %w", name, err)
		}

		pages[name] = clone
	}

	return pages, nil
}

// renderPage renders a page template to the response writer. If the request
// has the HX-Request header (htmx navigation), only the "content" block is
// rendered for a partial swap. Otherwise the full page via "base.html" is
// rendered.
//
// History restores (browser back/forward when htmx's history cache misses)
// arrive with HX-Request: true AND HX-History-Restore-Request: true. They
// need the full page back, otherwise the nav rail and other shell elements
// disappear from the restored DOM.
func renderPage(w http.ResponseWriter, r *http.Request, pages map[string]*template.Template, name string, data *PageData) {
	tmpl, ok := pages[name]
	if !ok {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	// Choose which template to execute: full page or htmx partial.
	// "partial" wraps the content block with a <title> tag so htmx
	// can update document.title on navigation.
	target := "base.html"
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-History-Restore-Request") != "true" {
		target = "partial"
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, target, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
