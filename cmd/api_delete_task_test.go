package cmd

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// newAPITestServer builds an httptest.Server with the full RegisterAPI mux
// installed. Tests that exercise HTTP endpoints share this so they stay
// aligned with the actual production wiring (instead of hand-calling the
// handler functions, which would skip path-value parsing).
func newAPITestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	RegisterAPI(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestAPIDeleteTaskHappyPath exercises DELETE /api/tasks/{id} end-to-end
// against a real seeded project: returns the deleted record JSON, removes
// the row from the DB, and removes the markdown file from disk.
func TestAPIDeleteTaskHappyPath(t *testing.T) {
	setupProjectWithTaskForDelete(t)
	srv := newAPITestServer(t)

	// Capture the file path before deletion so we can verify it's gone.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	_ = db.QueryRow("SELECT path FROM tasks WHERE id = 'TASK-1'").Scan(&path)
	_ = db.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tasks/TASK-1", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var body DeleteTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != "TASK-1" || body.SpecID != "SPEC-1" || !body.Deleted {
		t.Errorf("unexpected response: %+v", body)
	}

	// DB row gone.
	db, _ = sql.Open("sqlite", CacheDBFile)
	defer func() { _ = db.Close() }()
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-1'").Scan(&n)
	if n != 0 {
		t.Errorf("DB row remains after delete (count=%d)", n)
	}

	// File gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should be removed: stat err=%v", err)
	}
}

// TestAPIDeleteTaskCaseInsensitive verifies the path value is normalized to
// upper case (TASK-N), matching the rest of the API.
func TestAPIDeleteTaskCaseInsensitive(t *testing.T) {
	setupProjectWithTaskForDelete(t)
	srv := newAPITestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tasks/task-1", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
}

// TestAPIDeleteTaskNotFound verifies a missing ID returns 404 with a JSON
// error body, not a generic 500.
func TestAPIDeleteTaskNotFound(t *testing.T) {
	setupProjectWithTaskForDelete(t)
	srv := newAPITestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tasks/TASK-999", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != errCodeNotFound {
		t.Errorf("error code: got %q, want %q", body["code"], errCodeNotFound)
	}
}

// TestAPIDeleteTaskRejectsNonTaskID verifies the path-prefix guard rejects
// IDs that don't start with TASK- (e.g. SPEC-1, KB-1) before touching the DB.
func TestAPIDeleteTaskRejectsNonTaskID(t *testing.T) {
	setupProjectWithTaskForDelete(t)
	srv := newAPITestServer(t)

	for _, badID := range []string{"SPEC-1", "KB-1", "FOO"} {
		req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tasks/"+badID, http.NoBody)
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("DELETE %s: %v", badID, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", badID, resp.StatusCode)
		}
	}

	// SPEC-1 should still exist in the DB after the rejected calls above.
	db, _ := sql.Open("sqlite", CacheDBFile)
	defer func() { _ = db.Close() }()
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-1'").Scan(&n)
	if n != 1 {
		t.Errorf("SPEC-1 should not be touched; got count=%d", n)
	}
}
