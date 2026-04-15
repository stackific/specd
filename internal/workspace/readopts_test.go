package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

// --- GetSpecLinks ---

func TestGetSpecLinksEmpty(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})

	links, err := w.GetSpecLinks("SPEC-1")
	if err != nil {
		t.Fatalf("GetSpecLinks: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestGetSpecLinksWithLinks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "business", Summary: "b"})
	w.Link("SPEC-1", "SPEC-2")

	links, err := w.GetSpecLinks("SPEC-1")
	if err != nil {
		t.Fatalf("GetSpecLinks: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].ID != "SPEC-2" {
		t.Errorf("link ID = %s", links[0].ID)
	}
	if links[0].Title != "B" {
		t.Errorf("link title = %s", links[0].Title)
	}
}

// --- GetTaskLinks ---

func TestGetTaskLinksEmpty(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t"})

	links, err := w.GetTaskLinks("TASK-1")
	if err != nil {
		t.Fatalf("GetTaskLinks: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestGetTaskLinksWithLinks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2"})
	w.Link("TASK-1", "TASK-2")

	links, err := w.GetTaskLinks("TASK-1")
	if err != nil {
		t.Fatalf("GetTaskLinks: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].ID != "TASK-2" {
		t.Errorf("link ID = %s", links[0].ID)
	}
	if links[0].Status == "" {
		t.Error("link status should not be empty")
	}
}

// --- GetSpecProgress ---

func TestGetSpecProgressNoTasks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})

	p, err := w.GetSpecProgress("SPEC-1")
	if err != nil {
		t.Fatalf("GetSpecProgress: %v", err)
	}
	if p.Total != 0 {
		t.Errorf("total = %d", p.Total)
	}
	if p.Percent != 0 {
		t.Errorf("percent = %f", p.Percent)
	}
}

func TestGetSpecProgressMixed(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "done"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "cancelled"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T4", Summary: "t4", Status: "done"})

	p, err := w.GetSpecProgress("SPEC-1")
	if err != nil {
		t.Fatalf("GetSpecProgress: %v", err)
	}
	if p.Total != 4 {
		t.Errorf("total = %d, want 4", p.Total)
	}
	if p.Done != 2 {
		t.Errorf("done = %d, want 2", p.Done)
	}
	if p.Cancelled != 1 {
		t.Errorf("cancelled = %d, want 1", p.Cancelled)
	}
	if p.Active != 3 { // 4 - 1 cancelled
		t.Errorf("active = %d, want 3", p.Active)
	}
	// 2 done / 3 active = 66.67%
	if p.Percent < 66 || p.Percent > 67 {
		t.Errorf("percent = %f, want ~66.67", p.Percent)
	}
}

// --- GetTaskDeps ---

func TestGetTaskDepsEmpty(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t"})

	deps, err := w.GetTaskDeps("TASK-1")
	if err != nil {
		t.Fatalf("GetTaskDeps: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestGetTaskDepsWithDeps(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocker", Summary: "b", Status: "done"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocked", Summary: "b"})
	w.Depend("TASK-2", []string{"TASK-1"})

	deps, err := w.GetTaskDeps("TASK-2")
	if err != nil {
		t.Fatalf("GetTaskDeps: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].ID != "TASK-1" {
		t.Errorf("dep ID = %s", deps[0].ID)
	}
	if !deps[0].Ready {
		t.Error("dep should be ready (status=done)")
	}
}

func TestGetTaskDepsNotReady(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocker", Summary: "b", Status: "in_progress"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocked", Summary: "b"})
	w.Depend("TASK-2", []string{"TASK-1"})

	deps, _ := w.GetTaskDeps("TASK-2")
	if deps[0].Ready {
		t.Error("dep should NOT be ready (status=in_progress)")
	}
}

// --- ListSpecs --empty ---

func TestListSpecsEmpty(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "With Task", Type: "technical", Summary: "s"})
	w.NewSpec(NewSpecInput{Title: "No Task", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t"})

	// Only specs without tasks.
	empty, err := w.ListSpecs(ListSpecsFilter{Empty: true})
	if err != nil {
		t.Fatalf("ListSpecs empty: %v", err)
	}
	if len(empty) != 1 {
		t.Fatalf("expected 1 empty spec, got %d", len(empty))
	}
	if empty[0].ID != "SPEC-2" {
		t.Errorf("empty spec = %s, want SPEC-2", empty[0].ID)
	}
}

func TestListSpecsEmptyAllHaveTasks(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t"})

	empty, _ := w.ListSpecs(ListSpecsFilter{Empty: true})
	if len(empty) != 0 {
		t.Errorf("expected 0 empty specs, got %d", len(empty))
	}
}

// --- NewSpec returns candidates ---

func TestNewSpecReturnsCandidates(t *testing.T) {
	w := setupWorkspace(t)

	// Add a KB doc and an existing spec so candidates have something to find.
	md := filepath.Join(w.Root, "auth.md")
	os.WriteFile(md, []byte("OAuth authentication flow tokens."), 0o644)
	w.KBAdd(KBAddInput{Source: md, Title: "Auth Guide"})
	w.NewSpec(NewSpecInput{Title: "User Authentication", Type: "technical", Summary: "Auth system"})

	// Create a related spec.
	result, err := w.NewSpec(NewSpecInput{
		Title:   "OAuth with GitHub",
		Type:    "technical",
		Summary: "OAuth flow using GitHub as identity provider",
		Body:    "OAuth authentication implementation.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}
	if result.Candidates == nil {
		t.Fatal("candidates should not be nil")
	}
	// Should find at least the existing spec as a candidate.
	t.Logf("spec candidates: %d, task candidates: %d, kb candidates: %d",
		len(result.Candidates.Specs), len(result.Candidates.Tasks), len(result.Candidates.KBChunks))
}

// --- NewTask returns candidates ---

func TestGetSpecProgressWontfix(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "done"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "wontfix"})

	p, _ := w.GetSpecProgress("SPEC-1")
	if p.WontFix != 1 {
		t.Errorf("wontfix = %d", p.WontFix)
	}
	if p.Active != 1 {
		t.Errorf("active = %d, want 1", p.Active)
	}
	if p.Percent != 100 {
		t.Errorf("percent = %f, want 100", p.Percent)
	}
}

func TestGetTaskDepsReadyStatuses(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Done", Summary: "d", Status: "done"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Cancelled", Summary: "c", Status: "cancelled"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Wontfix", Summary: "w", Status: "wontfix"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Blocked", Summary: "b"})

	w.Depend("TASK-4", []string{"TASK-1", "TASK-2", "TASK-3"})

	deps, _ := w.GetTaskDeps("TASK-4")
	if len(deps) != 3 {
		t.Fatalf("deps = %d, want 3", len(deps))
	}
	for _, d := range deps {
		if !d.Ready {
			t.Errorf("dep %s (status=%s) should be ready", d.ID, d.Status)
		}
	}
}

func TestNewTaskReturnsCandidates(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "Auth", Type: "technical", Summary: "Auth"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Design schema", Summary: "DB schema"})

	result, err := w.NewTask(NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Implement schema migration",
		Summary: "Run the DB schema migration",
	})
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	if result.Candidates == nil {
		t.Fatal("candidates should not be nil")
	}
	t.Logf("task candidates: %d", len(result.Candidates.Tasks))
}
