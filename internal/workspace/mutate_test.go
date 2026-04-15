package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Spec update ---

func TestUpdateSpecTitle(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "Original", Type: "technical", Summary: "s", Body: "body"})

	newTitle := "Updated Title"
	err := w.UpdateSpec("SPEC-1", UpdateSpecInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateSpec: %v", err)
	}

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Title != "Updated Title" {
		t.Errorf("title = %q", spec.Title)
	}
}

func TestUpdateSpecType(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})

	newType := "business"
	w.UpdateSpec("SPEC-1", UpdateSpecInput{Type: &newType})

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Type != "business" {
		t.Errorf("type = %q", spec.Type)
	}
}

func TestUpdateSpecSummary(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "old"})

	newSummary := "new summary"
	w.UpdateSpec("SPEC-1", UpdateSpecInput{Summary: &newSummary})

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Summary != "new summary" {
		t.Errorf("summary = %q", spec.Summary)
	}
}

func TestUpdateSpecBody(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s", Body: "old body"})

	newBody := "new body content"
	w.UpdateSpec("SPEC-1", UpdateSpecInput{Body: &newBody})

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Body != "new body content" {
		t.Errorf("body = %q", spec.Body)
	}
}

func TestUpdateSpecPreservesLinks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "b"})
	w.Link("SPEC-1", "SPEC-2")

	newTitle := "Updated A"
	w.UpdateSpec("SPEC-1", UpdateSpecInput{Title: &newTitle})

	// Verify link is still in frontmatter.
	spec, _ := w.ReadSpec("SPEC-1")
	data, _ := os.ReadFile(filepath.Join(w.Root, spec.Path))
	if !strings.Contains(string(data), "SPEC-2") {
		t.Error("linked_specs should be preserved after update")
	}
}

func TestUpdateSpecNotFound(t *testing.T) {
	w := setupWorkspace(t)
	newTitle := "x"
	err := w.UpdateSpec("SPEC-999", UpdateSpecInput{Title: &newTitle})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateSpecPartial(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "Original", Type: "technical", Summary: "orig-summary", Body: "orig-body"})

	// Only update title, everything else preserved.
	newTitle := "New Title"
	w.UpdateSpec("SPEC-1", UpdateSpecInput{Title: &newTitle})

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Title != "New Title" {
		t.Errorf("title = %q", spec.Title)
	}
	if spec.Summary != "orig-summary" {
		t.Errorf("summary changed: %q", spec.Summary)
	}
	if spec.Body != "orig-body" {
		t.Errorf("body changed: %q", spec.Body)
	}
}

// --- Spec rename ---

func TestRenameSpec(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "Old Name", Type: "technical", Summary: "s"})

	err := w.RenameSpec("SPEC-1", "Brand New Name")
	if err != nil {
		t.Fatalf("RenameSpec: %v", err)
	}

	spec, _ := w.ReadSpec("SPEC-1")
	if spec.Title != "Brand New Name" {
		t.Errorf("title = %q", spec.Title)
	}
	if spec.Slug != "brand-new-name" {
		t.Errorf("slug = %q", spec.Slug)
	}
	if !strings.Contains(spec.Path, "brand-new-name") {
		t.Errorf("path = %q, should contain new slug", spec.Path)
	}

	// Verify the new directory exists.
	absPath := filepath.Join(w.Root, spec.Path)
	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("new spec file missing: %v", err)
	}
}

func TestRenameSpecUpdatesTaskPaths(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "Spec A", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Task 1", Summary: "t"})

	w.RenameSpec("SPEC-1", "Renamed Spec")

	task, _ := w.ReadTask("TASK-1")
	if !strings.Contains(task.Path, "renamed-spec") {
		t.Errorf("task path = %q, should contain new slug", task.Path)
	}
}

