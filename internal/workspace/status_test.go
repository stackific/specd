package workspace

import (
	"testing"
)

func TestStatusEmptyWorkspace(t *testing.T) {
	w := setupWorkspace(t)

	result, err := w.Status(false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if result.Specs.Total != 0 {
		t.Errorf("specs total = %d", result.Specs.Total)
	}
	if result.Tasks.Total != 0 {
		t.Errorf("tasks total = %d", result.Tasks.Total)
	}
	if result.Tidy.Stale {
		t.Error("fresh workspace should not be stale")
	}
}

func TestStatusWithData(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Business spec", Type: "business", Summary: "a business spec"})
	w.NewSpec(NewSpecInput{Title: "Tech spec", Type: "technical", Summary: "a technical spec"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "task one", Status: "todo"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "task two", Status: "in_progress"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-2", Title: "T3", Summary: "task three", Status: "done"})

	result, err := w.Status(false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if result.Specs.Total != 2 {
		t.Errorf("specs total = %d", result.Specs.Total)
	}
	if result.Specs.Business != 1 {
		t.Errorf("specs business = %d", result.Specs.Business)
	}
	if result.Specs.Technical != 1 {
		t.Errorf("specs technical = %d", result.Specs.Technical)
	}
	if result.Tasks.Total != 3 {
		t.Errorf("tasks total = %d", result.Tasks.Total)
	}
	if result.Tasks.Todo != 1 {
		t.Errorf("tasks todo = %d", result.Tasks.Todo)
	}
	if result.Tasks.InProgress != 1 {
		t.Errorf("tasks in_progress = %d", result.Tasks.InProgress)
	}
	if result.Tasks.Done != 1 {
		t.Errorf("tasks done = %d", result.Tasks.Done)
	}
}

func TestStatusDetailed(t *testing.T) {
	w := setupWorkspace(t)

	result, err := w.Status(true)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if result.Lint == nil {
		t.Error("expected lint results in detailed mode")
	}
}

func TestFormatStatus(t *testing.T) {
	result := &StatusResult{}
	result.Specs.Total = 2
	result.Specs.Business = 1
	result.Specs.Technical = 1
	result.Tasks.Total = 5
	result.Tasks.Done = 3

	out := FormatStatus(result)
	if out == "" {
		t.Error("FormatStatus returned empty string")
	}
}
