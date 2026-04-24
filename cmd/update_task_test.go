package cmd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

func resetUpdateTaskFlags() {
	updateTaskCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupUpdateTaskProject creates a project with a spec and a task that has criteria.
func setupUpdateTaskProject(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetNewTaskFlags()
	resetUpdateTaskFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resetNewSpecFlags()
	rootCmd.SetArgs([]string{
		"new-spec", "--title", "Auth", "--summary", "OAuth2",
		"--body", "## Overview\n\nAuth spec.\n\n## Acceptance Criteria\n\n- The system must authenticate",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resetNewTaskFlags()
	rootCmd.SetArgs([]string{
		"new-task", "--spec-id", "SPEC-1", "--title", "Login", "--summary", "Login handler",
		"--body", "## Overview\n\nBuild login.\n\n## Acceptance Criteria\n\n- [ ] The handler must validate credentials\n- [ ] The handler should return a JWT token\n- [ ] The response will include user info",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// captureUpdateTask runs update-task and returns the parsed response.
func captureUpdateTask(t *testing.T, args ...string) UpdateTaskResponse {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetUpdateTaskFlags()
	rootCmd.SetArgs(append([]string{"update-task"}, args...))
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("update-task: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var resp UpdateTaskResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v\noutput: %s", err, out[:n])
	}
	return resp
}

// TestUpdateTaskStatus verifies changing task status.
func TestUpdateTaskStatus(t *testing.T) {
	setupUpdateTaskProject(t)

	resp := captureUpdateTask(t, "--id", "TASK-1", "--status", "todo")
	if resp.Status != "todo" {
		t.Errorf("expected status todo, got %q", resp.Status)
	}
	if resp.ID != "TASK-1" {
		t.Errorf("expected ID TASK-1, got %s", resp.ID)
	}

	// Verify DB was updated.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var status string
	_ = db.QueryRow("SELECT status FROM tasks WHERE id = 'TASK-1'").Scan(&status)
	if status != "todo" {
		t.Errorf("DB status: expected todo, got %q", status)
	}
}

// TestUpdateTaskCheckCriteria verifies checking acceptance criteria.
func TestUpdateTaskCheckCriteria(t *testing.T) {
	setupUpdateTaskProject(t)

	resp := captureUpdateTask(t, "--id", "TASK-1", "--check", "1,3")

	// Criteria 1 and 3 should be checked, 2 unchecked.
	if len(resp.Criteria) != 3 {
		t.Fatalf("expected 3 criteria, got %d", len(resp.Criteria))
	}
	if resp.Criteria[0].Checked != 1 {
		t.Errorf("criterion 1 should be checked")
	}
	if resp.Criteria[1].Checked != 0 {
		t.Errorf("criterion 2 should be unchecked")
	}
	if resp.Criteria[2].Checked != 1 {
		t.Errorf("criterion 3 should be checked")
	}

	// Verify DB.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var checked int
	_ = db.QueryRow("SELECT checked FROM task_criteria WHERE task_id = 'TASK-1' AND position = 1").Scan(&checked)
	if checked != 1 {
		t.Errorf("DB criterion 1 should be checked")
	}
}

// TestUpdateTaskUncheckCriteria verifies unchecking a previously checked criterion.
func TestUpdateTaskUncheckCriteria(t *testing.T) {
	setupUpdateTaskProject(t)

	// First check criterion 1.
	captureUpdateTask(t, "--id", "TASK-1", "--check", "1")

	// Then uncheck it.
	resp := captureUpdateTask(t, "--id", "TASK-1", "--uncheck", "1")

	if resp.Criteria[0].Checked != 0 {
		t.Errorf("criterion 1 should be unchecked after --uncheck")
	}
}

// TestUpdateTaskRewritesFile verifies the task file is rewritten with
// correct checkbox state.
func TestUpdateTaskRewritesFile(t *testing.T) {
	setupUpdateTaskProject(t)

	captureUpdateTask(t, "--id", "TASK-1", "--check", "2", "--status", "in_progress")

	// Read the task file and verify checkboxes.
	taskFile := filepath.Join("specd", "specs", "spec-1", "TASK-1.md")
	data, err := os.ReadFile(taskFile) //nolint:gosec // test reads a known file
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Criterion 2 should be checked, others unchecked.
	if !strings.Contains(content, "- [ ] The handler must validate credentials") {
		t.Error("criterion 1 should be unchecked in file")
	}
	if !strings.Contains(content, "- [x] The handler should return a JWT token") {
		t.Error("criterion 2 should be checked in file")
	}
	if !strings.Contains(content, "- [ ] The response will include user info") {
		t.Error("criterion 3 should be unchecked in file")
	}
	if !strings.Contains(content, "status: in_progress") {
		t.Error("status should be in_progress in file")
	}
}

// TestUpdateTaskInvalidPosition verifies error for nonexistent criterion position.
func TestUpdateTaskInvalidPosition(t *testing.T) {
	setupUpdateTaskProject(t)

	resetUpdateTaskFlags()
	rootCmd.SetArgs([]string{"update-task", "--id", "TASK-1", "--check", "99"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent criterion position")
	}
}

// TestUpdateTaskNotFound verifies error for nonexistent task.
func TestUpdateTaskNotFound(t *testing.T) {
	setupUpdateTaskProject(t)

	resetUpdateTaskFlags()
	rootCmd.SetArgs([]string{"update-task", "--id", "TASK-999", "--status", "done"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

// TestUpdateTaskStatusAndCheck verifies both status and criteria in one call.
func TestUpdateTaskStatusAndCheck(t *testing.T) {
	setupUpdateTaskProject(t)

	resp := captureUpdateTask(t, "--id", "TASK-1", "--status", "done", "--check", "1,2,3")
	if resp.Status != "done" {
		t.Errorf("expected status done, got %q", resp.Status)
	}
	for i, cr := range resp.Criteria {
		if cr.Checked != 1 {
			t.Errorf("criterion %d should be checked", i+1)
		}
	}
}

// TestRebuildTaskBody verifies the body rebuild preserves non-criteria content.
func TestRebuildTaskBody(t *testing.T) {
	body := "## Overview\n\nSome text.\n\n## Acceptance Criteria\n\n- [ ] Must do A\n- [ ] Should do B\n\n## Notes\n\nExtra notes."
	criteria := []criterionState{
		{text: "Must do A", checked: 1},
		{text: "Should do B", checked: 0},
	}

	result := rebuildTaskBody(body, criteria)

	if !strings.Contains(result, "- [x] Must do A") {
		t.Error("expected criterion A to be checked")
	}
	if !strings.Contains(result, "- [ ] Should do B") {
		t.Error("expected criterion B to be unchecked")
	}
	if !strings.Contains(result, "## Overview") {
		t.Error("overview section should be preserved")
	}
	if !strings.Contains(result, "## Notes") {
		t.Error("notes section should be preserved")
	}
	if !strings.Contains(result, "Extra notes.") {
		t.Error("notes content should be preserved")
	}
}