func TestRenameSpecNotFound(t *testing.T) {
	w := setupWorkspace(t)
	err := w.RenameSpec("SPEC-999", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Spec delete ---

func TestDeleteSpec(t *testing.T) {
	w := setupWorkspace(t)
	result, _ := w.NewSpec(NewSpecInput{Title: "To Delete", Type: "technical", Summary: "s"})

	err := w.DeleteSpec("SPEC-1")
	if err != nil {
		t.Fatalf("DeleteSpec: %v", err)
	}

	// Verify gone from DB.
	_, err = w.ReadSpec("SPEC-1")
	if err == nil {
		t.Error("spec should be deleted")
	}

	// Verify directory removed.
	absDir := filepath.Join(w.Root, filepath.Dir(result.Path))
	if _, err := os.Stat(absDir); err == nil {
		t.Error("spec directory should be removed")
	}

	// Verify in trash.
	var count int
	w.DB.QueryRow("SELECT count(*) FROM trash WHERE original_id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Errorf("trash count = %d", count)
	}
}

func TestDeleteSpecCascadesTasks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t"})

	w.DeleteSpec("SPEC-1")

	// Task should be cascade deleted.
	_, err := w.ReadTask("TASK-1")
	if err == nil {
		t.Error("task should be cascade deleted with spec")
	}
}

func TestDeleteSpecNotFound(t *testing.T) {
	w := setupWorkspace(t)
	err := w.DeleteSpec("SPEC-999")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Task update ---

func TestUpdateTaskTitle(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Original", Summary: "s"})

	newTitle := "Updated Task"
	err := w.UpdateTask("TASK-1", UpdateTaskInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	task, _ := w.ReadTask("TASK-1")
	if task.Title != "Updated Task" {
		t.Errorf("title = %q", task.Title)
	}
}

func TestUpdateTaskBody(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s", Body: "old body"})

	newBody := "new body\n\n## Acceptance criteria\n\n- [ ] New criterion"
	w.UpdateTask("TASK-1", UpdateTaskInput{Body: &newBody})

	task, _ := w.ReadTask("TASK-1")
	if task.Body != newBody {
		t.Errorf("body = %q", task.Body)
	}

	// Criteria should be re-synced.
	criteria, _ := w.ListCriteria("TASK-1")
	if len(criteria) != 1 {
		t.Errorf("criteria count = %d, want 1", len(criteria))
	}
}

func TestUpdateTaskPreservesStatus(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s", Status: "todo"})

	newTitle := "Updated"
	w.UpdateTask("TASK-1", UpdateTaskInput{Title: &newTitle})

	task, _ := w.ReadTask("TASK-1")
	if task.Status != "todo" {
		t.Errorf("status = %q, should be preserved", task.Status)
	}
}

func TestUpdateTaskNotFound(t *testing.T) {
	w := setupWorkspace(t)
	newTitle := "x"
	err := w.UpdateTask("TASK-999", UpdateTaskInput{Title: &newTitle})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Task move (status change) ---

func TestMoveTask(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s"})

	err := w.MoveTask("TASK-1", "in_progress")
	if err != nil {
		t.Fatalf("MoveTask: %v", err)
	}

	task, _ := w.ReadTask("TASK-1")
	if task.Status != "in_progress" {
		t.Errorf("status = %q", task.Status)
	}
}

func TestMoveTaskFrontmatterSync(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s"})

	w.MoveTask("TASK-1", "done")

	task, _ := w.ReadTask("TASK-1")
	data, _ := os.ReadFile(filepath.Join(w.Root, task.Path))
	if !strings.Contains(string(data), "status: done") {
		t.Error("frontmatter should reflect new status")
	}
}

func TestMoveTaskNoOp(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s", Status: "todo"})

	// Moving to same status should be no-op.
	err := w.MoveTask("TASK-1", "todo")
	if err != nil {
		t.Fatalf("MoveTask no-op: %v", err)
	}
}

func TestMoveTaskNotFound(t *testing.T) {
	w := setupWorkspace(t)
	err := w.MoveTask("TASK-999", "done")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMoveTaskThroughStatuses(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s"})

	statuses := []string{"todo", "in_progress", "blocked", "pending_verification", "done"}
	for _, s := range statuses {
		if err := w.MoveTask("TASK-1", s); err != nil {
			t.Fatalf("MoveTask to %s: %v", s, err)
		}
		task, _ := w.ReadTask("TASK-1")
		if task.Status != s {
			t.Errorf("after move to %s, status = %q", s, task.Status)
		}
	}
}

// --- Task rename ---

func TestRenameTask(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Old Task", Summary: "s"})

	err := w.RenameTask("TASK-1", "New Task Name")
	if err != nil {
		t.Fatalf("RenameTask: %v", err)
	}

	task, _ := w.ReadTask("TASK-1")
	if task.Title != "New Task Name" {
		t.Errorf("title = %q", task.Title)
	}
	if task.Slug != "new-task-name" {
		t.Errorf("slug = %q", task.Slug)
	}
	if !strings.Contains(task.Path, "new-task-name") {
		t.Errorf("path = %q", task.Path)
	}

	// Verify new file exists.
	absPath := filepath.Join(w.Root, task.Path)
	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("renamed file missing: %v", err)
	}
}

func TestRenameTaskNotFound(t *testing.T) {
	w := setupWorkspace(t)
	err := w.RenameTask("TASK-999", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Task delete ---

func TestDeleteTask(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	result, _ := w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "s"})

	err := w.DeleteTask("TASK-1")
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	_, err = w.ReadTask("TASK-1")
	if err == nil {
		t.Error("task should be deleted")
	}

	absPath := filepath.Join(w.Root, result.Path)
	if _, err := os.Stat(absPath); err == nil {
		t.Error("task file should be removed")
	}

	var count int
	w.DB.QueryRow("SELECT count(*) FROM trash WHERE original_id = 'TASK-1'").Scan(&count)
	if count != 1 {
		t.Errorf("trash count = %d", count)
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	w := setupWorkspace(t)
	err := w.DeleteTask("TASK-999")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteTaskCascadesCriteria(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{
		SpecID: "SPEC-1", Title: "T", Summary: "s",
		Body: "## Acceptance criteria\n\n- [ ] C1\n- [ ] C2",
	})

	w.DeleteTask("TASK-1")

	var count int
	w.DB.QueryRow("SELECT count(*) FROM task_criteria WHERE task_id = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Errorf("criteria count = %d, should be 0 after delete", count)
	}
}

// --- Reorder specs ---

func TestReorderSpecBefore(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "b"})
	w.NewSpec(NewSpecInput{Title: "C", Type: "technical", Summary: "c"})

	// Move C before A.
	err := w.ReorderSpec("SPEC-3", ReorderInput{Mode: ReorderBefore, TargetID: "SPEC-1"})
	if err != nil {
		t.Fatalf("ReorderSpec: %v", err)
	}

	specs, _ := w.ListSpecs(ListSpecsFilter{})
	if specs[0].ID != "SPEC-3" {
		t.Errorf("first spec = %s, want SPEC-3", specs[0].ID)
	}
	if specs[1].ID != "SPEC-1" {
		t.Errorf("second spec = %s, want SPEC-1", specs[1].ID)
	}
}

func TestReorderSpecAfter(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "b"})
	w.NewSpec(NewSpecInput{Title: "C", Type: "technical", Summary: "c"})

	// Move A after B.
	err := w.ReorderSpec("SPEC-1", ReorderInput{Mode: ReorderAfter, TargetID: "SPEC-2"})
	if err != nil {
		t.Fatalf("ReorderSpec: %v", err)
	}

	specs, _ := w.ListSpecs(ListSpecsFilter{})
	if specs[0].ID != "SPEC-2" {
		t.Errorf("first = %s, want SPEC-2", specs[0].ID)
	}
	if specs[1].ID != "SPEC-1" {
		t.Errorf("second = %s, want SPEC-1", specs[1].ID)
	}
}

func TestReorderSpecTo(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "b"})
	w.NewSpec(NewSpecInput{Title: "C", Type: "technical", Summary: "c"})

	// Move C to position 0.
	err := w.ReorderSpec("SPEC-3", ReorderInput{Mode: ReorderTo, Position: 0})
	if err != nil {
		t.Fatalf("ReorderSpec: %v", err)
	}

	specs, _ := w.ListSpecs(ListSpecsFilter{})
	if specs[0].ID != "SPEC-3" {
		t.Errorf("first = %s, want SPEC-3", specs[0].ID)
	}
}

