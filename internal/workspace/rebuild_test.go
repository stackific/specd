package workspace

import (
	"os"
	"testing"
)

func TestRebuildEmptyWorkspace(t *testing.T) {
	w := setupWorkspace(t)

	result, err := w.Rebuild(false)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	if result.Specs != 0 {
		t.Errorf("specs = %d", result.Specs)
	}
	if result.Tasks != 0 {
		t.Errorf("tasks = %d", result.Tasks)
	}
}

func TestRebuildWithSpecsAndTasks(t *testing.T) {
	w := setupWorkspace(t)

	// Create some data.
	w.NewSpec(NewSpecInput{Title: "Auth", Type: "functional", Summary: "auth spec"})
	w.NewSpec(NewSpecInput{Title: "Billing", Type: "business", Summary: "billing spec"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Design schema", Summary: "design db schema", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Implement JWT", Summary: "jwt implementation", Status: "in_progress"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-2", Title: "Payment flow", Summary: "payment integration", Status: "backlog"})

	// Rebuild.
	result, err := w.Rebuild(false)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	if result.Specs != 2 {
		t.Errorf("rebuilt specs = %d, want 2", result.Specs)
	}
	if result.Tasks != 3 {
		t.Errorf("rebuilt tasks = %d, want 3", result.Tasks)
	}

	// Verify data is accessible.
	spec, err := w.ReadSpec("SPEC-1")
	if err != nil {
		t.Fatalf("ReadSpec after rebuild: %v", err)
	}
	if spec.Title != "Auth" {
		t.Errorf("spec title = %q", spec.Title)
	}

	task, err := w.ReadTask("TASK-1")
	if err != nil {
		t.Fatalf("ReadTask after rebuild: %v", err)
	}
	if task.Title != "Design schema" {
		t.Errorf("task title = %q", task.Title)
	}

	// Verify counters are set correctly.
	nextSpec, _ := w.DB.GetMeta("next_spec_id")
	if nextSpec != "3" {
		t.Errorf("next_spec_id = %s, want 3", nextSpec)
	}
	nextTask, _ := w.DB.GetMeta("next_task_id")
	if nextTask != "4" {
		t.Errorf("next_task_id = %s, want 4", nextTask)
	}
}

func TestRebuildPreservesUserName(t *testing.T) {
	w := setupWorkspace(t)

	w.DB.SetMeta("user_name", "TestUser")
	w.NewSpec(NewSpecInput{Title: "S", Type: "functional", Summary: "test spec"})

	_, err := w.Rebuild(false)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	name, err := w.DB.GetMeta("user_name")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if name != "TestUser" {
		t.Errorf("user_name = %q, want TestUser", name)
	}
}

func TestRebuildWithKBDocs(t *testing.T) {
	w := setupWorkspace(t)

	// Add a KB doc.
	kbContent := "# Test Document\n\nParagraph one about authentication.\n\nParagraph two about authorization."
	tmpFile := t.TempDir() + "/test.md"
	writeTestFile(t, tmpFile, kbContent)

	_, err := w.KBAdd(KBAddInput{Source: tmpFile, Title: "Test Doc"})
	if err != nil {
		t.Fatalf("KBAdd: %v", err)
	}

	// Rebuild.
	result, err := w.Rebuild(false)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	if result.KBDocs != 1 {
		t.Errorf("rebuilt KB docs = %d, want 1", result.KBDocs)
	}
	if result.KBChunks == 0 {
		t.Error("expected at least 1 rebuilt KB chunk")
	}

	nextKB, _ := w.DB.GetMeta("next_kb_id")
	if nextKB != "2" {
		t.Errorf("next_kb_id = %s, want 2", nextKB)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
