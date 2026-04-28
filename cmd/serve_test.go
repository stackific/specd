package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// TestFindAvailablePort verifies that findAvailablePort returns an open port.
func TestFindAvailablePort(t *testing.T) {
	port, err := findAvailablePort(18000)
	if err != nil {
		t.Fatalf("findAvailablePort: %v", err)
	}
	if port < 18000 {
		t.Errorf("expected port >= 18000, got %d", port)
	}

	// Verify we can actually listen on it.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		// Port might have been grabbed between check and listen — not a test failure.
		t.Skipf("port %d no longer available (race): %v", port, err)
	}
	_ = ln.Close()
}

// TestFindAvailablePortSkipsOccupied verifies that occupied ports are skipped.
func TestFindAvailablePortSkipsOccupied(t *testing.T) {
	// Occupy a port.
	ln, err := net.Listen("tcp", ":0") //nolint:gosec // test needs all-interface bind to match findAvailablePort
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	occupiedPort := ln.Addr().(*net.TCPAddr).Port

	// Start scanning from the occupied port.
	port, err := findAvailablePort(occupiedPort)
	if err != nil {
		t.Fatalf("findAvailablePort: %v", err)
	}
	if port == occupiedPort {
		t.Errorf("should have skipped occupied port %d", occupiedPort)
	}
	if port < occupiedPort {
		t.Errorf("expected port >= %d, got %d", occupiedPort, port)
	}
}

// TestHandleGetDefaultRoute verifies the API endpoint returns JSON.
func TestHandleGetDefaultRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/meta/default-route", http.NoBody)
	w := httptest.NewRecorder()

	handleGetDefaultRoute(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["default_route"] == "" {
		t.Error("expected non-empty default_route")
	}
}