func TestReorderSpecTargetNotFound(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})

	err := w.ReorderSpec("SPEC-1", ReorderInput{Mode: ReorderBefore, TargetID: "SPEC-999"})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

// --- Reorder tasks ---

func TestReorderTaskBefore(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "todo"})

	// Move T3 before T1.
	err := w.ReorderTask("TASK-3", ReorderInput{Mode: ReorderBefore, TargetID: "TASK-1"})
	if err != nil {
		t.Fatalf("ReorderTask: %v", err)
	}

	tasks, _ := w.ListTasks(ListTasksFilter{Status: "todo"})
	if tasks[0].ID != "TASK-3" {
		t.Errorf("first = %s, want TASK-3", tasks[0].ID)
	}
}

func TestReorderTaskAfter(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "todo"})

	// Move T1 after T2.
	w.ReorderTask("TASK-1", ReorderInput{Mode: ReorderAfter, TargetID: "TASK-2"})

	tasks, _ := w.ListTasks(ListTasksFilter{Status: "todo"})
	if tasks[0].ID != "TASK-2" {
		t.Errorf("first = %s, want TASK-2", tasks[0].ID)
	}
	if tasks[1].ID != "TASK-1" {
		t.Errorf("second = %s, want TASK-1", tasks[1].ID)
	}
}

