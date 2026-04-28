// api_depends_on_test.go covers the PUT /api/tasks/{id}/depends_on flow:
// the small pure helpers (normalize / verify / replace) plus the full HTTP
// handler. The test seeds a tiny project (three tasks under one spec) using
// the same setupBoardProject helper used by the board tests, then exercises
// the handler with httptest so the assertions cover validation, the DB
// transaction, and the JSON response shape.
package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"testing"
)

// TestNormalizeDependsOnInputDedupesAndUppercases verifies the pure helper:
// trimming whitespace, deduping, uppercasing, and rejecting invalid input.
func TestNormalizeDependsOnInputDedupesAndUppercases(t *testing.T) {
	got, err := normalizeDependsOnInput("TASK-1", []string{"task-2", " TASK-3 ", "TASK-2", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"TASK-2", "TASK-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// TestNormalizeDependsOnInputRejectsSelfDependency verifies that a task can
// never depend on itself; this is enforced before the DB layer because the
// schema's primary key wouldn't catch a self-reference.
func TestNormalizeDependsOnInputRejectsSelfDependency(t *testing.T) {
	if _, err := normalizeDependsOnInput("TASK-1", []string{"TASK-1"}); err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}
}

// TestNormalizeDependsOnInputRejectsBadPrefix verifies that any id without
// the TASK- prefix is rejected at the boundary.
func TestNormalizeDependsOnInputRejectsBadPrefix(t *testing.T) {
	if _, err := normalizeDependsOnInput("TASK-1", []string{"SPEC-1"}); err == nil {
		t.Fatal("expected error for non-TASK id, got nil")
	}
}

// TestApiSetTaskDependsOnReplacesSet covers the happy path: posting a new
// set replaces whatever was previously stored. We pre-seed two dependencies
// then PUT a different set and check the rows.
func TestApiSetTaskDependsOnReplacesSet(t *testing.T) {
	db, projDir, _ := setupBoardProject(t)

	// Pre-seed: TASK-1 depends on TASK-2 only.
	if _, err := db.Exec(
		"INSERT INTO task_dependencies(blocker_task, blocked_task) VALUES ('TASK-2', 'TASK-1')",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	chdir(t, projDir)

	body, _ := json.Marshal(apiSetTaskDependsOnRequest{DependsOn: []string{"TASK-3"}})
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/TASK-1/depends_on", bytes.NewReader(body))
	req.SetPathValue("id", "TASK-1")
	rec := httptest.NewRecorder()
	apiSetTaskDependsOnHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	got := readDeps(t, db, "TASK-1")
	want := []string{"TASK-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("blockers after PUT = %v, want %v", got, want)
	}

	// Response should be a TaskDetailResponse with the new depends_on_refs.
	var resp apiTaskDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := refIDs(resp.DependsOnRefs), []string{"TASK-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("response depends_on_refs = %v, want %v", got, want)
	}
}

// TestApiSetTaskDependsOnEmptyClearsAll verifies that PUT with an empty
// array clears every blocker — the same code path the UI uses for the X-icon
// "remove" action when the user removes the only remaining dependency.
func TestApiSetTaskDependsOnEmptyClearsAll(t *testing.T) {
	db, projDir, _ := setupBoardProject(t)
	if _, err := db.Exec(
		"INSERT INTO task_dependencies(blocker_task, blocked_task) VALUES ('TASK-2', 'TASK-1'), ('TASK-3', 'TASK-1')",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	chdir(t, projDir)

	body, _ := json.Marshal(apiSetTaskDependsOnRequest{DependsOn: []string{}})
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/TASK-1/depends_on", bytes.NewReader(body))
	req.SetPathValue("id", "TASK-1")
	rec := httptest.NewRecorder()
	apiSetTaskDependsOnHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := readDeps(t, db, "TASK-1"); len(got) != 0 {
		t.Fatalf("expected no deps, got %v", got)
	}
}

// TestApiSetTaskDependsOnRejectsUnknown asserts that an id that does not
// exist in the tasks table returns 400 with no DB writes.
func TestApiSetTaskDependsOnRejectsUnknown(t *testing.T) {
	db, projDir, _ := setupBoardProject(t)
	chdir(t, projDir)

	body, _ := json.Marshal(apiSetTaskDependsOnRequest{DependsOn: []string{"TASK-999"}})
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/TASK-1/depends_on", bytes.NewReader(body))
	req.SetPathValue("id", "TASK-1")
	rec := httptest.NewRecorder()
	apiSetTaskDependsOnHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
	if got := readDeps(t, db, "TASK-1"); len(got) != 0 {
		t.Fatalf("expected no DB writes on bad input, got %v", got)
	}
}

// TestApiSetTaskDependsOnRejectsSelfDependency asserts that self-deps are
// rejected with 400 before any DB mutation.
func TestApiSetTaskDependsOnRejectsSelfDependency(t *testing.T) {
	_, projDir, _ := setupBoardProject(t)
	chdir(t, projDir)

	body, _ := json.Marshal(apiSetTaskDependsOnRequest{DependsOn: []string{"TASK-1"}})
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/TASK-1/depends_on", bytes.NewReader(body))
	req.SetPathValue("id", "TASK-1")
	rec := httptest.NewRecorder()
	apiSetTaskDependsOnHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestApiSetTaskDependsOnUnknownSubject asserts that a missing subject task
// id returns 404. The depends_on body is irrelevant in this case; the
// handler must verify the subject exists before mutating anything.
func TestApiSetTaskDependsOnUnknownSubject(t *testing.T) {
	_, projDir, _ := setupBoardProject(t)
	chdir(t, projDir)

	body, _ := json.Marshal(apiSetTaskDependsOnRequest{DependsOn: []string{"TASK-2"}})
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/TASK-9999/depends_on", bytes.NewReader(body))
	req.SetPathValue("id", "TASK-9999")
	rec := httptest.NewRecorder()
	apiSetTaskDependsOnHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}

// chdir switches into projDir for the duration of the test so OpenProjectDB
// (which reads from cwd) finds the right cache file. Restores the previous
// cwd in t.Cleanup so test order doesn't matter.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// readDeps returns blocker_task ids for a given blocked task, sorted for
// stable comparison.
func readDeps(t *testing.T, db *sql.DB, blocked string) []string {
	t.Helper()
	rows, err := db.Query(
		"SELECT blocker_task FROM task_dependencies WHERE blocked_task = ? ORDER BY blocker_task",
		blocked,
	)
	if err != nil {
		t.Fatalf("query deps: %v", err)
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
	sort.Strings(out)
	return out
}

// refIDs lifts the IDs out of a slice of apiTaskRef so tests can compare
// with a flat []string.
func refIDs(refs []apiTaskRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.ID)
	}
	return out
}
