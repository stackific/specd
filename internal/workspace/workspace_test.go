package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func setupWorkspace(t *testing.T) *Workspace {
	t.Helper()
	dir := t.TempDir()
	w, err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { w.Close() })
	return w
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	w, err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer w.Close()

	// Verify directories.
	for _, sub := range []string{"specd/specs", "specd/kb", ".specd"} {
		if _, err := os.Stat(filepath.Join(dir, sub)); err != nil {
			t.Errorf("missing %s: %v", sub, err)
		}
	}

	// Verify files.
	for _, f := range []string{"specd/specs/index.md", "specd/specs/log.md", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}

	// Verify .gitignore content.
	data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if got := string(data); got != ".specd/\n" {
		t.Errorf(".gitignore = %q", got)
	}

	// Verify DB is usable.
	ver, err := w.DB.GetMeta("schema_version")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if ver != "1" {
		t.Errorf("schema_version = %q", ver)
	}
}

func TestInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	w, err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("first Init: %v", err)
	}
	w.Close()

	_, err = Init(dir, InitOptions{})
	if err == nil {
		t.Fatal("expected error for duplicate init")
	}
}

func TestInitForce(t *testing.T) {
	dir := t.TempDir()
	w1, err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("first Init: %v", err)
	}
	w1.Close()

	w2, err := Init(dir, InitOptions{Force: true})
	if err != nil {
		t.Fatalf("Init --force: %v", err)
	}
	w2.Close()
}

func TestFindRoot(t *testing.T) {
	dir := t.TempDir()
	w, err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	w.Close()

	// Create a subdirectory.
	sub := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(sub, 0o755)

	root, err := FindRoot(sub)
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	if root != dir {
		t.Errorf("FindRoot = %q, want %q", root, dir)
	}
}

func TestNewSpecAndRead(t *testing.T) {
	w := setupWorkspace(t)

	result, err := w.NewSpec(NewSpecInput{
		Title:   "OAuth with GitHub",
		Type:    "technical",
		Summary: "OAuth flow using GitHub",
		Body:    "# OAuth\n\nBody.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	if result.ID != "SPEC-1" {
		t.Errorf("id = %s", result.ID)
	}

	// Verify file exists.
	absPath := filepath.Join(w.Root, result.Path)
	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("spec file missing: %v", err)
	}

	// Read back.
	spec, err := w.ReadSpec("SPEC-1")
	if err != nil {
		t.Fatalf("ReadSpec: %v", err)
	}
	if spec.Title != "OAuth with GitHub" {
		t.Errorf("title = %q", spec.Title)
	}
	if spec.CreatedBy == "" {
		// user_name might be set from git config; just check it's populated.
		t.Log("created_by is empty (no git user.name configured)")
	}
}

func TestNewTaskAndRead(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.NewSpec(NewSpecInput{
		Title:   "Auth",
		Type:    "technical",
		Summary: "Auth spec",
		Body:    "Body.",
	})
	if err != nil {
		t.Fatalf("NewSpec: %v", err)
	}

	result, err := w.NewTask(NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Design schema",
		Summary: "Design DB schema",
		Body:    "# Schema\n\n## Acceptance criteria\n\n- [ ] Users table\n- [x] Sessions table",
	})
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}

	if result.ID != "TASK-1" {
		t.Errorf("id = %s", result.ID)
	}

	// Read back.
	task, err := w.ReadTask("TASK-1")
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	if task.Status != "backlog" {
		t.Errorf("status = %q", task.Status)
	}
	if task.SpecID != "SPEC-1" {
		t.Errorf("spec_id = %q", task.SpecID)
	}

	// Verify criteria were stored.
	criteria, err := w.ListCriteria("TASK-1")
	if err != nil {
		t.Fatalf("ListCriteria: %v", err)
	}
	if len(criteria) != 2 {
		t.Fatalf("criteria len = %d, want 2", len(criteria))
	}
	if criteria[0].Checked {
		t.Error("first criterion should be unchecked")
	}
	if !criteria[1].Checked {
		t.Error("second criterion should be checked")
	}
}

func TestListSpecs(t *testing.T) {
	w := setupWorkspace(t)

	for _, s := range []NewSpecInput{
		{Title: "Spec A", Type: "technical", Summary: "A"},
		{Title: "Spec B", Type: "business", Summary: "B"},
		{Title: "Spec C", Type: "technical", Summary: "C"},
	} {
		if _, err := w.NewSpec(s); err != nil {
			t.Fatalf("NewSpec: %v", err)
		}
	}

	// List all.
	all, err := w.ListSpecs(ListSpecsFilter{})
	if err != nil {
		t.Fatalf("ListSpecs: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all len = %d", len(all))
	}

	// Filter by type.
	tech, err := w.ListSpecs(ListSpecsFilter{Type: "technical"})
	if err != nil {
		t.Fatalf("ListSpecs type: %v", err)
	}
	if len(tech) != 2 {
		t.Errorf("technical len = %d", len(tech))
	}

	// Limit.
	limited, err := w.ListSpecs(ListSpecsFilter{Limit: 1})
	if err != nil {
		t.Fatalf("ListSpecs limit: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("limited len = %d", len(limited))
	}
}

func TestListTasks(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})

	for _, inp := range []NewTaskInput{
		{SpecID: "SPEC-1", Title: "T1", Summary: "t1", Status: "todo"},
		{SpecID: "SPEC-1", Title: "T2", Summary: "t2", Status: "in_progress"},
		{SpecID: "SPEC-1", Title: "T3", Summary: "t3", Status: "todo"},
	} {
		if _, err := w.NewTask(inp); err != nil {
			t.Fatalf("NewTask: %v", err)
		}
	}

	// Filter by status.
	todo, err := w.ListTasks(ListTasksFilter{Status: "todo"})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(todo) != 2 {
		t.Errorf("todo len = %d", len(todo))
	}

	// Filter by spec.
	bySpec, err := w.ListTasks(ListTasksFilter{SpecID: "SPEC-1"})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(bySpec) != 3 {
		t.Errorf("by spec len = %d", len(bySpec))
	}
}

func TestNewTaskInvalidSpec(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.NewTask(NewTaskInput{
		SpecID:  "SPEC-999",
		Title:   "Bad task",
		Summary: "should fail",
	})
	if err == nil {
		t.Fatal("expected error for invalid spec ID")
	}
}

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"OAuth with GitHub":     "oauth-with-github",
		"Hello, World!":        "hello-world",
		"  spaces  everywhere": "spaces-everywhere",
		"UPPER CASE":           "upper-case",
	}
	for input, want := range tests {
		got := Slugify(input)
		if got != want {
			t.Errorf("Slugify(%q) = %q, want %q", input, got, want)
		}
	}
}
