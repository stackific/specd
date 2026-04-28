// api_set_linked_specs_test.go covers the PUT /api/specs/{id}/linked_specs
// flow: the small pure helpers (normalize / diff) plus the full HTTP
// handler. Reuses the setupProjectWithSpec + new-spec CLI seed helpers from
// the unlink test file so the project state on disk is realistic.
package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

// TestNormalizeLinkedSpecsInputDedupesAndUppercases verifies trim/upper/dedupe
// on the request body, including dropping empty strings.
func TestNormalizeLinkedSpecsInputDedupesAndUppercases(t *testing.T) {
	got, err := normalizeLinkedSpecsInput("SPEC-1", []string{"spec-2", " SPEC-3 ", "SPEC-2", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"SPEC-2", "SPEC-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// TestNormalizeLinkedSpecsInputRejectsSelfLink verifies a spec can't link to
// itself; this is enforced before the DB layer so a self-row never gets in.
func TestNormalizeLinkedSpecsInputRejectsSelfLink(t *testing.T) {
	if _, err := normalizeLinkedSpecsInput("SPEC-1", []string{"SPEC-1"}); err == nil {
		t.Fatal("expected error for self-link, got nil")
	}
}

// TestDiffLinkedSpecs verifies the add/remove diff is computed correctly:
// items in `next` not in `current` go to adds; items in `current` not in
// `next` go to removes; items in both are preserved untouched.
func TestDiffLinkedSpecs(t *testing.T) {
	add, remove := diffLinkedSpecs(
		[]string{"SPEC-2", "SPEC-3"},
		[]string{"SPEC-3", "SPEC-4"},
	)
	if !reflect.DeepEqual(add, []string{"SPEC-4"}) {
		t.Errorf("toAdd: got %v, want [SPEC-4]", add)
	}
	if !reflect.DeepEqual(remove, []string{"SPEC-2"}) {
		t.Errorf("toRemove: got %v, want [SPEC-2]", remove)
	}
}

// TestApiSetLinkedSpecsAddsNewLink verifies a clean-slate add: no prior
// links, body has one target, both directions land in spec_links.
func TestApiSetLinkedSpecsAddsNewLink(t *testing.T) {
	setupProjectWithLinkedSpecs(t) // SPEC-1 ↔ SPEC-2 pre-linked

	// Add SPEC-3 to the seed so we have a third spec.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{
		"new-spec", "--title", "Spec Three",
		"--summary", "Third",
		"--body", "Body three",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec: %v", err)
	}
	srv := newAPITestServer(t)

	body, _ := json.Marshal(apiSetLinkedSpecsRequest{
		LinkedSpecs: []string{"SPEC-2", "SPEC-3"}, // Add SPEC-3, keep SPEC-2.
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/specs/SPEC-1/linked_specs", bytes.NewReader(body))
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	// DB should hold both directions for SPEC-1 ↔ SPEC-2 and SPEC-1 ↔ SPEC-3.
	db, _ := sql.Open("sqlite", CacheDBFile)
	defer func() { _ = db.Close() }()
	rows := readSpecLinks(t, db, "SPEC-1")
	want := []string{"SPEC-2", "SPEC-3"}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("spec_links from SPEC-1: got %v, want %v", rows, want)
	}
}

// TestApiSetLinkedSpecsRemovesAbsent verifies that a target absent from the
// request is treated as a remove: the existing link to SPEC-2 is dropped.
func TestApiSetLinkedSpecsRemovesAbsent(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	body, _ := json.Marshal(apiSetLinkedSpecsRequest{LinkedSpecs: []string{}})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/specs/SPEC-1/linked_specs", bytes.NewReader(body))
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	db, _ := sql.Open("sqlite", CacheDBFile)
	defer func() { _ = db.Close() }()
	if rows := readSpecLinks(t, db, "SPEC-1"); len(rows) != 0 {
		t.Errorf("expected SPEC-1 to have no links, got %v", rows)
	}
}

// TestApiSetLinkedSpecsRejectsUnknown verifies a non-existent target spec
// is rejected with 400 and no DB writes occur.
func TestApiSetLinkedSpecsRejectsUnknown(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	body, _ := json.Marshal(apiSetLinkedSpecsRequest{LinkedSpecs: []string{"SPEC-999"}})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/specs/SPEC-1/linked_specs", bytes.NewReader(body))
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}

	// Original SPEC-1 ↔ SPEC-2 link should be untouched.
	db, _ := sql.Open("sqlite", CacheDBFile)
	defer func() { _ = db.Close() }()
	if rows := readSpecLinks(t, db, "SPEC-1"); !reflect.DeepEqual(rows, []string{"SPEC-2"}) {
		t.Errorf("expected SPEC-1 still linked to [SPEC-2], got %v", rows)
	}
}

// TestApiSetLinkedSpecsRejectsSelfLink verifies the validation guard fires
// before any DB mutation.
func TestApiSetLinkedSpecsRejectsSelfLink(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	body, _ := json.Marshal(apiSetLinkedSpecsRequest{LinkedSpecs: []string{"SPEC-1"}})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/specs/SPEC-1/linked_specs", bytes.NewReader(body))
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

// TestApiSetLinkedSpecsUnknownSubject verifies a missing source spec id
// returns 404; the body is irrelevant because the subject must exist before
// any mutation is considered.
func TestApiSetLinkedSpecsUnknownSubject(t *testing.T) {
	setupProjectWithLinkedSpecs(t)
	srv := newAPITestServer(t)

	body, _ := json.Marshal(apiSetLinkedSpecsRequest{LinkedSpecs: []string{"SPEC-2"}})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/specs/SPEC-9999/linked_specs", bytes.NewReader(body))
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

// readSpecLinks returns the to_spec ids for a given from_spec, sorted.
func readSpecLinks(t *testing.T, db *sql.DB, from string) []string {
	t.Helper()
	rows, err := db.Query(
		"SELECT to_spec FROM spec_links WHERE from_spec = ? ORDER BY to_spec", from,
	)
	if err != nil {
		t.Fatalf("query spec_links: %v", err)
	}
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, id)
	}
	return out
}
