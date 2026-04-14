package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDependAndFrontmatter(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "T1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "T2"})

	if err := w.Depend("TASK-2", []string{"TASK-1"}); err != nil {
		t.Fatalf("Depend: %v", err)
	}

	deps, _ := w.getTaskDependencies("TASK-2")
	if len(deps) != 1 || deps[0] != "TASK-1" {
		t.Errorf("deps = %v", deps)
	}

	// Verify frontmatter.
	task, _ := w.ReadTask("TASK-2")
	data, _ := os.ReadFile(filepath.Join(w.Root, task.Path))
	if !strings.Contains(string(data), "TASK-1") {
		t.Error("frontmatter should contain depends_on TASK-1")
	}
}

func TestUndepend(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "T1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "T2"})
	w.Depend("TASK-2", []string{"TASK-1"})

	if err := w.Undepend("TASK-2", []string{"TASK-1"}); err != nil {
		t.Fatalf("Undepend: %v", err)
	}

	deps, _ := w.getTaskDependencies("TASK-2")
	if len(deps) != 0 {
		t.Errorf("deps should be empty, got %v", deps)
	}
}

func TestDependCycleDetection(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "T1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "T2"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "T3"})

	// T2 depends on T1, T3 depends on T2.
	w.Depend("TASK-2", []string{"TASK-1"})
	w.Depend("TASK-3", []string{"TASK-2"})

	// T1 depending on T3 would create a cycle: T1 -> T3 -> T2 -> T1.
	err := w.Depend("TASK-1", []string{"TASK-3"})
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

func TestDependSelfCycle(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "T1"})

	err := w.Depend("TASK-1", []string{"TASK-1"})
	if err == nil {
		t.Fatal("expected cycle detection for self-dependency")
	}
}
