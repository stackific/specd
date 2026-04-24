package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestHandleIndexServesHTML verifies that the index handler returns HTML.
func TestHandleIndexServesHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	handleIndex(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %q", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "htmx") {
		t.Error("response should contain htmx script reference")
	}
	if !strings.Contains(body, "<title>specd</title>") {
		t.Error("response should contain specd title")
	}
}
