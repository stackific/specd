// Package web provides the embedded HTTP server for the specd web UI.
// It serves Go templates rendered with htmx support, plus static assets
// (CSS, JS, fonts, images) from embedded filesystems.
package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/stackific/specd/internal/workspace"
)

// mdHeading strips markdown heading prefixes.
var mdHeading = regexp.MustCompile(`(?m)^#{1,6}\s+`)

// mdLink matches markdown links [text](url) and keeps the text.
var mdLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

// mdListPrefix strips markdown list prefixes (-, *, +, 1.).
var mdListPrefix = regexp.MustCompile(`(?m)^[-*+]\s|^\d+\.\s`)

// mdBlockquote strips blockquote prefixes.
var mdBlockquote = regexp.MustCompile(`(?m)^>\s+`)

// Server holds the parsed templates, workspace reference, and HTTP handler.
type Server struct {
	w       *workspace.Workspace
	pages   map[string]*template.Template
	cssFile string // hashed CSS filename, e.g. "style.CKqEBrDf.css"
	handler http.Handler
}

// PageData is passed to every template render.
type PageData struct {
	Title   string
	CSSFile string
	Active  string // nav highlight key: "board", "specs", "kb", "search", "status", "trash"
	Error   string // flash error message from ?error= query param
	Data    any    // page-specific payload
}

