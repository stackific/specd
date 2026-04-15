package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTaskWithCriteria(t *testing.T) (*Workspace, string) {
	t.Helper()
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{
		Title:   "Auth",
		Type:    "technical",
		Summary: "Auth spec",
		Body:    "Body.",
	})

	result, err := w.NewTask(NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Schema design",
		Summary: "Design it",
		Body:    "# Schema\n\n## Acceptance criteria\n\n- [ ] Users table\n- [ ] Sessions table\n- [x] Migrations",
	})
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	return w, result.ID
}

func readTaskFile(t *testing.T, w *Workspace, taskID string) string {
	t.Helper()
	task, err := w.ReadTask(taskID)
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(w.Root, task.Path))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(data)
}

func TestCheckCriterionRoundTrip(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	// Check criterion 1.
	if err := w.CheckCriterion(taskID, 1); err != nil {
		t.Fatalf("CheckCriterion: %v", err)
	}

	// Verify SQLite.
	criteria, _ := w.ListCriteria(taskID)
	if !criteria[0].Checked {
		t.Error("criterion 1 should be checked in SQLite")
	}

	// Verify markdown.
	content := readTaskFile(t, w, taskID)
	if !strings.Contains(content, "- [x] Users table") {
		t.Errorf("markdown should contain checked criterion, got:\n%s", content)
	}
}

func TestUncheckCriterionRoundTrip(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	// Criterion 3 is already checked. Uncheck it.
	if err := w.UncheckCriterion(taskID, 3); err != nil {
		t.Fatalf("UncheckCriterion: %v", err)
	}

	criteria, _ := w.ListCriteria(taskID)
	if criteria[2].Checked {
		t.Error("criterion 3 should be unchecked in SQLite")
	}

	content := readTaskFile(t, w, taskID)
	if strings.Contains(content, "- [x] Migrations") {
		t.Error("markdown should have unchecked Migrations")
	}
	if !strings.Contains(content, "- [ ] Migrations") {
		t.Error("markdown should contain unchecked Migrations")
	}
}

func TestAddCriterionRoundTrip(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	c, err := w.AddCriterion(taskID, "Rollback support")
	if err != nil {
		t.Fatalf("AddCriterion: %v", err)
	}
	if c.Position != 4 {
		t.Errorf("position = %d, want 4", c.Position)
	}

	criteria, _ := w.ListCriteria(taskID)
	if len(criteria) != 4 {
		t.Fatalf("criteria count = %d, want 4", len(criteria))
	}

	content := readTaskFile(t, w, taskID)
	if !strings.Contains(content, "- [ ] Rollback support") {
		t.Error("markdown should contain new criterion")
	}
}

func TestRemoveCriterionRoundTrip(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	// Remove criterion 2 (Sessions table).
	if err := w.RemoveCriterion(taskID, 2); err != nil {
		t.Fatalf("RemoveCriterion: %v", err)
	}

	criteria, _ := w.ListCriteria(taskID)
	if len(criteria) != 2 {
		t.Fatalf("criteria count = %d, want 2", len(criteria))
	}

	// Verify renumbering: old position 3 (Migrations) should now be position 2.
	if criteria[1].Text != "Migrations" || criteria[1].Position != 2 {
		t.Errorf("renumbered criterion = %+v", criteria[1])
	}

	content := readTaskFile(t, w, taskID)
	if strings.Contains(content, "Sessions table") {
		t.Error("markdown should not contain removed criterion")
	}
}

func TestCheckCriterionNotFound(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	err := w.CheckCriterion(taskID, 99)
	if err == nil {
		t.Fatal("expected error for nonexistent criterion")
	}
}

func TestRemoveCriterionNotFound(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	err := w.RemoveCriterion(taskID, 99)
	if err == nil {
		t.Fatal("expected error for nonexistent criterion")
	}
}

func TestAddCriterionToTaskWithoutSection(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{
		Title:   "S",
		Type:    "technical",
		Summary: "s",
		Body:    "Body.",
	})

	result, _ := w.NewTask(NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "No criteria",
		Summary: "Task with no criteria section",
		Body:    "# Just a body\n\nNo criteria here.",
	})

	_, err := w.AddCriterion(result.ID, "New criterion")
	if err != nil {
		t.Fatalf("AddCriterion: %v", err)
	}

	content := readTaskFile(t, w, result.ID)
	if !strings.Contains(content, "## Acceptance criteria") {
		t.Error("should have added acceptance criteria section")
	}
	if !strings.Contains(content, "- [ ] New criterion") {
		t.Error("should contain the new criterion")
	}
}

func TestReplaceCriteriaSection(t *testing.T) {
	input := "---\ntitle: Test\n---\n\n# Body\n\n## Acceptance criteria\n\n- [ ] Old\n\n## Other section\n\nMore content.\n"
	newSection := "## Acceptance criteria\n\n- [x] New\n"

	result := replaceCriteriaSection(input, newSection)

	if !strings.Contains(result, "- [x] New") {
		t.Error("should contain new criterion")
	}
	if strings.Contains(result, "- [ ] Old") {
		t.Error("should not contain old criterion")
	}
	if !strings.Contains(result, "## Other section") {
		t.Error("should preserve other sections")
	}
	if !strings.Contains(result, "More content.") {
		t.Error("should preserve content after criteria section")
	}
}

func TestListCriteriaEmpty(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t", Body: "No criteria here."})

	criteria, err := w.ListCriteria("TASK-1")
	if err != nil {
		t.Fatalf("ListCriteria: %v", err)
	}
	if len(criteria) != 0 {
		t.Errorf("expected 0 criteria, got %d", len(criteria))
	}
}

func TestCheckCriterionIdempotent(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	w.CheckCriterion(taskID, 1)
	err := w.CheckCriterion(taskID, 1)
	if err != nil {
		t.Fatalf("idempotent check should not error: %v", err)
	}

	criteria, _ := w.ListCriteria(taskID)
	if !criteria[0].Checked {
		t.Error("should still be checked")
	}
}

func TestUncheckCriterionIdempotent(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	err := w.UncheckCriterion(taskID, 1) // already unchecked
	if err != nil {
		t.Fatalf("idempotent uncheck should not error: %v", err)
	}
}

func TestRemoveCriterionRenumbers(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{
		SpecID: "SPEC-1", Title: "T", Summary: "t",
		Body: "## Acceptance criteria\n\n- [ ] First\n- [ ] Second\n- [ ] Third",
	})

	w.RemoveCriterion("TASK-1", 2)

	criteria, _ := w.ListCriteria("TASK-1")
	if len(criteria) != 2 {
		t.Fatalf("criteria count = %d, want 2", len(criteria))
	}
	if criteria[0].Text != "First" || criteria[0].Position != 1 {
		t.Errorf("first: %+v", criteria[0])
	}
	if criteria[1].Text != "Third" || criteria[1].Position != 2 {
		t.Errorf("second (renumbered): %+v", criteria[1])
	}
}

func TestAddCriterionDuplicate(t *testing.T) {
	w, taskID := setupTaskWithCriteria(t)

	before, _ := w.ListCriteria(taskID)
	countBefore := len(before)

	_, err := w.AddCriterion(taskID, "Users table")
	if err != nil {
		t.Fatalf("AddCriterion duplicate text should work: %v", err)
	}

	after, _ := w.ListCriteria(taskID)
	if len(after) != countBefore+1 {
		t.Errorf("criteria count = %d, want %d", len(after), countBefore+1)
	}
}
