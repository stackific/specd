package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func resetListTasksFlags() {
	listTasksCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupProjectWithTasks creates a project with a spec and N tasks.
func setupProjectWithTasks(t *testing.T, count int) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()

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
	rootCmd.SetArgs([]string{"new-spec", "--title", "Parent Spec", "--summary", "A spec", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < count; i++ {
		resetNewTaskFlags()
		title := "Task " + string(rune('A'+i))
		rootCmd.SetArgs([]string{"new-task", "--spec-id", "SPEC-1", "--title", title, "--summary", "Summary " + title, "--body", "Body"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("new-task %d: %v", i+1, err)
		}
	}
}

// captureListTasks runs list-tasks and returns the parsed response.
func captureListTasks(t *testing.T, args ...string) ListTasksResponse {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetListTasksFlags()
	fullArgs := append([]string{"list-tasks"}, args...)
	rootCmd.SetArgs(fullArgs)
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("list-tasks: %v", err)
	}

	out := make([]byte, 16384)
	n, _ := r.Read(out)

	var resp ListTasksResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v\noutput: %s", err, out[:n])
	}
	return resp
}

// TestListTasksReturnsAll verifies listing all tasks.
func TestListTasksReturnsAll(t *testing.T) {
	setupProjectWithTasks(t, 3)
	resp := captureListTasks(t)

	if resp.TotalCount != 3 {
		t.Errorf("expected 3 total, got %d", resp.TotalCount)
	}
	if len(resp.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(resp.Tasks))
	}
}

// TestListTasksPagination verifies page size and page number work.
func TestListTasksPagination(t *testing.T) {
	setupProjectWithTasks(t, 5)

	resp := captureListTasks(t, "--page", "1", "--page-size", "2")
	if len(resp.Tasks) != 2 {
		t.Errorf("page 1: expected 2 tasks, got %d", len(resp.Tasks))
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", resp.TotalPages)
	}
}

// TestListTasksFilterBySpecID verifies filtering by parent spec.
func TestListTasksFilterBySpecID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Create 2 specs.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec A", "--summary", "A", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec B", "--summary", "B", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Create tasks under each.
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{"new-task", "--spec-id", "SPEC-1", "--title", "T1", "--summary", "S", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{"new-task", "--spec-id", "SPEC-2", "--title", "T2", "--summary", "S", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Filter by SPEC-1.
	resp := captureListTasks(t, "--spec-id", "SPEC-1")
	if resp.TotalCount != 1 {
		t.Errorf("expected 1 task for SPEC-1, got %d", resp.TotalCount)
	}
	if resp.Tasks[0].SpecID != "SPEC-1" {
		t.Errorf("expected spec_id SPEC-1, got %s", resp.Tasks[0].SpecID)
	}
}

// TestListTasksEmpty verifies empty result returns empty array.
func TestListTasksEmpty(t *testing.T) {
	setupProjectWithTasks(t, 0)
	resp := captureListTasks(t)

	if resp.TotalCount != 0 {
		t.Errorf("expected 0 total, got %d", resp.TotalCount)
	}
	if resp.Tasks == nil {
		t.Error("tasks should be empty array, not null")
	}
}
