package workspace

import (
	"strings"
	"testing"
)

func TestNextBasicOrdering(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "todo"})

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(result.Tasks))
	}

	// All should be ready with no deps.
	for _, item := range result.Tasks {
		if !item.Ready {
			t.Errorf("%s should be ready", item.ID)
		}
		if len(item.BlockedBy) != 0 {
			t.Errorf("%s should have empty blocked_by", item.ID)
		}
	}
}

func TestNextReadyBeforeNotReady(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocker", Summary: "b", Status: "in_progress"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocked", Summary: "blocked", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Free", Summary: "free", Status: "todo"})

	// TASK-2 depends on TASK-1 (in_progress, not ready).
	w.Depend("TASK-2", []string{"TASK-1"})

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 2 {
		t.Fatalf("got %d tasks, want 2 (only todo)", len(result.Tasks))
	}

	// TASK-3 (free, ready) should come before TASK-2 (blocked).
	if result.Tasks[0].ID != "TASK-3" {
		t.Errorf("first task = %s, want TASK-3 (ready)", result.Tasks[0].ID)
	}
	if result.Tasks[1].ID != "TASK-2" {
		t.Errorf("second task = %s, want TASK-2 (blocked)", result.Tasks[1].ID)
	}
	if result.Tasks[1].Ready {
		t.Error("TASK-2 should not be ready")
	}
	if len(result.Tasks[1].BlockedBy) != 1 || result.Tasks[1].BlockedBy[0] != "TASK-1" {
		t.Errorf("blocked_by = %v, want [TASK-1]", result.Tasks[1].BlockedBy)
	}
}

func TestNextPartiallyDoneBeforeFresh(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	body := "## Acceptance criteria\n\n- [ ] First\n- [ ] Second\n"
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Fresh", Summary: "fresh", Status: "todo", Body: body})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Partial", Summary: "partial", Status: "todo", Body: body})

	// Check one criterion on TASK-2.
	w.CheckCriterion("TASK-2", 1)

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(result.Tasks))
	}

	// TASK-2 (partially done) should come before TASK-1 (fresh).
	if result.Tasks[0].ID != "TASK-2" {
		t.Errorf("first = %s, want TASK-2 (partially done)", result.Tasks[0].ID)
	}
	if !result.Tasks[0].PartiallyDone {
		t.Error("TASK-2 should be partially_done")
	}
	if result.Tasks[0].CriteriaProgress != 0.5 {
		t.Errorf("progress = %f, want 0.5", result.Tasks[0].CriteriaProgress)
	}
}

func TestNextHigherProgressFirst(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	body3 := "## Acceptance criteria\n\n- [ ] A\n- [ ] B\n- [ ] C\n"
	body2 := "## Acceptance criteria\n\n- [ ] A\n- [ ] B\n"

	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Low", Summary: "low", Status: "todo", Body: body3})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "High", Summary: "high", Status: "todo", Body: body2})

	// TASK-1: check 1/3 = 33%
	w.CheckCriterion("TASK-1", 1)
	// TASK-2: check 1/2 = 50%
	w.CheckCriterion("TASK-2", 1)

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	// TASK-2 (50%) should come before TASK-1 (33%).
	if result.Tasks[0].ID != "TASK-2" {
		t.Errorf("first = %s, want TASK-2 (higher progress)", result.Tasks[0].ID)
	}
}

func TestNextFilterBySpec(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S1", Type: "technical", Summary: "s1"})
	w.NewSpec(NewSpecInput{Title: "S2", Type: "technical", Summary: "s2"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-2", Title: "T2", Summary: "t2", Status: "todo"})

	result, err := w.Next("SPEC-1", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(result.Tasks))
	}
	if result.Tasks[0].ID != "TASK-1" {
		t.Errorf("got %s, want TASK-1", result.Tasks[0].ID)
	}
}

func TestNextLimit(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	for i := 0; i < 5; i++ {
		w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t", Status: "todo"})
	}

	result, err := w.Next("", 3)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Errorf("got %d tasks, want 3", len(result.Tasks))
	}
}

func TestNextOnlyTodoTasks(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Backlog", Summary: "b", Status: "backlog"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "InProg", Summary: "ip", Status: "in_progress"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Todo", Summary: "todo", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Done", Summary: "d", Status: "done"})

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(result.Tasks))
	}
	if result.Tasks[0].ID != "TASK-3" {
		t.Errorf("got %s, want TASK-3", result.Tasks[0].ID)
	}
}

func TestNextReadyWhenDepDone(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Dep", Summary: "dep", Status: "done"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Work", Summary: "work", Status: "todo"})
	w.Depend("TASK-2", []string{"TASK-1"})

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(result.Tasks))
	}
	if !result.Tasks[0].Ready {
		t.Error("TASK-2 should be ready (dep is done)")
	}
}

func TestNextNoCriteria(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t", Status: "todo"})

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	item := result.Tasks[0]
	if item.PartiallyDone {
		t.Error("should not be partially_done with no criteria")
	}
	if item.CriteriaProgress != 0 {
		t.Errorf("progress = %f, want 0", item.CriteriaProgress)
	}
}

func TestNextEmpty(t *testing.T) {
	w := setupWorkspace(t)

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(result.Tasks) != 0 {
		t.Errorf("got %d tasks, want 0", len(result.Tasks))
	}
}

func TestNextDepCycleError(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "A", Summary: "a", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "B", Summary: "b", Status: "todo"})

	w.Depend("TASK-2", []string{"TASK-1"})

	// Force a cycle by inserting directly into the DB (bypassing cycle check).
	w.DB.Exec("INSERT INTO task_dependencies (blocker_task, blocked_task) VALUES ('TASK-2', 'TASK-1')")

	_, err := w.Next("", 10)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

func TestNextComplexSorting(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	body := "## Acceptance criteria\n\n- [ ] A\n- [ ] B\n- [ ] C\n- [ ] D\n"

	// Create tasks in specific order to test all 4 sorting levels.
	// TASK-1: blocker, in_progress (not a todo)
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocker", Summary: "b", Status: "in_progress"})
	// TASK-2: todo, blocked by TASK-1, no criteria (not ready, fresh)
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocked", Summary: "blocked", Status: "todo"})
	// TASK-3: todo, ready, fresh (no criteria)
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "ReadyFresh", Summary: "rf", Status: "todo"})
	// TASK-4: todo, ready, partially done (75%)
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "ReadyHigh", Summary: "rh", Status: "todo", Body: body})
	// TASK-5: todo, ready, partially done (25%)
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "ReadyLow", Summary: "rl", Status: "todo", Body: body})

	w.Depend("TASK-2", []string{"TASK-1"})

	// TASK-4: check 3/4 = 75%
	w.CheckCriterion("TASK-4", 1)
	w.CheckCriterion("TASK-4", 2)
	w.CheckCriterion("TASK-4", 3)

	// TASK-5: check 1/4 = 25%
	w.CheckCriterion("TASK-5", 1)

	result, err := w.Next("", 10)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	// Expected order:
	// 1. TASK-4 (ready, partially done, 75%)
	// 2. TASK-5 (ready, partially done, 25%)
	// 3. TASK-3 (ready, fresh, 0%)
	// 4. TASK-2 (not ready)
	expected := []string{"TASK-4", "TASK-5", "TASK-3", "TASK-2"}
	if len(result.Tasks) != len(expected) {
		t.Fatalf("got %d tasks, want %d", len(result.Tasks), len(expected))
	}
	for i, want := range expected {
		if result.Tasks[i].ID != want {
			t.Errorf("position %d: got %s, want %s", i, result.Tasks[i].ID, want)
		}
	}
}
