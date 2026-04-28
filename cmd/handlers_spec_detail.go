// handlers_spec_detail.go implements the GET /specs/{id} page handler.
// Renders a single spec with its acceptance criteria, child tasks, and
// linked specs, reusing LoadSpecDetail() so the web page and the get-spec
// CLI share one data path.
package cmd

import (
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

// SpecDetailPageData is the view model passed to the spec detail template.
type SpecDetailPageData struct {
	Spec *GetSpecResponse
}

// makeSpecDetailHandler returns an http.HandlerFunc for /specs/{id}. The {id}
// path value is read via r.PathValue (Go 1.22+ ServeMux pattern).
func makeSpecDetailHandler(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}

		db, _, err := OpenProjectDB()
		if err != nil {
			slog.Error("spec detail: db", "error", err)
			http.Error(w, "database unavailable", http.StatusInternalServerError)
			return
		}
		defer func() { _ = db.Close() }()

		spec, err := LoadSpecDetail(db, id)
		if err != nil {
			// LoadSpecDetail wraps fmt.Errorf("spec %s not found: %w", ...) on
			// the missing-row case; everything else is a 500. Until we
			// introduce a typed sentinel, match on the wrapped message.
			if strings.Contains(err.Error(), "not found") {
				http.NotFound(w, r)
				return
			}
			slog.Error("spec detail: load", "error", err)
			http.Error(w, "failed to load spec", http.StatusInternalServerError)
			return
		}

		renderPage(w, r, freshPages(), "spec_detail", &PageData{
			Title:   spec.ID + " — " + spec.Title,
			Active:  "specs",
			DevMode: devMode,
			CSSHash: cssHash,
			JSHash:  jsHash,
			Data:    SpecDetailPageData{Spec: spec},
		})
	}
}
