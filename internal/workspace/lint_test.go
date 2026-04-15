package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLintCleanWorkspace(t *testing.T) {
	w := setupWorkspace(t)

	result, err := w.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	if result.Counts.Errors != 0 {
		t.Errorf("expected 0 errors on clean workspace, got %d", result.Counts.Errors)
	}
}

func TestLintOrphanSpec(t *testing.T) {
	w := setupWorkspace(t)

	// Create a spec with no tasks and no links.
	_, err := w.NewSpec(NewSpecInput{
		Title:   "Orphan spec",
		Type:    "technical",
		Summary: "This spec has no links",
		Body:    "Body.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	result, err := w.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "orphan_spec" && issue.ID == "SPEC-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected orphan_spec warning for SPEC-1")
	}
}

func TestLintMissingFile(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.NewSpec(NewSpecInput{
		Title:   "Will delete",
		Type:    "technical",
		Summary: "File will be deleted",
		Body:    "Body.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	// Delete the spec file but leave the DB row.
	spec, _ := w.ReadSpec("SPEC-1")
	os.Remove(filepath.Join(w.Root, spec.Path))

	result, err := w.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "missing_file" && issue.ID == "SPEC-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing_file error for SPEC-1")
	}
}

func TestLintMissingSummary(t *testing.T) {
	w := setupWorkspace(t)

	// Insert a spec with a single-word summary directly.
	now := time.Now().UTC().Format(time.RFC3339)
	w.DB.Exec(`INSERT INTO specs (id, slug, title, type, summary, body, path, position, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'test', 'Test', 'technical', 'x', 'body', 'specd/specs/SPEC-1-test/spec.md', 0, 'hash', ?, ?)`,
		now, now)

	// Create the file so it doesn't show as missing.
	dir := filepath.Join(w.Root, "specd", "specs", "SPEC-1-test")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "spec.md"), []byte("---\ntitle: Test\ntype: technical\nsummary: x\n---\n\nbody"), 0o644)

	result, err := w.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "missing_summary" && issue.ID == "SPEC-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing_summary warning for SPEC-1")
	}
}

func TestLintStaleTidy(t *testing.T) {
	w := setupWorkspace(t)

	// Set last_tidy_at to 10 days ago.
	old := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	w.DB.SetMeta("last_tidy_at", old)

	result, err := w.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "stale_tidy" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected stale_tidy warning")
	}
}

func TestTidyUpdatesTimestamp(t *testing.T) {
	w := setupWorkspace(t)

	// Set tidy to old.
	old := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	w.DB.SetMeta("last_tidy_at", old)

	_, err := w.Tidy()
	if err != nil {
		t.Fatalf("Tidy: %v", err)
	}

	val, _ := w.DB.GetMeta("last_tidy_at")
	parsed, _ := time.Parse(time.RFC3339, val)
	if time.Since(parsed) > time.Minute {
		t.Errorf("last_tidy_at not updated: %s", val)
	}
}

func TestLintDepCycle(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "spec s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "task one", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "task two", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "task three", Status: "todo"})

	// Create a cycle: T1 -> T2 -> T3 -> T1 (bypassing cycle detection by direct DB insert).
	w.DB.Exec("INSERT INTO task_dependencies (blocker_task, blocked_task) VALUES ('TASK-1', 'TASK-2')")
	w.DB.Exec("INSERT INTO task_dependencies (blocker_task, blocked_task) VALUES ('TASK-2', 'TASK-3')")
	w.DB.Exec("INSERT INTO task_dependencies (blocker_task, blocked_task) VALUES ('TASK-3', 'TASK-1')")

	result, err := w.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "dependency_cycle" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected dependency_cycle error")
	}
}
