package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func resetGetTaskFlags() {
	getTaskCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupProjectWithTask creates an initialized project with a spec and a task.
func setupProjectWithTask(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetNewTaskFlags()
	resetGetTaskFlags()

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
	rootCmd.SetArgs([]string{"new-spec", "--title", "Auth Flow", "--summary", "OAuth2 login", "--body", "## Overview\n\nAuth spec."})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	taskBody := "## Overview\n\nBuild redirect.\n\n## Acceptance Criteria\n\n- [ ] The handler must redirect to consent screen\n- [ ] The state parameter should be random"
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{"new-task", "--spec-id", "SPEC-1", "--title", "Implement Redirect", "--summary", "Build OAuth redirect handler", "--body", taskBody})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// captureGetTask runs get-task and returns the parsed response.
func captureGetTask(t *testing.T, taskID string) GetTaskResponse {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetGetTaskFlags()
	rootCmd.SetArgs([]string{"get-task", "--id", taskID})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("get-task: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var resp GetTaskResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v\noutput: %s", err, out[:n])
	}
	return resp
}

// TestGetTaskReturnsFullTask verifies that get-task returns all fields.
func TestGetTaskReturnsFullTask(t *testing.T) {
	setupProjectWithTask(t)
	resp := captureGetTask(t, "TASK-1")

	if resp.ID != "TASK-1" {
		t.Errorf("expected ID TASK-1, got %s", resp.ID)
	}
	if resp.SpecID != "SPEC-1" {
		t.Errorf("expected spec_id SPEC-1, got %s", resp.SpecID)
	}
	if resp.Title != "Implement Redirect" {
		t.Errorf("expected title 'Implement Redirect', got %q", resp.Title)
	}
	if resp.Status != "backlog" {
		t.Errorf("expected status backlog, got %q", resp.Status)
	}
	if resp.Summary != "Build OAuth redirect handler" {
		t.Errorf("expected summary, got %q", resp.Summary)
	}
}

// TestGetTaskReturnsCriteria verifies that get-task includes acceptance criteria.
func TestGetTaskReturnsCriteria(t *testing.T) {
	setupProjectWithTask(t)
	resp := captureGetTask(t, "TASK-1")

	if len(resp.Criteria) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(resp.Criteria))
	}
	if resp.Criteria[0].Text != "The handler must redirect to consent screen" {
		t.Errorf("unexpected criterion 1: %q", resp.Criteria[0].Text)
	}
	if resp.Criteria[0].Checked != 0 {
		t.Errorf("expected unchecked, got %d", resp.Criteria[0].Checked)
	}
}

// TestGetTaskEmptyArraysNotNull verifies that linked_tasks, depends_on,
// and criteria are empty arrays when absent, not null.
func TestGetTaskEmptyArraysNotNull(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetNewTaskFlags()
	resetGetTaskFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec", "--summary", "S", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{"new-task", "--spec-id", "SPEC-1", "--title", "Task", "--summary", "S", "--body", "No criteria"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resp := captureGetTask(t, "TASK-1")

	if resp.LinkedTasks == nil {
		t.Error("linked_tasks should be empty array, not null")
	}
	if resp.DependsOn == nil {
		t.Error("depends_on should be empty array, not null")
	}
	if resp.Criteria == nil {
		t.Error("criteria should be empty array, not null")
	}
}

// TestGetTaskNotFound verifies that get-task returns an error for
// a non-existent task ID.
func TestGetTaskNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetGetTaskFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resetGetTaskFlags()
	rootCmd.SetArgs([]string{"get-task", "--id", "TASK-999"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}
