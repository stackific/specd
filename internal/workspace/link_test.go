package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkSpecs(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Auth", Type: "technical", Summary: "Auth"})
	w.NewSpec(NewSpecInput{Title: "OAuth", Type: "technical", Summary: "OAuth"})

	if err := w.Link("SPEC-1", "SPEC-2"); err != nil {
		t.Fatalf("Link: %v", err)
	}

	// Verify SQLite has both directions.
	links1, _ := w.getSpecLinks("SPEC-1")
	links2, _ := w.getSpecLinks("SPEC-2")
	if len(links1) != 1 || links1[0] != "SPEC-2" {
		t.Errorf("SPEC-1 links = %v", links1)
	}
	if len(links2) != 1 || links2[0] != "SPEC-1" {
		t.Errorf("SPEC-2 links = %v", links2)
	}

	// Verify frontmatter updated on both files.
	spec1, _ := w.ReadSpec("SPEC-1")
	data, _ := os.ReadFile(filepath.Join(w.Root, spec1.Path))
	if !strings.Contains(string(data), "SPEC-2") {
		t.Error("SPEC-1 frontmatter should contain SPEC-2")
	}
}

func TestUnlinkSpecs(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "A"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "B"})
	w.Link("SPEC-1", "SPEC-2")

	if err := w.Unlink("SPEC-1", "SPEC-2"); err != nil {
		t.Fatalf("Unlink: %v", err)
	}

	links, _ := w.getSpecLinks("SPEC-1")
	if len(links) != 0 {
		t.Errorf("links should be empty, got %v", links)
	}
}

func TestLinkTasks(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T1", Summary: "T1"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T2", Summary: "T2"})

	if err := w.Link("TASK-1", "TASK-2"); err != nil {
		t.Fatalf("Link tasks: %v", err)
	}

	links, _ := w.getTaskLinks("TASK-1")
	if len(links) != 1 || links[0] != "TASK-2" {
		t.Errorf("TASK-1 links = %v", links)
	}
}

func TestLinkCrossKindFails(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "T"})

	err := w.Link("SPEC-1", "TASK-1")
	if err == nil {
		t.Fatal("expected error for cross-kind link")
	}
}

func TestLinkIdempotent(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "A"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "B"})

	w.Link("SPEC-1", "SPEC-2")
	// Second link should not error (INSERT OR IGNORE).
	if err := w.Link("SPEC-1", "SPEC-2"); err != nil {
		t.Fatalf("duplicate Link: %v", err)
	}

	links, _ := w.getSpecLinks("SPEC-1")
	if len(links) != 1 {
		t.Errorf("should have exactly 1 link, got %d", len(links))
	}
}

func TestLinkNonexistentSource(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})

	err := w.Link("SPEC-999", "SPEC-1")
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestLinkNonexistentTarget(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})

	err := w.Link("SPEC-1", "SPEC-999")
	if err == nil {
		t.Fatal("expected error for nonexistent target")
	}
}

func TestUnlinkNonexistentLink(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "A", Type: "technical", Summary: "a"})
	w.NewSpec(NewSpecInput{Title: "B", Type: "technical", Summary: "b"})

	err := w.Unlink("SPEC-1", "SPEC-2")
	if err != nil {
		t.Fatalf("Unlink no-op should not error: %v", err)
	}
}

func TestLinkTaskNonexistent(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "s"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "T", Summary: "t"})

	err := w.Link("TASK-1", "TASK-999")
	if err == nil {
		t.Fatal("expected error for nonexistent task link target")
	}
}
