package cmd

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// setupProjectWithLinkedSpecs creates a project, two specs, and links them
// bidirectionally. Returns nothing — tests rely on the chdir + DB state set
// by setupProjectWithSpec + the explicit `update-spec --link-specs` calls.
func setupProjectWithLinkedSpecs(t *testing.T) {
	t.Helper()
	_ = setupProjectWithSpec(t)

	resetNewSpecFlags()
	rootCmd.SetArgs([]string{
		"new-spec", "--title", "Spec Two",
		"--summary", "Second spec for link tests",
		"--body", "Body two",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec: %v", err)
	}

	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-1", "--link-specs", "SPEC-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("link: %v", err)
	}
}

// TestAPIUnlinkSpecHappyPath verifies DELETE /api/specs/{id}/linked_specs/{linkedId}
// removes the link in BOTH directions and returns the updated source spec
// with empty linked_specs_refs.
func TestAPIUnlinkSpecHappyPath(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	// Sanity: SPEC-1 ↔ SPEC-2 should be linked at this point.
	db, _ := sql.Open("sqlite", CacheDBFile)
	var n int
	_ = db.QueryRow(
		"SELECT COUNT(*) FROM spec_links WHERE (from_spec='SPEC-1' AND to_spec='SPEC-2') OR (from_spec='SPEC-2' AND to_spec='SPEC-1')",
	).Scan(&n)
	if n != 2 {
		t.Fatalf("expected 2 spec_links rows before unlink, got %d", n)
	}
	_ = db.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/specs/SPEC-1/linked_specs/SPEC-2", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if id, _ := body["id"].(string); id != "SPEC-1" {
		t.Errorf("response id: got %q, want SPEC-1", id)
	}
	refs, ok := body["linked_specs_refs"].([]any)
	if !ok {
		t.Fatalf("linked_specs_refs missing or wrong shape")
	}
	if len(refs) != 0 {
		t.Errorf("linked_specs_refs should be empty after unlink, got %d", len(refs))
	}

	// Both directions removed in DB.
	db, _ = sql.Open("sqlite", CacheDBFile)
	defer func() { _ = db.Close() }()
	_ = db.QueryRow(
		"SELECT COUNT(*) FROM spec_links WHERE (from_spec='SPEC-1' AND to_spec='SPEC-2') OR (from_spec='SPEC-2' AND to_spec='SPEC-1')",
	).Scan(&n)
	if n != 0 {
		t.Errorf("spec_links rows remain after unlink: %d", n)
	}

	// Both spec markdown files should have been rewritten — verify by
	// reading them back and asserting the cross-reference is gone.
	for _, id := range []string{"spec-1", "spec-2"} {
		path := filepath.Join("specd", "specs", id, "spec.md")
		content, err := os.ReadFile(path) //nolint:gosec // test path
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		other := strings.ToUpper(strings.ReplaceAll(id, "spec-", "SPEC-"))
		switch other {
		case "SPEC-1":
			other = "SPEC-2"
		case "SPEC-2":
			other = "SPEC-1"
		}
		if strings.Contains(string(content), other) {
			t.Errorf("%s still references %s", path, other)
		}
	}
}

// TestAPIUnlinkSpecNotFound returns 404 when the source spec doesn't exist.
func TestAPIUnlinkSpecNotFound(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/specs/SPEC-99/linked_specs/SPEC-2", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got %d, want 404", resp.StatusCode)
	}
}

// TestAPIUnlinkSpecRejectsSelf protects against an oddly-formed request
// trying to "unlink" a spec from itself, which the DB would happily no-op
// but is a programmer error worth surfacing.
func TestAPIUnlinkSpecRejectsSelf(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/specs/SPEC-1/linked_specs/SPEC-1", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got %d, want 400", resp.StatusCode)
	}
}
