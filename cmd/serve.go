// serve.go implements `specd serve`. Starts an HTTP server that renders
// Go templates with htmx support and serves static assets. Scans for
// available ports starting from DefaultServePort (8000), prints progress,
// and opens the browser on success.
package cmd

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	// Ensure correct MIME types for embedded static assets.
	// Some OS MIME databases lack entries for .css, .js, .woff2.
	_ = mime.AddExtensionType(".css", "text/css")
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".woff2", "font/woff2")
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// serveCmd implements `specd serve`.
// It starts an HTTP server with the Web UI, scanning for an available port.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the specd Web UI",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().Int("port", DefaultServePort, "starting port number")
	serveCmd.Flags().Bool("no-open", false, "do not open the browser on start")
	serveCmd.Flags().Bool("dev", false, "enable dev mode (live reload, no template caching)")
	serveCmd.Flags().String("dir", "", "directory to serve from (defaults to current directory)")
	rootCmd.AddCommand(serveCmd)
}

func runServe(c *cobra.Command, _ []string) error {
	dir, _ := c.Flags().GetString("dir")
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			return fmt.Errorf("changing to directory %q: %w", dir, err)
		}
	}

	startPort, _ := c.Flags().GetInt("port")
	devMode, _ := c.Flags().GetBool("dev")

	// Parse templates from the embedded filesystem.
	pages, err := parseTemplates(templateFS)
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	// Compute content hashes for cache busting.
	cssHash := computeFileHash(staticFS, "css/dist/app.css")
	jsHash := computeFileHash(staticFS, "js/app.js")

	// Find an available port starting from startPort.
	port, err := findAvailablePort(startPort)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	// Set up routes.
	mux := http.NewServeMux()

	// API endpoints.
	mux.HandleFunc("GET /api/meta/default-route", handleGetDefaultRoute)

	// Settings actions (server-rendered HTML responses, not REST).
	mux.HandleFunc("POST /settings/default-route", handleSetDefaultRoute)

	// Kanban board endpoints — partial-rendering responses for htmx swaps.
	freshPages := makeFreshPages(devMode, pages)
	mux.HandleFunc("GET /api/tasks/board", func(w http.ResponseWriter, r *http.Request) {
		handleGetTasksBoard(freshPages())(w, r)
	})
	mux.HandleFunc("POST /api/tasks/{id}/criteria/{position}/toggle", makeTaskCriteriaToggleHandler(freshPages))
	mux.HandleFunc("POST /api/tasks/move", func(w http.ResponseWriter, r *http.Request) {
		handleMoveTask(freshPages())(w, r)
	})

	// Dev-mode live reload SSE endpoint.
	if devMode {
		mux.HandleFunc("GET /api/dev/livereload", handleLiveReload)
	}

	// Page routes — each renders a Go template with htmx partial support.
	page := makePageHandler(freshPages, devMode, cssHash, jsHash)
	mux.HandleFunc("GET /docs/tutorial", page("tutorial", "Tutorial", "docs"))
	mux.HandleFunc("GET /docs", page("docs", "Documentation", "docs"))
	mux.HandleFunc("GET /tasks", makeTasksHandler(freshPages, devMode, cssHash, jsHash))
	mux.HandleFunc("GET /tasks/{id}", makeTaskDetailHandler(freshPages, devMode, cssHash, jsHash))
	mux.HandleFunc("GET /specs", makeSpecsHandler(freshPages, devMode, cssHash, jsHash))
	mux.HandleFunc("GET /specs/{id}", makeSpecDetailHandler(freshPages, devMode, cssHash, jsHash))
	mux.HandleFunc("GET /kb", page("kb", "Knowledge Base", "kb"))
	mux.HandleFunc("GET /kb/{id}", makeKBDetailHandler(freshPages, devMode, cssHash, jsHash))
	mux.HandleFunc("GET /search", makeSearchHandler(freshPages, devMode, cssHash, jsHash))
	mux.HandleFunc("GET /settings", handleSettingsPage(freshPages, devMode, cssHash, jsHash))

	// Root redirect to default route.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		route := readDefaultRoute()
		http.Redirect(w, r, route, http.StatusTemporaryRedirect)
	})

	// Static assets — served from the embedded static filesystem.
	registerStaticRoutes(mux, staticFS)

	slog.Info("serve", "port", port, "dev", devMode)

	noOpen, _ := c.Flags().GetBool("no-open")
	if !noOpen {
		fmt.Printf("specd Web UI running at %s\n", url)
		if err := openBrowser(url); err != nil {
			slog.Debug("could not open browser", "error", err)
		}
	}

	// Start the server (blocks until interrupted).
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // local dev server, no TLS needed
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// makePageHandler returns a builder that produces page http.HandlerFuncs for
// generic pages with no per-page data. Each call captures (name, title, active)
// for one route.
func makePageHandler(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) func(name, title, active string) http.HandlerFunc {
	return func(name, title, active string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			renderPage(w, r, freshPages(), name, &PageData{
				Title:   title,
				Active:  active,
				DevMode: devMode,
				CSSHash: cssHash,
				JSHash:  jsHash,
			})
		}
	}
}

// makeFreshPages returns a closure that yields the templates map. In dev mode
// templates are re-parsed on every call so changes are picked up live; in
// production it returns the same cached map every call.
func makeFreshPages(devMode bool, cached map[string]*template.Template) func() map[string]*template.Template {
	if !devMode {
		return func() map[string]*template.Template { return cached }
	}
	return func() map[string]*template.Template {
		p, err := parseTemplates(templateFS)
		if err != nil {
			return cached
		}
		return p
	}
}

