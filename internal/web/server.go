// Package web provides the embedded HTTP server for the specd web UI.
// It serves Go templates rendered with htmx support, plus static assets
// (CSS, JS, fonts, images) from embedded filesystems.
package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

// Server holds the parsed templates and HTTP handler for the web UI.
type Server struct {
	pages   map[string]*template.Template
	cssFile string // hashed CSS filename, e.g. "style.CKqEBrDf.css"
	handler http.Handler
}

// PageData is passed to every template render.
type PageData struct {
	Title   string
	CSSFile string
}

// NewServer creates a web server from the given embedded filesystems.
// templateFS should contain templates/, distFS should contain styles/dist/,
// assetsFS should contain assets/.
func NewServer(templateFS, distFS, assetsFS fs.FS) (*Server, error) {
	s := &Server{pages: make(map[string]*template.Template)}

	// Discover the hashed CSS filename from distFS.
	cssFile, err := discoverCSSFile(distFS)
	if err != nil {
		return nil, fmt.Errorf("discover CSS file: %w", err)
	}
	s.cssFile = cssFile

	// Parse shared templates (layout + partials).
	shared, err := template.New("").ParseFS(templateFS,
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
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /about", s.handleAbout)

	s.handler = mux
	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// renderPage renders a full page or htmx partial depending on the request.
func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, pageName string, data PageData) {
	data.CSSFile = s.cssFile

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