func TestReorderTaskTo(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "todo"})

	// Move T3 to position 1.
	w.ReorderTask("TASK-3", ReorderInput{Mode: ReorderTo, Position: 1})

	tasks, _ := w.ListTasks(ListTasksFilter{Status: "todo"})
	if tasks[1].ID != "TASK-3" {
		t.Errorf("position 1 = %s, want TASK-3", tasks[1].ID)
	}
}

func TestReorderTaskOnlyAffectsSameStatus(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "in_progress"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "todo"})

	// Reorder within todo — should not affect in_progress task.
	w.ReorderTask("TASK-3", ReorderInput{Mode: ReorderTo, Position: 0})

	todoTasks, _ := w.ListTasks(ListTasksFilter{Status: "todo"})
	if len(todoTasks) != 2 {
		t.Errorf("todo count = %d", len(todoTasks))
	}
	if todoTasks[0].ID != "TASK-3" {
		t.Errorf("first todo = %s, want TASK-3", todoTasks[0].ID)
	}

	ipTasks, _ := w.ListTasks(ListTasksFilter{Status: "in_progress"})
	if len(ipTasks) != 1 || ipTasks[0].ID != "TASK-2" {
		t.Error("in_progress task should be unaffected")
	}
}

// --- resolveInsertIndex ---

func TestResolveInsertIndexBefore(t *testing.T) {
	ordered := []string{"A", "B", "C"}
	idx, err := resolveInsertIndex(ordered, ReorderInput{Mode: ReorderBefore, TargetID: "B"}, "X")
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Errorf("idx = %d, want 1", idx)
	}
}

func TestResolveInsertIndexAfter(t *testing.T) {
	ordered := []string{"A", "B", "C"}
	idx, err := resolveInsertIndex(ordered, ReorderInput{Mode: ReorderAfter, TargetID: "B"}, "X")
	if err != nil {
		t.Fatal(err)
	}
	if idx != 2 {
		t.Errorf("idx = %d, want 2", idx)
	}
}

