package cmd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

// resetNewTaskFlags clears flag state to prevent leakage between tests.
func resetNewTaskFlags() {
	newTaskCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupTaskProject creates an initialized project with a spec already in the DB.
func setupTaskProject(t *testing.T) (projectDir string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetNewTaskFlags()

	projectDir = filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Create a spec to be the parent.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Auth Flow", "--summary", "OAuth2 login", "--body", "## Details\n\nImplement OAuth2.\n\n## Acceptance Criteria\n\n- The system must support OAuth2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec: %v", err)
	}

	return projectDir
}

// TestNewTaskCreatesFileAndDBRecord verifies that new-task creates the
// task.md file, inserts a row into the database, and increments the counter.
func TestNewTaskCreatesFileAndDBRecord(t *testing.T) {
	setupTaskProject(t)
	resetNewTaskFlags()

	rootCmd.SetArgs([]string{
		"new-task",
		"--spec-id", "SPEC-1",
		"--title", "Implement Redirect",
		"--summary", "Build OAuth redirect handler",
		"--body", "## Overview\n\nBuild redirect.\n\n## Acceptance Criteria\n\n- [ ] The handler must redirect to consent screen\n- [ ] The state parameter should be random",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-task: %v", err)
	}

	// Verify task file exists in the spec directory.
	taskFile := filepath.Join("specd", "specs", "spec-1", "TASK-1.md")
	if _, err := os.Stat(taskFile); err != nil {
		t.Fatalf("task file not created: %v", err)
	}

	// Verify DB record.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var id, specID, status string
	err = db.QueryRow("SELECT id, spec_id, status FROM tasks WHERE id = 'TASK-1'").Scan(&id, &specID, &status)
	if err != nil {
		t.Fatalf("reading task from DB: %v", err)
	}
	if specID != "SPEC-1" {
		t.Errorf("expected spec_id SPEC-1, got %s", specID)
	}
	if status != "backlog" {
		t.Errorf("expected default status %q, got %q", "backlog", status)
	}

	// Verify counter incremented.
	var nextID int
	_ = db.QueryRow("SELECT CAST(value AS INTEGER) FROM meta WHERE key = 'next_task_id'").Scan(&nextID)
	if nextID != 2 {
		t.Errorf("expected next_task_id=2, got %d", nextID)
	}

	// Verify task_criteria were inserted.
	var criteriaCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM task_criteria WHERE task_id = 'TASK-1'").Scan(&criteriaCount)
	if criteriaCount != 2 {
		t.Errorf("expected 2 task criteria, got %d", criteriaCount)
	}
}

// TestNewTaskOutputJSON verifies the JSON response structure.
func TestNewTaskOutputJSON(t *testing.T) {
	setupTaskProject(t)
	resetNewTaskFlags()

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{
		"new-task",
		"--spec-id", "SPEC-1",
		"--title", "Test Task",
		"--summary", "A test task",
		"--body", "## Details\n\nTask body.",
	})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("new-task: %v", err)
	}

	out := make([]byte, 4096)
	n, _ := r.Read(out)
	output := string(out[:n])

	var resp NewTaskResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("parsing JSON response: %v\noutput: %s", err, output)
	}
	if resp.ID != "TASK-1" {
		t.Errorf("expected ID TASK-1, got %s", resp.ID)
	}
	if resp.SpecID != "SPEC-1" {
		t.Errorf("expected spec_id SPEC-1, got %s", resp.SpecID)
	}
	if resp.Status != "backlog" {
		t.Errorf("expected status backlog, got %s", resp.Status)
	}
}

// TestNewTaskInvalidSpecID verifies that creating a task with a nonexistent
// spec ID returns an error.
func TestNewTaskInvalidSpecID(t *testing.T) {
	setupTaskProject(t)
	resetNewTaskFlags()

	rootCmd.SetArgs([]string{
		"new-task",
		"--spec-id", "SPEC-999",
		"--title", "Orphan Task",
		"--summary", "No spec",
		"--body", "Body",
	})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent spec, got nil")
	}
}