// registerStaticRoutes sets up file serving routes for embedded static assets.
func registerStaticRoutes(mux *http.ServeMux, assets fs.FS) {
	// Subdirectories served as-is.
	serveSubDir(mux, assets, "/vendor/", "vendor")
	serveSubDir(mux, assets, "/css/", "css/dist")
	serveSubDir(mux, assets, "/js/", "js")
	serveSubDir(mux, assets, "/fonts/", "fonts")

	// Root-level static files (favicons, logos, manifest, robots.txt).
	rootFiles := []string{
		"favicon.ico", "favicon-16x16.png", "favicon-32x32.png",
		"apple-touch-icon.png", "android-chrome-192x192.png", "android-chrome-512x512.png",
		"logo.svg", "logo-dark.svg", "site.webmanifest", "robots.txt",
	}
	for _, name := range rootFiles {
		path := name // capture for closure
		mux.HandleFunc("GET /"+path, func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = "/" + path
			http.FileServer(http.FS(mustSub(assets, "images"))).ServeHTTP(w, r)
		})
	}
}

// serveSubDir registers a handler that serves files from a subdirectory of
// the static filesystem under the given URL prefix.
func serveSubDir(mux *http.ServeMux, assets fs.FS, prefix, dir string) {
	sub := mustSub(assets, dir)
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.FS(sub))))
}

// mustSub wraps fs.Sub and panics on error (used for embedded FS paths that
// are guaranteed to exist at compile time).
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(fmt.Sprintf("static sub-filesystem %q: %v", dir, err))
	}
	return sub
}

// handleLiveReload serves an SSE stream for dev-mode live reload.
// The client connects and waits; when the server restarts (via Air),
// the connection drops and the client-side script polls until it
// reconnects, then reloads the page.
func handleLiveReload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Flush headers immediately so the browser knows the connection is alive.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Block until the client disconnects or the server shuts down.
	<-r.Context().Done()
}

// computeFileHash reads a file from the given filesystem and returns the
// first 8 characters of its SHA-256 hash for cache busting.
// Returns an empty string if the file cannot be read.
func computeFileHash(assets fs.FS, path string) string {
	data, err := fs.ReadFile(assets, path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])[:8]
}

// readDefaultRoute queries the database for the configured default route.
// Returns DefaultRoute on any error (DB not initialized, missing key, etc.).
func readDefaultRoute() string {
	db, _, err := OpenProjectDB()
	if err != nil {
		return DefaultRoute
	}
	defer func() { _ = db.Close() }()

	var route string
	err = db.QueryRow("SELECT value FROM meta WHERE key = ?", MetaDefaultRoute).Scan(&route)
	if err != nil {
		return DefaultRoute
	}
	return route
}

// SettingsData is passed to the settings page template.
type SettingsData struct {
	CurrentRoute     string
	StartpageChoices []StartpageChoice
}

// handleSettingsPage renders /settings with the current default route and
// the available startpage choices.
func handleSettingsPage(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, r, freshPages(), "settings", &PageData{
			Title:   "Settings",
			Active:  "settings",
			DevMode: devMode,
			CSSHash: cssHash,
			JSHash:  jsHash,
			Data: SettingsData{
				CurrentRoute:     readDefaultRoute(),
				StartpageChoices: StartpageChoices,
			},
		})
	}
}

// handleSetDefaultRoute updates meta.default_route from a form POST submitted
// by the settings page. Validates the value against StartpageChoices and
// returns a small HTML fragment for htmx to swap into a status element.
func handleSetDefaultRoute(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxSettingsFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	// Resolve the submitted value to a canonical route from StartpageChoices
	// so all subsequent uses (DB write, log) operate on a value from a fixed
	// allowlist, not raw user input.
	canonical, ok := lookupStartpageRoute(r.PostForm.Get("default_route"))
	if !ok {
		http.Error(w, "invalid default_route", http.StatusBadRequest)
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}
	defer func() { _ = db.Close() }()

	if err := WriteMeta(db, MetaDefaultRoute, canonical); err != nil {
		slog.Error("write default_route", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}

	slog.Info("settings.default_route", "route", canonical)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<span class="primary-text">Saved</span>`)
}

// lookupStartpageRoute returns the canonical route from StartpageChoices
// matching the given input. The second return reports whether a match was
// found.
func lookupStartpageRoute(route string) (string, bool) {
	for _, c := range StartpageChoices {
		if c.Route == route {
			return c.Route, true
		}
	}
	return "", false
}

// handleGetDefaultRoute returns the configured default route as JSON.
func handleGetDefaultRoute(w http.ResponseWriter, _ *http.Request) {
	route := readDefaultRoute()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"default_route": route})
}

// ReadMeta reads a single value from the meta table by key.
// Returns sql.ErrNoRows if the key does not exist.
func ReadMeta(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("reading meta %s: %w", key, err)
	}
	return value, nil
}

// WriteMeta upserts a key/value pair in the meta table.
func WriteMeta(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("writing meta %s: %w", key, err)
	}
	return nil
}

// findAvailablePort tries ports starting from startPort, returning the first
// available one. Prints progress to the terminal as it scans.
func findAvailablePort(startPort int) (int, error) {
	for i := 0; i < MaxPortAttempts; i++ {
		port := startPort + i
		ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			fmt.Printf("Port %d in use, trying %d...\n", port, port+1)
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port found in range %d–%d", startPort, startPort+MaxPortAttempts-1)
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start() //nolint:gosec // url is constructed internally, not from user input
	case "linux":
		return exec.Command("xdg-open", rawURL).Start() //nolint:gosec // url is constructed internally
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start() //nolint:gosec // url is constructed internally
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
