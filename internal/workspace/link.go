package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
)

// Link creates an undirected link between two items of the same kind.
// Both IDs must be either SPEC-N or TASK-N (no cross-kind linking).
// Updates SQLite and both items' frontmatter.
func (w *Workspace) Link(fromID, toID string) error {
	return w.WithLock(func() error {
		if isSpec(fromID) && isSpec(toID) {
			return w.linkSpecs(fromID, toID)
		}
		if isTask(fromID) && isTask(toID) {
			return w.linkTasks(fromID, toID)
		}
		return fmt.Errorf("cannot link %s to %s: both must be specs or both must be tasks", fromID, toID)
	})
}

// Unlink removes an undirected link between two items of the same kind.
// Updates SQLite and both items' frontmatter.
func (w *Workspace) Unlink(fromID, toID string) error {
	return w.WithLock(func() error {
		if isSpec(fromID) && isSpec(toID) {
			return w.unlinkSpecs(fromID, toID)
		}
		if isTask(fromID) && isTask(toID) {
			return w.unlinkTasks(fromID, toID)
		}
		return fmt.Errorf("cannot unlink %s from %s: both must be specs or both must be tasks", fromID, toID)
	})
}

func (w *Workspace) linkSpecs(a, b string) error {
	// Verify both exist.
	if _, err := w.ReadSpec(a); err != nil {
		return err
	}
	if _, err := w.ReadSpec(b); err != nil {
		return err
	}

	// Insert both directions (undirected).
	_, err := w.DB.Exec(
		"INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)", a, b)
	if err != nil {
		return fmt.Errorf("insert spec link: %w", err)
	}
	_, err = w.DB.Exec(
		"INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)", b, a)
	if err != nil {
		return fmt.Errorf("insert spec link reverse: %w", err)
	}

	// Update frontmatter on both specs.
	if err := w.syncSpecLinks(a); err != nil {
		return err
	}
	return w.syncSpecLinks(b)
}

func (w *Workspace) unlinkSpecs(a, b string) error {
	w.DB.Exec("DELETE FROM spec_links WHERE from_spec = ? AND to_spec = ?", a, b)
	w.DB.Exec("DELETE FROM spec_links WHERE from_spec = ? AND to_spec = ?", b, a)

	if err := w.syncSpecLinks(a); err != nil {
		return err
	}
	return w.syncSpecLinks(b)
}

func (w *Workspace) linkTasks(a, b string) error {
	if _, err := w.ReadTask(a); err != nil {
		return err
	}
	if _, err := w.ReadTask(b); err != nil {
		return err
	}

	_, err := w.DB.Exec(
		"INSERT OR IGNORE INTO task_links (from_task, to_task) VALUES (?, ?)", a, b)
	if err != nil {
		return fmt.Errorf("insert task link: %w", err)
	}
	_, err = w.DB.Exec(
		"INSERT OR IGNORE INTO task_links (from_task, to_task) VALUES (?, ?)", b, a)
	if err != nil {
		return fmt.Errorf("insert task link reverse: %w", err)
	}

	if err := w.syncTaskLinks(a); err != nil {
		return err
	}
	return w.syncTaskLinks(b)
}

func (w *Workspace) unlinkTasks(a, b string) error {
	w.DB.Exec("DELETE FROM task_links WHERE from_task = ? AND to_task = ?", a, b)
	w.DB.Exec("DELETE FROM task_links WHERE from_task = ? AND to_task = ?", b, a)

	if err := w.syncTaskLinks(a); err != nil {
		return err
	}
	return w.syncTaskLinks(b)
}

// syncSpecLinks reads current links from SQLite and updates the spec's
// linked_specs frontmatter field.
func (w *Workspace) syncSpecLinks(specID string) error {
	spec, err := w.ReadSpec(specID)
	if err != nil {
		return err
	}

	links, err := w.getSpecLinks(specID)
	if err != nil {
		return err
	}

	return w.rewriteSpecFrontmatter(spec, func(fm *frontmatter.SpecFrontmatter) {
		fm.LinkedSpecs = links
	})
}

// syncTaskLinks reads current links from SQLite and updates the task's
// linked_tasks frontmatter field.
func (w *Workspace) syncTaskLinks(taskID string) error {
	task, err := w.ReadTask(taskID)
	if err != nil {
		return err
	}

	links, err := w.getTaskLinks(taskID)
	if err != nil {
		return err
	}

	return w.rewriteTaskFrontmatter(task, func(fm *frontmatter.TaskFrontmatter) {
		fm.LinkedTasks = links
	})
}

// getSpecLinks returns the IDs of all specs linked to the given spec.
func (w *Workspace) getSpecLinks(specID string) ([]string, error) {
	rows, err := w.DB.Query(
		"SELECT to_spec FROM spec_links WHERE from_spec = ? ORDER BY to_spec", specID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// getTaskLinks returns the IDs of all tasks linked to the given task.
func (w *Workspace) getTaskLinks(taskID string) ([]string, error) {
	rows, err := w.DB.Query(
		"SELECT to_task FROM task_links WHERE from_task = ? ORDER BY to_task", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// rewriteSpecFrontmatter reads a spec's markdown file, applies a mutation
// function to the frontmatter, and writes it back. Updates content_hash.
func (w *Workspace) rewriteSpecFrontmatter(spec *Spec, mutate func(*frontmatter.SpecFrontmatter)) error {
	absPath := filepath.Join(w.Root, spec.Path)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read spec file: %w", err)
	}

	doc, err := frontmatter.Parse(string(data))
	if err != nil {
		return fmt.Errorf("parse spec file: %w", err)
	}

	fm, err := frontmatter.DecodeSpec(doc.RawFrontmatter)
	if err != nil {
		return err
	}

	mutate(fm)

	content, err := frontmatter.RenderSpec(fm, doc.Body)
	if err != nil {
		return err
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return err
	}

	_, err = w.DB.Exec("UPDATE specs SET content_hash = ? WHERE id = ?",
		hash.String(content), spec.ID)
	return err
}

// rewriteTaskFrontmatter reads a task's markdown file, applies a mutation
// function to the frontmatter, and writes it back. Updates content_hash.
func (w *Workspace) rewriteTaskFrontmatter(task *Task, mutate func(*frontmatter.TaskFrontmatter)) error {
	absPath := filepath.Join(w.Root, task.Path)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read task file: %w", err)
	}

	doc, err := frontmatter.Parse(string(data))
	if err != nil {
		return fmt.Errorf("parse task file: %w", err)
	}

	fm, err := frontmatter.DecodeTask(doc.RawFrontmatter)
	if err != nil {
		return err
	}

	mutate(fm)

	content, err := frontmatter.RenderTask(fm, doc.Body)
	if err != nil {
		return err
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return err
	}

	_, err = w.DB.Exec("UPDATE tasks SET content_hash = ? WHERE id = ?",
		hash.String(content), task.ID)
	return err
}

func isSpec(id string) bool {
	return strings.HasPrefix(id, "SPEC-")
}

func isTask(id string) bool {
	return strings.HasPrefix(id, "TASK-")
}