// TestReadMetaReturnsValue verifies ReadMeta reads from the meta table.
func TestReadMetaReturnsValue(t *testing.T) {
	dir := t.TempDir()
	if err := InitDB(dir, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	val, err := ReadMeta(db, MetaDefaultRoute)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if val != DefaultRoute {
		t.Errorf("expected %q, got %q", DefaultRoute, val)
	}
}

// TestReadMetaMissingKey verifies ReadMeta returns an error for unknown keys.
func TestReadMetaMissingKey(t *testing.T) {
	dir := t.TempDir()
	if err := InitDB(dir, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	_, err = ReadMeta(db, "nonexistent_key")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

// TestInitDBSeedsDefaultRoute verifies that InitDB seeds default_route.
func TestInitDBSeedsDefaultRoute(t *testing.T) {
	dir := t.TempDir()
	if err := InitDB(dir, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var value string
	err = db.QueryRow("SELECT value FROM meta WHERE key = ?", MetaDefaultRoute).Scan(&value)
	if err != nil {
		t.Fatalf("querying default_route: %v", err)
	}
	if value != DefaultRoute {
		t.Errorf("expected %q, got %q", DefaultRoute, value)
	}
}

// setupHandlerProject initializes a fresh project in a temp dir, chdirs into
// it, and returns. Cleanup restores the original working directory.
func setupHandlerProject(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	if err := InitDB(tmp, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProjectConfig(tmp, &ProjectConfig{
		Dir:        "specd",
		SpecTypes:  []string{"business"},
		TaskStages: []string{"backlog", "done"},
	}); err != nil {
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

// TestWriteMetaUpsert verifies WriteMeta inserts and updates values.
func TestWriteMetaUpsert(t *testing.T) {
	dir := t.TempDir()
	if err := InitDB(dir, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := WriteMeta(db, MetaDefaultRoute, "/specs"); err != nil {
		t.Fatalf("first WriteMeta: %v", err)
	}
	v, err := ReadMeta(db, MetaDefaultRoute)
	if err != nil {
		t.Fatal(err)
	}
	if v != "/specs" {
		t.Errorf("expected /specs, got %q", v)
	}

	// Second write of the same key should overwrite, not error.
	if err := WriteMeta(db, MetaDefaultRoute, "/kb"); err != nil {
		t.Fatalf("second WriteMeta: %v", err)
	}
	v, err = ReadMeta(db, MetaDefaultRoute)
	if err != nil {
		t.Fatal(err)
	}
	if v != "/kb" {
		t.Errorf("expected /kb after upsert, got %q", v)
	}
}

// TestHandleSetDefaultRouteValid verifies the POST handler persists a valid choice.
func TestHandleSetDefaultRouteValid(t *testing.T) {
	setupHandlerProject(t)

	form := strings.NewReader("default_route=/specs")
	req := httptest.NewRequest(http.MethodPost, "/settings/default-route", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleSetDefaultRoute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if got := readDefaultRoute(); got != "/specs" {
		t.Errorf("expected /specs persisted, got %q", got)
	}
}

// TestHandleSetDefaultRouteInvalid rejects values outside StartpageChoices.
func TestHandleSetDefaultRouteInvalid(t *testing.T) {
	setupHandlerProject(t)

	form := strings.NewReader("default_route=/evil")
	req := httptest.NewRequest(http.MethodPost, "/settings/default-route", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handleSetDefaultRoute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if got := readDefaultRoute(); got != DefaultRoute {
		t.Errorf("expected unchanged default %q, got %q", DefaultRoute, got)
	}
}

// TestHandleSetDefaultRouteAllChoices ensures every advertised choice round-trips.
func TestHandleSetDefaultRouteAllChoices(t *testing.T) {
	setupHandlerProject(t)

	for _, c := range StartpageChoices {
		form := strings.NewReader("default_route=" + c.Route)
		req := httptest.NewRequest(http.MethodPost, "/settings/default-route", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handleSetDefaultRoute(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", c.Label, w.Code)
			continue
		}
		if got := readDefaultRoute(); got != c.Route {
			t.Errorf("%s: expected %q, got %q", c.Label, c.Route, got)
		}
	}
}

// TestIsValidStartpageRoute covers the validator helper.
func TestIsValidStartpageRoute(t *testing.T) {
	for _, c := range StartpageChoices {
		if !IsValidStartpageRoute(c.Route) {
			t.Errorf("expected %q to be valid", c.Route)
		}
	}
	for _, bad := range []string{"", "/", "/evil", "/docs", "/specs/"} {
		if IsValidStartpageRoute(bad) {
			t.Errorf("expected %q to be invalid", bad)
		}
	}
}

// testTemplateFS returns a minimal in-memory FS for template parsing tests.
func testTemplateFS() fstest.MapFS {
	return fstest.MapFS{
		"layouts/base.html": {Data: []byte(
			`{{define "base.html"}}<!DOCTYPE html><html><head>` +
				`<title>{{.Title}} — specd</title>` +
				`</head><body>` +
				`{{template "nav" .}}` +
				`<main>{{block "content" .}}{{end}}</main>` +
				`{{template "footer" .}}` +
				`</body></html>{{end}}`,
		)},
		"layouts/partial.html": {Data: []byte(
			`{{define "partial"}}<title>{{.Title}} — specd</title>` +
				`{{block "content" .}}{{end}}{{end}}`,
		)},
		"partials/nav.html":    {Data: []byte(`{{define "nav"}}<nav>{{if isActive .Active "docs"}}docs-active{{end}}</nav>{{end}}`)},
		"partials/footer.html": {Data: []byte(`{{define "footer"}}<footer>footer</footer>{{end}}`)},
		"pages/tutorial.html":  {Data: []byte(`{{define "content"}}<h1>Tutorial</h1>{{end}}`)},
		"pages/docs.html":      {Data: []byte(`{{define "content"}}<h1>Docs</h1>{{end}}`)},
	}
}

// TestParseTemplates verifies that templates are parsed correctly.
func TestParseTemplates(t *testing.T) {
	pages, err := parseTemplates(testTemplateFS())
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	if _, ok := pages["tutorial"]; !ok {
		t.Error("expected 'tutorial' page")
	}
	if _, ok := pages["docs"]; !ok {
		t.Error("expected 'docs' page")
	}
}

// TestRenderPageFullPage verifies that a non-htmx request renders the full page.
func TestRenderPageFullPage(t *testing.T) {
	pages, err := parseTemplates(testTemplateFS())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/tutorial", http.NoBody)
	w := httptest.NewRecorder()

	renderPage(w, req, pages, "tutorial", &PageData{Title: "Tutorial", Active: "docs"})

	body := w.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected full HTML document")
	}
	if !strings.Contains(body, "<h1>Tutorial</h1>") {
		t.Error("expected tutorial content")
	}
	if !strings.Contains(body, "docs-active") {
		t.Error("expected active nav for docs")
	}
	if !strings.Contains(body, "<footer>footer</footer>") {
		t.Error("expected footer")
	}
}

// TestRenderPageHtmxPartial verifies that an htmx request renders only the content block.
func TestRenderPageHtmxPartial(t *testing.T) {
	pages, err := parseTemplates(testTemplateFS())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/tutorial", http.NoBody)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	renderPage(w, req, pages, "tutorial", &PageData{Title: "Tutorial", Active: "docs"})

	body := w.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("htmx partial should not include full HTML document")
	}
	if !strings.Contains(body, "<title>Tutorial — specd</title>") {
		t.Error("expected title in partial for htmx to update document.title")
	}
	if !strings.Contains(body, "<h1>Tutorial</h1>") {
		t.Error("expected tutorial content in partial")
	}
	if strings.Contains(body, "<footer>") {
		t.Error("htmx partial should not include footer")
	}
}

// TestRenderPageNotFound verifies that a missing page returns 404.
func TestRenderPageNotFound(t *testing.T) {
	pages, err := parseTemplates(testTemplateFS())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	w := httptest.NewRecorder()

	renderPage(w, req, pages, "nonexistent", &PageData{Title: "Not Found"})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestStaticFileServing verifies that static assets are served correctly.
func TestStaticFileServing(t *testing.T) {
	assets := fstest.MapFS{
		"vendor/htmx.min.js": {Data: []byte("htmx-content")},
		"css/dist/app.css":   {Data: []byte("body{}")},
		"js/app.js":          {Data: []byte("console.log('app')")},
		"fonts/test.woff2":   {Data: []byte("font-data")},
		"images/favicon.ico": {Data: []byte("icon-data")},
		"images/logo.svg":    {Data: []byte("<svg></svg>")},
		"images/robots.txt":  {Data: []byte("User-agent: *")},
	}

	mux := http.NewServeMux()
	registerStaticRoutes(mux, assets)

	tests := []struct {
		path     string
		contains string
	}{
		{"/vendor/htmx.min.js", "htmx-content"},
		{"/css/app.css", "body{}"},
		{"/js/app.js", "console.log"},
		{"/fonts/test.woff2", "font-data"},
		{"/favicon.ico", "icon-data"},
		{"/logo.svg", "<svg>"},
		{"/robots.txt", "User-agent"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			if !strings.Contains(w.Body.String(), tt.contains) {
				t.Errorf("expected body to contain %q", tt.contains)
			}
		})
	}
}

// TestComputeFileHash verifies content hash generation.
func TestComputeFileHash(t *testing.T) {
	fs := fstest.MapFS{
		"css/dist/app.css": {Data: []byte("body{}")},
	}

	hash := computeFileHash(fs, "css/dist/app.css")
	if len(hash) != 8 {
		t.Errorf("expected 8-char hash, got %q (len %d)", hash, len(hash))
	}

	// Same content produces same hash.
	hash2 := computeFileHash(fs, "css/dist/app.css")
	if hash != hash2 {
		t.Errorf("expected deterministic hash, got %q and %q", hash, hash2)
	}

	// Missing file returns empty string.
	empty := computeFileHash(fs, "nonexistent.css")
	if empty != "" {
		t.Errorf("expected empty string for missing file, got %q", empty)
	}
}

// TestLiveReloadEndpoint verifies the SSE endpoint responds correctly.
func TestLiveReloadEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/dev/livereload", http.NoBody)
	w := httptest.NewRecorder()

	// handleLiveReload blocks on context, so cancel it immediately.
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	cancel() // unblock immediately

	handleLiveReload(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
}
