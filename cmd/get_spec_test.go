package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func resetGetSpecFlags() {
	getSpecCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupLinkedSpecs creates an initialized project with two specs linked together.
// Returns after chdir into the project dir. Cleanup is handled via t.Cleanup.
func setupLinkedSpecs(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetUpdateSpecFlags()
	resetGetSpecFlags()

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
	rootCmd.SetArgs([]string{"new-spec", "--title", "Auth Flow", "--summary", "OAuth2 login", "--body", "Auth body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Sessions", "--summary", "Token mgmt", "--body", "Session body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-1", "--type", "functional", "--link-specs", "SPEC-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// captureGetSpec runs get-spec and returns the parsed response.
func captureGetSpec(t *testing.T, specID string) GetSpecResponse {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetGetSpecFlags()
	rootCmd.SetArgs([]string{"get-spec", "--id", specID})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("get-spec: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var resp GetSpecResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v\noutput: %s", err, out[:n])
	}
	return resp
}

// TestGetSpecReturnsFullSpec verifies that get-spec returns all fields
// including linked specs.
func TestGetSpecReturnsFullSpec(t *testing.T) {
	setupLinkedSpecs(t)
	resp := captureGetSpec(t, "SPEC-1")

	if resp.ID != "SPEC-1" {
		t.Errorf("expected ID SPEC-1, got %s", resp.ID)
	}
	if resp.Title != "Auth Flow" {
		t.Errorf("expected title 'Auth Flow', got %q", resp.Title)
	}
	if resp.Type != "functional" {
		t.Errorf("expected type functional, got %q", resp.Type)
	}
	if len(resp.LinkedSpecs) != 1 || resp.LinkedSpecs[0] != "SPEC-2" {
		t.Errorf("expected linked_specs [SPEC-2], got %v", resp.LinkedSpecs)
	}
	if resp.Body != "Auth body" {
		t.Errorf("expected body 'Auth body', got %q", resp.Body)
	}
	if resp.Summary != "OAuth2 login" {
		t.Errorf("expected summary 'OAuth2 login', got %q", resp.Summary)
	}
}

// TestGetSpecNoLinksReturnsEmptyArray verifies that a spec with no links
// returns an empty array, not null.
func TestGetSpecNoLinksReturnsEmptyArray(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetGetSpecFlags()

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
	rootCmd.SetArgs([]string{"new-spec", "--title", "Solo Spec", "--summary", "No links", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetGetSpecFlags()
	rootCmd.SetArgs([]string{"get-spec", "--id", "SPEC-1"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("get-spec: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	// Parse and verify linked_specs is an empty array.
	var resp GetSpecResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v", err)
	}
	if resp.LinkedSpecs == nil {
		t.Error("linked_specs should be empty array, not null")
	}
	if len(resp.LinkedSpecs) != 0 {
		t.Errorf("expected 0 linked specs, got %d", len(resp.LinkedSpecs))
	}
}

// TestGetSpecEmptyArraysNotNull verifies that claims and tasks are
// empty arrays when a spec has no tasks or claims.
func TestGetSpecEmptyArraysNotNull(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetGetSpecFlags()

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
	rootCmd.SetArgs([]string{"new-spec", "--title", "No Tasks", "--summary", "Empty", "--body", "Just a body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resp := captureGetSpec(t, "SPEC-1")
	if resp.Claims == nil {
		t.Error("claims should be empty array, not null")
	}
	if len(resp.Claims) != 0 {
		t.Errorf("expected 0 claims, got %d", len(resp.Claims))
	}
	if resp.Tasks == nil {
		t.Error("tasks should be empty array, not null")
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

// TestGetSpecWithTasksAndClaims verifies that get-spec returns tasks
// with criteria and spec claims.
func TestGetSpecWithTasksAndClaims(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetNewTaskFlags()
	resetGetSpecFlags()

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

	specBody := "## Overview\n\nSpec with claims.\n\n## Acceptance Criteria\n\n- The system must authenticate users\n- The system should log failed attempts"
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Auth", "--summary", "Auth spec", "--body", specBody})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	taskBody := "## Overview\n\nImplement login.\n\n## Acceptance Criteria\n\n- [ ] The handler must validate credentials\n- [ ] The handler should return a JWT token"
	resetNewTaskFlags()
	rootCmd.SetArgs([]string{"new-task", "--spec-id", "SPEC-1", "--title", "Login", "--summary", "Login handler", "--body", taskBody})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resp := captureGetSpec(t, "SPEC-1")

	// Verify claims from spec.
	if len(resp.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(resp.Claims))
	}
	if resp.Claims[0].Text != "The system must authenticate users" {
		t.Errorf("unexpected claim 1: %q", resp.Claims[0].Text)
	}
	if resp.Claims[1].Text != "The system should log failed attempts" {
		t.Errorf("unexpected claim 2: %q", resp.Claims[1].Text)
	}

	// Verify tasks.
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
	}
	task := resp.Tasks[0]
	if task.ID != "TASK-1" {
		t.Errorf("expected task ID TASK-1, got %s", task.ID)
	}
	if task.Title != "Login" {
		t.Errorf("expected task title 'Login', got %q", task.Title)
	}

	// Verify task criteria.
	if len(task.Criteria) != 2 {
		t.Fatalf("expected 2 task criteria, got %d", len(task.Criteria))
	}
	if task.Criteria[0].Text != "The handler must validate credentials" {
		t.Errorf("unexpected criterion 1: %q", task.Criteria[0].Text)
	}
	if task.Criteria[1].Text != "The handler should return a JWT token" {
		t.Errorf("unexpected criterion 2: %q", task.Criteria[1].Text)
	}
}

// TestGetSpecNotFound verifies that get-spec returns an error for
// a non-existent spec ID.
func TestGetSpecNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetGetSpecFlags()

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

	resetGetSpecFlags()
	rootCmd.SetArgs([]string{"get-spec", "--id", "SPEC-999"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for non-existent spec")
	}
}
