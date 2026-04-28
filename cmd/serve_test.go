package cmd

import (
	"database/sql"
	"fmt"
	"net"
	"path/filepath"
	"testing"
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

// TestLookupStartpageRoute exercises the validator used by the JSON settings
// endpoint (and /api/settings/default-route).
func TestLookupStartpageRoute(t *testing.T) {
	for _, c := range StartpageChoices {
		got, ok := lookupStartpageRoute(c.Route)
		if !ok || got != c.Route {
			t.Errorf("lookupStartpageRoute(%q) = (%q, %v), want (%q, true)", c.Route, got, ok, c.Route)
		}
	}
	for _, bad := range []string{"", "/evil", "/docs", "/specs/"} {
		if _, ok := lookupStartpageRoute(bad); ok {
			t.Errorf("lookupStartpageRoute(%q) should be false", bad)
		}
	}
}