// NewServer creates a web server from the given embedded filesystems.
// templateFS should contain templates/, distFS should contain styles/dist/,
// assetsFS should contain assets/.
func NewServer(w *workspace.Workspace, templateFS, distFS, assetsFS fs.FS) (*Server, error) {
	s := &Server{
		w:     w,
		pages: make(map[string]*template.Template),
	}

	// Discover the hashed CSS filename from distFS.
	cssFile, err := discoverCSSFile(distFS)
	if err != nil {
		return nil, fmt.Errorf("discover CSS file: %w", err)
	}
	s.cssFile = cssFile

	// Template functions available in all templates.
	funcMap := template.FuncMap{
		"isActive": func(active, name string) bool {
			return active == name
		},
		// add returns the sum of two integers (for template arithmetic).
		"add": func(a, b int) int { return a + b },
		// truncate returns the first n characters of s, appending "..." if truncated.
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		// stripTruncate strips markdown then truncates — for citation previews.
		"stripTruncate": func(s string, n int) string {
			out := mdHeading.ReplaceAllString(s, "")
			out = mdLink.ReplaceAllString(out, "$1")
			out = mdListPrefix.ReplaceAllString(out, "")
			out = mdBlockquote.ReplaceAllString(out, "")
			out = strings.ReplaceAll(out, "***", "")
			out = strings.ReplaceAll(out, "**", "")
			out = strings.ReplaceAll(out, "__", "")
			out = strings.ReplaceAll(out, "~~", "")
			out = strings.ReplaceAll(out, "`", "")
			out = strings.Join(strings.Fields(out), " ")
			out = strings.TrimSpace(out)
			if len(out) <= n {
				return out
			}
			return out[:n] + "..."
		},
		// toJSON marshals a value to JSON and returns it as template.JS
		// so html/template does not double-escape it inside <script> tags.
		"toJSON": func(v any) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("null")
			}
			return template.JS(b)
		},
	}

	// Parse shared templates (layout + partials).
	shared, err := template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layouts/*.html",
		"templates/partials/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse shared templates: %w", err)
	}

	// Parse each page template by cloning shared and adding the page.
	// This allows each page to define its own "content" block without conflict.
	pageEntries, err := fs.ReadDir(templateFS, "templates/pages")
	if err != nil {
		return nil, fmt.Errorf("read pages dir: %w", err)
	}
	for _, entry := range pageEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".html")
		clone, err := shared.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone for page %s: %w", name, err)
		}
		_, err = clone.ParseFS(templateFS, "templates/pages/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}
		s.pages[name] = clone
	}

	// Build the router.
	mux := http.NewServeMux()

	// /assets/ serves vendored JS (htmx, beer.js, app.js) and Vite-built
	// files (CSS bundle, font woff2s). CSS references fonts at /assets/*.woff2.
	jsFS, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return nil, fmt.Errorf("sub assets: %w", err)
	}
	viteAssetsFS, err := fs.Sub(distFS, "styles/dist/assets")
	if err != nil {
		return nil, fmt.Errorf("sub vite assets: %w", err)
	}
	mux.Handle("GET /assets/", http.StripPrefix("/assets/",
		mergedFileServer(jsFS, viteAssetsFS)))

	// Vendor CSS and fonts from styles/dist/vendor/.
	vendorFS, err := fs.Sub(distFS, "styles/dist/vendor")
	if err != nil {
		return nil, fmt.Errorf("sub vendor: %w", err)
	}
	mux.Handle("GET /vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))

	// Static files from styles/dist/ root (favicons, robots.txt, logos, etc.).
	staticFS, err := fs.Sub(distFS, "styles/dist")
	if err != nil {
		return nil, fmt.Errorf("sub static: %w", err)
	}
	staticServer := http.FileServer(http.FS(staticFS))
	for _, path := range []string{
		"/favicon.ico", "/favicon-32x32.png", "/favicon-16x16.png",
		"/apple-touch-icon.png", "/site.webmanifest", "/robots.txt",
		"/logo.svg", "/logo-dark.svg",
		"/android-chrome-192x192.png", "/android-chrome-512x512.png",
	} {
		mux.Handle("GET "+path, staticServer)
	}

	// Page routes.
	mux.HandleFunc("GET /{$}", s.handleBoard)
	mux.HandleFunc("GET /specs", s.handleSpecs)
	mux.HandleFunc("GET /specs/{id}", s.handleSpecDetail)
	mux.HandleFunc("GET /tasks/{id}", s.handleTaskDetail)
	mux.HandleFunc("GET /kb", s.handleKB)
	mux.HandleFunc("GET /kb/{id}", s.handleKBDetail)
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /trash", s.handleTrash)
	mux.HandleFunc("GET /rejected", s.handleRejected)

	// Board drag-and-drop endpoints.
	mux.HandleFunc("POST /api/board/move", s.handleDragMove)
	mux.HandleFunc("POST /api/board/reorder", s.handleDragReorder)

	// Spec mutations.
	mux.HandleFunc("POST /specs", s.handleCreateSpec)
	mux.HandleFunc("POST /specs/{id}/update", s.handleUpdateSpec)
	mux.HandleFunc("POST /specs/{id}/delete", s.handleDeleteSpec)

	// Task mutations.
	mux.HandleFunc("POST /tasks", s.handleCreateTask)
	mux.HandleFunc("POST /tasks/{id}/update", s.handleUpdateTask)
	mux.HandleFunc("POST /tasks/{id}/move", s.handleMoveTask)
	mux.HandleFunc("POST /tasks/{id}/reorder", s.handleReorderTask)
	mux.HandleFunc("POST /tasks/{id}/delete", s.handleDeleteTask)

	// Criteria mutations.
	mux.HandleFunc("POST /tasks/{id}/criteria", s.handleAddCriterion)
	mux.HandleFunc("POST /tasks/{id}/criteria/{pos}/check", s.handleCheckCriterion)
	mux.HandleFunc("POST /tasks/{id}/criteria/{pos}/uncheck", s.handleUncheckCriterion)
	mux.HandleFunc("POST /tasks/{id}/criteria/{pos}/remove", s.handleRemoveCriterion)

	// KB API endpoints (JSON, for client-side reader).
	mux.HandleFunc("GET /api/kb/{id}", s.handleAPIKBDoc)
	mux.HandleFunc("GET /api/kb/{id}/chunks", s.handleAPIKBChunks)
	mux.HandleFunc("GET /api/kb/{id}/chunk/{position}", s.handleAPIKBChunk)
	mux.HandleFunc("GET /api/kb/{id}/raw", s.handleAPIKBRaw)

	// KB mutations.
	mux.HandleFunc("POST /kb", s.handleAddKB)
	mux.HandleFunc("POST /kb/{id}/delete", s.handleDeleteKB)

	// Trash mutations.
	mux.HandleFunc("POST /trash/{id}/restore", s.handleRestoreTrash)
	mux.HandleFunc("POST /trash/purge", s.handlePurgeTrash)

	s.handler = mux
	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// redirectWithError redirects to the given path with an error query param.
// Used by form POST handlers to show errors inline instead of a blank page.
func redirectWithError(w http.ResponseWriter, r *http.Request, path, msg string) {
	dest := path + "?error=" + url.QueryEscape(msg)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", dest)
		return
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// ErrorData holds the status code and message for the error page.
type ErrorData struct {
	Code    int
	Message string
}

// renderError renders a styled error page using the base layout.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	data := PageData{
		Title:   fmt.Sprintf("%d", code),
		CSSFile: s.cssFile,
		Data:    ErrorData{Code: code, Message: msg},
	}
	tmpl, ok := s.pages["error"]
	if !ok {
		http.Error(w, msg, code)
		return
	}
	w.WriteHeader(code)
	if r.Header.Get("HX-Request") == "true" {
		tmpl.ExecuteTemplate(w, "content", data)
		return
	}
	tmpl.ExecuteTemplate(w, "base.html", data)
}

// renderFormPartial renders a named form template with HTTP 422 status.
// Used by dialog form handlers to return the form with validation errors
// for htmx to swap in place (dialog stays open, user input preserved).
func (s *Server) renderFormPartial(w http.ResponseWriter, pageName, formName string, data any) {
	tmpl, ok := s.pages[pageName]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := tmpl.ExecuteTemplate(w, formName, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderPage renders a full page or htmx partial depending on the request.
func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, pageName string, data PageData) {
	data.CSSFile = s.cssFile
	if e := r.URL.Query().Get("error"); e != "" {
		data.Error = e
	}

	tmpl, ok := s.pages[pageName]
	if !ok {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		// htmx request: render only the page content block.
		if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Full page render: execute the base layout.
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// discoverCSSFile finds the hashed CSS filename in styles/dist/assets/.
func discoverCSSFile(distFS fs.FS) (string, error) {
	entries, err := fs.ReadDir(distFS, "styles/dist/assets")
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".css") {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("no CSS file found in styles/dist/assets/")
}

// mergedFileServer tries the primary FS first, falls back to secondary.
// Used to serve both vendored JS and Vite-built assets (CSS, fonts) at /assets/.
func mergedFileServer(primary, secondary fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try primary (vendored JS) first.
		if _, err := fs.Stat(primary, r.URL.Path); err == nil {
			http.FileServer(http.FS(primary)).ServeHTTP(w, r)
			return
		}
		// Fall back to secondary (Vite-built CSS/fonts).
		http.FileServer(http.FS(secondary)).ServeHTTP(w, r)
	})
}
