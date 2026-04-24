package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
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

// TestSPAHandlerServesFiles verifies the SPA handler serves existing files.
func TestSPAHandlerServesFiles(t *testing.T) {
	fs := fstest.MapFS{
		"index.html":           {Data: []byte("<html><title>specd</title></html>")},
		"assets/index-abc.css": {Data: []byte("body{}")},
	}

	handler := spaHandler(fs)

	// Request an existing asset file.
	req := httptest.NewRequest(http.MethodGet, "/assets/index-abc.css", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "body{}") {
		t.Error("expected CSS content")
	}
}

// TestSPAHandlerFallsBackToIndex verifies unknown paths serve index.html.
func TestSPAHandlerFallsBackToIndex(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": {Data: []byte("<html><title>specd</title></html>")},
	}

	handler := spaHandler(fs)

	// Request a non-existent path (client-side route).
	req := httptest.NewRequest(http.MethodGet, "/tutorial", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<title>specd</title>") {
		t.Error("expected index.html content as SPA fallback")
	}
}

// TestSPAHandlerRootRedirects verifies "/" redirects to the default route.
func TestSPAHandlerRootRedirects(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": {Data: []byte("<html></html>")},
	}

	handler := spaHandler(fs)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 redirect, got %d", w.Code)
	}

	loc := w.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header on redirect")
	}
	// Without an initialized DB, should fall back to DefaultRoute.
	if loc != DefaultRoute {
		t.Errorf("expected redirect to %q, got %q", DefaultRoute, loc)
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
