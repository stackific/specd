package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrashListEmpty(t *testing.T) {
	w := setupWorkspace(t)

	items, err := w.ListTrash(TrashListFilter{})
	if err != nil {
		t.Fatalf("ListTrash: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 trash items, got %d", len(items))
	}
}

func TestTrashDeleteAndList(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "To delete", Type: "functional", Summary: "will be deleted"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Task to delete", Summary: "will be deleted"})

	// Delete the task.
	if err := w.DeleteTask("TASK-1"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	items, err := w.ListTrash(TrashListFilter{})
	if err != nil {
		t.Fatalf("ListTrash: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 trash item, got %d", len(items))
	}
	if items[0].Kind != "task" || items[0].OriginalID != "TASK-1" {
		t.Errorf("unexpected trash item: %+v", items[0])
	}

	// Filter by kind.
	specItems, _ := w.ListTrash(TrashListFilter{Kind: "spec"})
	if len(specItems) != 0 {
		t.Errorf("expected 0 spec trash items, got %d", len(specItems))
	}

	taskItems, _ := w.ListTrash(TrashListFilter{Kind: "task"})
	if len(taskItems) != 1 {
		t.Errorf("expected 1 task trash item, got %d", len(taskItems))
	}
}

func TestTrashRestoreTask(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Parent", Type: "functional", Summary: "parent spec"})
	w.NewTask(NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Restorable",
		Summary: "will be restored",
		Body:    "## Restorable\n\n## Acceptance criteria\n\n- [ ] Criterion 1",
	})

	// Delete the task.
	if err := w.DeleteTask("TASK-1"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	// Verify it's gone from tasks.
	_, err := w.ReadTask("TASK-1")
	if err == nil {
		t.Fatal("expected task to be deleted")
	}

	// List trash to get the trash ID.
	items, _ := w.ListTrash(TrashListFilter{})
	if len(items) != 1 {
		t.Fatalf("expected 1 trash item, got %d", len(items))
	}

	// Restore.
	result, err := w.RestoreTrash(items[0].ID)
	if err != nil {
		t.Fatalf("RestoreTrash: %v", err)
	}

	if result.RestoredID != "TASK-1" {
		t.Errorf("restored ID = %s, want TASK-1", result.RestoredID)
	}

	// Verify it's back in tasks.
	task, err := w.ReadTask("TASK-1")
	if err != nil {
		t.Fatalf("ReadTask after restore: %v", err)
	}
	if task.Title != "Restorable" {
		t.Errorf("title = %q", task.Title)
	}

	// Verify file exists.
	if _, err := os.Stat(filepath.Join(w.Root, result.Path)); err != nil {
		t.Errorf("restored file missing: %v", err)
	}

	// Verify trash is empty.
	remaining, _ := w.ListTrash(TrashListFilter{})
	if len(remaining) != 0 {
		t.Errorf("expected empty trash after restore, got %d", len(remaining))
	}
}

func TestTrashRestoreSpec(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Spec to restore", Type: "business", Summary: "restore test spec"})

	if err := w.DeleteSpec("SPEC-1"); err != nil {
		t.Fatalf("DeleteSpec: %v", err)
	}

	items, _ := w.ListTrash(TrashListFilter{})
	if len(items) != 1 {
		t.Fatalf("expected 1 trash item, got %d", len(items))
	}

	result, err := w.RestoreTrash(items[0].ID)
	if err != nil {
		t.Fatalf("RestoreTrash: %v", err)
	}

	if result.Kind != "spec" {
		t.Errorf("kind = %s", result.Kind)
	}

	// Verify spec is readable.
	spec, err := w.ReadSpec(result.RestoredID)
	if err != nil {
		t.Fatalf("ReadSpec: %v", err)
	}
	if spec.Title != "Spec to restore" {
		t.Errorf("title = %q", spec.Title)
	}
}

func TestTrashRestoreWithIDConflict(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Original", Type: "functional", Summary: "original spec"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "To delete", Summary: "delete me"})

	if err := w.DeleteTask("TASK-1"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	// Manually re-insert a task with TASK-1 ID to simulate ID reuse after merge.
	now := "2024-01-01T00:00:00Z"
	spec, _ := w.ReadSpec("SPEC-1")
	specDir := filepath.Dir(spec.Path)
	taskPath := filepath.Join(specDir, "TASK-1-conflict.md")
	absPath := filepath.Join(w.Root, taskPath)
	os.WriteFile(absPath, []byte("---\ntitle: Conflict\nstatus: backlog\nsummary: conflict\n---\n\nbody"), 0o644)
	w.DB.Exec(`INSERT INTO tasks (id, slug, spec_id, title, status, summary, body, path, position, content_hash, created_at, updated_at)
		VALUES ('TASK-1', 'conflict', 'SPEC-1', 'Conflict', 'backlog', 'conflict', 'body', ?, 0, 'hash', ?, ?)`,
		taskPath, now, now)

	items, _ := w.ListTrash(TrashListFilter{})
	if len(items) != 1 {
		t.Fatalf("expected 1 trash item, got %d", len(items))
	}

	result, err := w.RestoreTrash(items[0].ID)
	if err != nil {
		t.Fatalf("RestoreTrash: %v", err)
	}

	// Should have been renumbered since TASK-1 is occupied.
	if result.RestoredID == "TASK-1" {
		t.Error("expected restored task to get a new ID since TASK-1 is taken")
	}
	if result.Warning == "" {
		t.Error("expected warning about ID conflict")
	}
}

func TestPurgeAllTrash(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S1", Type: "functional", Summary: "spec one"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "task one"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "task two"})

	w.DeleteTask("TASK-1")
	w.DeleteTask("TASK-2")

	items, _ := w.ListTrash(TrashListFilter{})
	if len(items) != 2 {
		t.Fatalf("expected 2 trash items, got %d", len(items))
	}

	count, err := w.PurgeAllTrash()
	if err != nil {
		t.Fatalf("PurgeAllTrash: %v", err)
	}
	if count != 2 {
		t.Errorf("purged %d, want 2", count)
	}

	remaining, _ := w.ListTrash(TrashListFilter{})
	if len(remaining) != 0 {
		t.Errorf("expected empty trash, got %d", len(remaining))
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		hours float64
	}{
		{"30d", 30 * 24},
		{"7d", 7 * 24},
		{"24h", 24},
		{"60m", 1},
	}

	for _, tt := range tests {
		dur, err := parseDuration(tt.input)
		if err != nil {
			t.Errorf("parseDuration(%q): %v", tt.input, err)
			continue
		}
		if dur.Hours() != tt.hours {
			t.Errorf("parseDuration(%q) = %v hours, want %v", tt.input, dur.Hours(), tt.hours)
		}
	}
}
