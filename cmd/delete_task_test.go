package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

func resetDeleteTaskFlags() {
	deleteTaskCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupProjectWithTaskForDelete creates a project with a spec and task,
// returns after chdir into the project dir.
func setupProjectWithTaskForDelete(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetNewTaskFlags()
	resetDeleteTaskFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Test Spec", "--summary", "A test", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{
		"new-task", "--spec-id", "SPEC-1",
		"--title", "Test Task", "--summary", "A task",
		"--body", "## Details\n\nBody.\n\n## Acceptance Criteria\n\n- [ ] Must do X",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// TestDeleteTaskRemovesFromDB verifies that delete-task removes the task
// row and its cascaded dependents from the database.
func TestDeleteTaskRemovesFromDB(t *testing.T) {
	setupProjectWithTaskForDelete(t)

	resetDeleteTaskFlags()
	rootCmd.SetArgs([]string{"delete-task", "--id", "TASK-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-task: %v", err)
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Error("task should be deleted from DB")
	}

	// Criteria should be cascade-deleted too.
	_ = db.QueryRow("SELECT COUNT(*) FROM task_criteria WHERE task_id = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Error("task criteria should be cascade-deleted")
	}
}

// TestDeleteTaskRemovesFile verifies that delete-task removes the task
// file from disk.
func TestDeleteTaskRemovesFile(t *testing.T) {
	setupProjectWithTaskForDelete(t)

	// Find the task file path.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	var taskPath string
	_ = db.QueryRow("SELECT path FROM tasks WHERE id = 'TASK-1'").Scan(&taskPath)
	_ = db.Close()

	if _, err := os.Stat(taskPath); err != nil {
		t.Fatalf("task file should exist before delete: %v", err)
	}

	resetDeleteTaskFlags()
	rootCmd.SetArgs([]string{"delete-task", "--id", "TASK-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-task: %v", err)
	}

	if _, err := os.Stat(taskPath); !os.IsNotExist(err) {
		t.Error("task file should be removed from disk after delete")
	}
}

// TestDeleteTaskSpecSurvives verifies that deleting a task does not affect
// the parent spec.
func TestDeleteTaskSpecSurvives(t *testing.T) {
	setupProjectWithTaskForDelete(t)

	resetDeleteTaskFlags()
	rootCmd.SetArgs([]string{"delete-task", "--id", "TASK-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-task: %v", err)
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Error("parent spec SPEC-1 should survive task deletion")
	}

	// Spec directory should still exist.
	if _, err := os.Stat(filepath.Join("specd", "specs", "spec-1")); err != nil {
		t.Error("spec directory should still exist on disk")
	}
}

// TestDeleteTaskOtherTasksSurvive verifies that deleting one task does not
// affect sibling tasks.
func TestDeleteTaskOtherTasksSurvive(t *testing.T) {
	setupProjectWithTaskForDelete(t)

	// Create a second task.
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{
		"new-task", "--spec-id", "SPEC-1",
		"--title", "Task Two", "--summary", "Second task",
		"--body", "Body two",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Delete TASK-1.
	resetDeleteTaskFlags()
	rootCmd.SetArgs([]string{"delete-task", "--id", "TASK-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-task: %v", err)
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-2'").Scan(&count)
	if count != 1 {
		t.Error("TASK-2 should survive deletion of TASK-1")
	}
}

// TestDeleteTaskIDsNotReused verifies that after deleting a task,
// creating a new task gets the next ID, not the deleted one.
func TestDeleteTaskIDsNotReused(t *testing.T) {
	setupProjectWithTaskForDelete(t)

	resetDeleteTaskFlags()
	rootCmd.SetArgs([]string{"delete-task", "--id", "TASK-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-task: %v", err)
	}

	resetNewTaskFlags()
	rootCmd.SetArgs([]string{
		"new-task", "--spec-id", "SPEC-1",
		"--title", "After Delete", "--summary", "Should be TASK-2",
		"--body", "Body",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Error("TASK-1 should not be reused")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-2'").Scan(&count)
	if count != 1 {
		t.Error("new task should be TASK-2")
	}
}

// TestDeleteTaskNotFound verifies that deleting a nonexistent task returns
// an error.
func TestDeleteTaskNotFound(t *testing.T) {
	setupProjectWithTaskForDelete(t)

	resetDeleteTaskFlags()
	rootCmd.SetArgs([]string{"delete-task", "--id", "TASK-999"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when deleting nonexistent task")
	}
}