func TestResolveInsertIndexToClamp(t *testing.T) {
	ordered := []string{"A", "B"}
	idx, _ := resolveInsertIndex(ordered, ReorderInput{Mode: ReorderTo, Position: 100}, "X")
	if idx != 2 {
		t.Errorf("idx = %d, want 2 (clamped)", idx)
	}

	idx, _ = resolveInsertIndex(ordered, ReorderInput{Mode: ReorderTo, Position: -5}, "X")
	if idx != 0 {
		t.Errorf("idx = %d, want 0 (clamped)", idx)
	}
}

func TestResolveInsertIndexNotFound(t *testing.T) {
	ordered := []string{"A", "B"}
	_, err := resolveInsertIndex(ordered, ReorderInput{Mode: ReorderBefore, TargetID: "Z"}, "X")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMoveTaskPositionInTargetStatus(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "in_progress"})

	w.MoveTask("TASK-1", "in_progress")

	ipTasks, _ := w.ListTasks(ListTasksFilter{Status: "in_progress"})
	if len(ipTasks) != 2 {
		t.Fatalf("in_progress count = %d, want 2", len(ipTasks))
	}
	if ipTasks[0].ID != "TASK-3" {
		t.Errorf("first in_progress = %s, want TASK-3", ipTasks[0].ID)
	}
	if ipTasks[1].ID != "TASK-1" {
		t.Errorf("second in_progress = %s, want TASK-1", ipTasks[1].ID)
	}
}

func TestDeleteTaskCascadesLinks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2"})
	w.Link("TASK-1", "TASK-2")

	w.DeleteTask("TASK-1")

	var count int
	w.DB.QueryRow("SELECT count(*) FROM task_links WHERE from_task = 'TASK-1' OR to_task = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Errorf("link count = %d, should be 0 after cascade", count)
	}
}

func TestDeleteTaskCascadesDeps(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2"})
	w.Depend("TASK-2", []string{"TASK-1"})

	w.DeleteTask("TASK-1")

	var count int
	w.DB.QueryRow("SELECT count(*) FROM task_dependencies WHERE blocker_task = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Errorf("dep count = %d, should be 0 after cascade", count)
	}
}

func TestDeleteSpecCascadesLinks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "b"})
	w.Link("SPEC-1", "SPEC-2")

	w.DeleteSpec("SPEC-1")

	var count int
	w.DB.QueryRow("SELECT count(*) FROM spec_links WHERE from_spec = 'SPEC-1' OR to_spec = 'SPEC-1'").Scan(&count)
	if count != 0 {
		t.Errorf("link count = %d, should be 0 after cascade", count)
	}
}

func TestDeleteSpecCascadesCitations(t *testing.T) {
	w := setupWithKB(t)
	w.Cite("SPEC-1", []CitationInput{{KBID: "KB-1", ChunkPosition: 0}})

	w.DeleteSpec("SPEC-1")

	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 0 {
		t.Errorf("citation count = %d, should cascade delete", count)
	}
}

func TestDeleteSpecTrashMetadata(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "Trashable", Type: "business", Summary: "s"})

	w.DeleteSpec("SPEC-1")

	var metadata string
	w.DB.QueryRow("SELECT metadata FROM trash WHERE original_id = 'SPEC-1'").Scan(&metadata)
	if !strings.Contains(metadata, "Trashable") {
		t.Errorf("trash metadata missing title: %s", metadata)
	}
	if !strings.Contains(metadata, "business") {
		t.Errorf("trash metadata missing type: %s", metadata)
	}
}

func TestUpdateTaskClearsCriteria(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{
		SpecID: "SPEC-1", Title: "T", Summary: "t",
		Body: "## Acceptance criteria\n\n- [ ] C1\n- [ ] C2",
	})

	criteria, _ := w.ListCriteria("TASK-1")
	if len(criteria) != 2 {
		t.Fatalf("before: %d criteria", len(criteria))
	}

	newBody := "No criteria anymore."
	w.UpdateTask("TASK-1", UpdateTaskInput{Body: &newBody})

	criteria, _ = w.ListCriteria("TASK-1")
	if len(criteria) != 0 {
		t.Errorf("after: %d criteria, want 0", len(criteria))
	}
}