// TestNewTaskMultipleTasks verifies creating multiple tasks for the same spec.
func TestNewTaskMultipleTasks(t *testing.T) {
	setupTaskProject(t)

	for i, title := range []string{"Task One", "Task Two", "Task Three"} {
		resetNewTaskFlags()
		rootCmd.SetArgs([]string{
			"new-task",
			"--spec-id", "SPEC-1",
			"--title", title,
			"--summary", "Summary " + title,
			"--body", "## Details\n\nBody for " + title,
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("new-task %d: %v", i+1, err)
		}
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE spec_id = 'SPEC-1'").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 tasks, got %d", count)
	}

	var nextID int
	_ = db.QueryRow("SELECT CAST(value AS INTEGER) FROM meta WHERE key = 'next_task_id'").Scan(&nextID)
	if nextID != 4 {
		t.Errorf("expected next_task_id=4, got %d", nextID)
	}
}

// TestExtractTaskCriteria verifies parsing of checkbox items from task body.
func TestExtractTaskCriteria(t *testing.T) {
	body := `# Title

## Overview

Some overview text.

## Acceptance Criteria

- [ ] The handler must redirect to the consent screen
- [x] The state parameter should be random
- [ ] Users may configure scopes

## Notes

Not part of criteria.
`
	criteria := extractTaskCriteria(body)
	if len(criteria) != 3 {
		t.Fatalf("expected 3 criteria, got %d: %v", len(criteria), criteria)
	}
	if criteria[0] != "The handler must redirect to the consent screen" {
		t.Errorf("criteria 0: got %q", criteria[0])
	}
	if criteria[1] != "The state parameter should be random" {
		t.Errorf("criteria 1: got %q", criteria[1])
	}
}

// TestExtractTaskCriteriaNoCriteria verifies empty result when no section exists.
func TestExtractTaskCriteriaNoCriteria(t *testing.T) {
	body := "# Title\n\n## Overview\n\nNo criteria here."
	criteria := extractTaskCriteria(body)
	if len(criteria) != 0 {
		t.Errorf("expected 0 criteria, got %d", len(criteria))
	}
}

// TestExtractTaskCriteriaPlainBulletsIgnored verifies plain bullets are not
// parsed as task criteria (only checkboxes count).
func TestExtractTaskCriteriaPlainBulletsIgnored(t *testing.T) {
	body := "# Title\n\n## Acceptance Criteria\n\n- Plain bullet not a checkbox\n- [ ] Only this is a criterion\n"
	criteria := extractTaskCriteria(body)
	if len(criteria) != 1 {
		t.Fatalf("expected 1 criterion (plain bullets should be ignored), got %d: %v", len(criteria), criteria)
	}
	if criteria[0] != "Only this is a criterion" {
		t.Errorf("expected checkbox text, got %q", criteria[0])
	}
}

// TestParseCheckboxItem verifies the checkbox item parser.
func TestParseCheckboxItem(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"- [ ] Unchecked item", "Unchecked item"},
		{"- [x] Checked item", "Checked item"},
		{"- [X] Upper case checked", "Upper case checked"},
		{"- Plain bullet", ""},
		{"- [ ] ", ""},           // empty text after checkbox
		{"Not a bullet", ""},     // not a list item
		{"  - [ ] Indented", ""}, // indented — not handled at this level
	}
	for _, tt := range tests {
		got := parseCheckboxItem(tt.line)
		if got != tt.want {
			t.Errorf("parseCheckboxItem(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

// TestNewTaskMarkdownContainsCorrectFrontmatter verifies the task.md file
// content has correct YAML frontmatter.
func TestNewTaskMarkdownContainsCorrectFrontmatter(t *testing.T) {
	setupTaskProject(t)
	resetNewTaskFlags()

	rootCmd.SetArgs([]string{
		"new-task",
		"--spec-id", "SPEC-1",
		"--title", "Setup Database",
		"--summary", "Create the database schema",
		"--body", "## Overview\n\nCreate tables.",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-task: %v", err)
	}

	data, err := os.ReadFile(filepath.Join("specd", "specs", "spec-1", "TASK-1.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	for _, want := range []string{
		"id: TASK-1",
		"spec_id: SPEC-1",
		"status: backlog",
		"summary: Create the database schema",
		"# Setup Database",
	} {
		if !contains(content, want) {
			t.Errorf("task.md missing %q", want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
