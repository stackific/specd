package watcher

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackific/specd/internal/workspace"
)

func setupWatchedWorkspace(t *testing.T) (*workspace.Workspace, *Watcher) {
	t.Helper()

	dir := t.TempDir()
	w, err := workspace.Init(dir, workspace.InitOptions{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	logger := log.New(os.Stderr, "test: ", log.LstdFlags)
	wt, err := New(w, logger)
	if err != nil {
		w.Close()
		t.Fatalf("New watcher: %v", err)
	}

	if err := wt.Start(); err != nil {
		w.Close()
		t.Fatalf("Start watcher: %v", err)
	}

	t.Cleanup(func() {
		wt.Stop()
		w.Close()
	})

	return w, wt
}

// waitForSync waits for debounced watcher events to process.
// Must account for fsnotify detection + 200ms debounce + processing time.
func waitForSync() {
	time.Sleep(800 * time.Millisecond)
}

func TestWatcherDetectsTaskEdit(t *testing.T) {
	w, _ := setupWatchedWorkspace(t)

	// Create a spec and task via CLI.
	_, err := w.NewSpec(workspace.NewSpecInput{
		Title:   "Auth",
		Type:    "technical",
		Summary: "Authentication spec",
		Body:    "Body.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	result, err := w.NewTask(workspace.NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Design schema",
		Summary: "Design the DB schema",
		Body:    "# Design schema\n\n## Acceptance criteria\n\n- [ ] Users table\n- [ ] Sessions table",
	})
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}

	// Wait for watcher to register the new spec directory.
	waitForSync()

	// Read initial state.
	task, _ := w.ReadTask("TASK-1")
	if task.Title != "Design schema" {
		t.Fatalf("initial title = %q", task.Title)
	}

	// Simulate a user editing the task file to check a criterion.
	absPath := filepath.Join(w.Root, result.Path)
	data, _ := os.ReadFile(absPath)
	modified := replaceInString(string(data), "- [ ] Users table", "- [x] Users table")
	os.WriteFile(absPath, []byte(modified), 0o644)

	waitForSync()

	// Verify the criteria was updated in SQLite.
	criteria, err := w.ListCriteria("TASK-1")
	if err != nil {
		t.Fatalf("ListCriteria: %v", err)
	}

	foundChecked := false
	for _, c := range criteria {
		if c.Text == "Users table" && c.Checked {
			foundChecked = true
		}
	}
	if !foundChecked {
		t.Error("expected 'Users table' criterion to be checked after file edit")
	}
}

func TestWatcherDetectsSpecTitleEdit(t *testing.T) {
	w, _ := setupWatchedWorkspace(t)

	specResult, err := w.NewSpec(workspace.NewSpecInput{
		Title:   "Original Title",
		Type:    "technical",
		Summary: "A test spec",
		Body:    "Body content.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	// Wait for watcher to register the new spec directory.
	waitForSync()

	// Edit the spec file to change the title.
	absPath := filepath.Join(w.Root, specResult.Path)
	data, _ := os.ReadFile(absPath)
	modified := replaceInString(string(data), "title: Original Title", "title: Updated Title")
	os.WriteFile(absPath, []byte(modified), 0o644)

	waitForSync()

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Title != "Updated Title" {
		t.Errorf("spec title = %q, want 'Updated Title'", spec.Title)
	}
}

func TestWatcherSkipsCLIWrite(t *testing.T) {
	w, _ := setupWatchedWorkspace(t)

	_, err := w.NewSpec(workspace.NewSpecInput{
		Title:   "Test Spec",
		Type:    "technical",
		Summary: "A test spec",
		Body:    "Body.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	// The CLI write sets the content hash, so the watcher should skip it.
	// We just need to verify no errors and the spec is still readable.
	waitForSync()

	spec, err := w.ReadSpec("SPEC-1")
	if err != nil {
		t.Fatalf("ReadSpec: %v", err)
	}
	if spec.Title != "Test Spec" {
		t.Errorf("title = %q", spec.Title)
	}
}

func TestWatcherRejectsNonCanonicalFile(t *testing.T) {
	w, _ := setupWatchedWorkspace(t)

	// Create a non-canonical file in specs directory.
	badFile := filepath.Join(w.Root, "specd", "specs", "random-notes.md")
	os.WriteFile(badFile, []byte("# Random notes\n"), 0o644)

	waitForSync()

	// Check rejected_files table.
	var count int
	w.DB.QueryRow("SELECT COUNT(*) FROM rejected_files").Scan(&count)
	if count == 0 {
		t.Error("expected non-canonical file to be rejected")
	}
}

func TestWatcherHandlesTaskDeletion(t *testing.T) {
	w, _ := setupWatchedWorkspace(t)

	w.NewSpec(workspace.NewSpecInput{
		Title:   "S",
		Type:    "technical",
		Summary: "spec s",
		Body:    "Body.",
	})
	result, _ := w.NewTask(workspace.NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Task to delete",
		Summary: "will be deleted via file removal",
		Body:    "Body.",
	})

	// Wait for watcher to register the new spec directory.
	waitForSync()

	// Delete the file directly (simulating user deleting in editor).
	absPath := filepath.Join(w.Root, result.Path)
	os.Remove(absPath)

	waitForSync()

	// Task should be in trash now.
	_, err := w.ReadTask("TASK-1")
	if err == nil {
		t.Error("expected task to be deleted after file removal")
	}

	var trashCount int
	w.DB.QueryRow("SELECT COUNT(*) FROM trash WHERE original_id = 'TASK-1' AND deleted_by = 'watcher'").Scan(&trashCount)
	if trashCount != 1 {
		t.Errorf("expected 1 trash entry from watcher, got %d", trashCount)
	}
}

func TestSpecIDFromPath(t *testing.T) {
	wt := &Watcher{}
	tests := map[string]string{
		"specd/specs/SPEC-42-oauth/spec.md":                         "SPEC-42",
		"specd/specs/SPEC-1-auth/TASK-5-design.md":                  "SPEC-1",
		"specd/specs/SPEC-100-long-name/spec.md":                    "SPEC-100",
		"specd/specs/not-a-spec/spec.md":                            "",
	}

	for input, want := range tests {
		got := wt.specIDFromPath(input)
		if got != want {
			t.Errorf("specIDFromPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTaskIDFromPath(t *testing.T) {
	wt := &Watcher{}
	tests := map[string]string{
		"specd/specs/SPEC-1-auth/TASK-5-design-schema.md": "TASK-5",
		"specd/specs/SPEC-1-auth/TASK-100-implement.md":   "TASK-100",
		"specd/specs/SPEC-1-auth/spec.md":                 "",
		"specd/specs/SPEC-1-auth/notes.md":                "",
	}

	for input, want := range tests {
		got := wt.taskIDFromPath(input)
		if got != want {
			t.Errorf("taskIDFromPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func replaceInString(s, old, new string) string {
	i := 0
	for i < len(s) {
		idx := indexOf(s[i:], old)
		if idx < 0 {
			break
		}
		s = s[:i+idx] + new + s[i+idx+len(old):]
		i = i + idx + len(new)
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
