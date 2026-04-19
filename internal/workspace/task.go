// Package workspace — task.go implements task lifecycle operations: create,
// read, list, update, move (status change), rename, and soft-delete. Tasks
// belong to a parent spec and are ordered per-status column for the kanban
// board. Acceptance criteria are parsed from the body and synced on update.
package workspace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
)

// Task is the domain representation of a task.
type Task struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	SpecID    string `json:"spec_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	Position  int    `json:"position"`
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// NewTaskInput holds the parameters for creating a new task.
type NewTaskInput struct {
	SpecID  string
	Title   string
	Summary string
	Body    string
	Status  string // defaults to "backlog"
}

// NewTaskResult is the JSON response from new-task.
type NewTaskResult struct {
	ID         string            `json:"id"`
	Path       string            `json:"path"`
	Candidates *CandidatesResult `json:"candidates,omitempty"`
}

// NewTask creates a new task under the given spec.
func (w *Workspace) NewTask(input NewTaskInput) (*NewTaskResult, error) {
	if err := validateNoH1(input.Body); err != nil {
		return nil, err
	}

	var result *NewTaskResult

	if input.Status == "" {
		input.Status = "backlog"
	}

	err := w.WithLock(func() error {
		// Verify parent spec exists.
		spec, err := w.ReadSpec(input.SpecID)
		if err != nil {
			return fmt.Errorf("parent spec: %w", err)
		}

		id, err := w.DB.NextID("task")
		if err != nil {
			return fmt.Errorf("allocate task id: %w", err)
		}

		taskID := fmt.Sprintf("TASK-%d", id)
		slug := Slugify(input.Title)
		fileName := fmt.Sprintf("%s-%s.md", taskID, slug)

		// Task file lives in the parent spec's directory.
		specDir := filepath.Dir(spec.Path)
		relPath := filepath.Join(specDir, fileName)

		// Render markdown file.
		fm := &frontmatter.TaskFrontmatter{
			Title:   input.Title,
			Status:  input.Status,
			Summary: input.Summary,
		}
		content, err := frontmatter.RenderTask(fm, input.Body)
		if err != nil {
			return fmt.Errorf("render task: %w", err)
		}

		absPath := filepath.Join(w.Root, relPath)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write task file: %w", err)
		}

		contentHash := hash.String(content)
		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		// Get max position for this status.
		var maxPos int
		err = w.DB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM tasks WHERE status = ?",
			input.Status).Scan(&maxPos)
		if err != nil {
			return err
		}

		_, err = w.DB.Exec(`INSERT INTO tasks (id, slug, spec_id, title, status, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			taskID, slug, input.SpecID, input.Title, input.Status, input.Summary, input.Body,
			relPath, maxPos+1, userName, userName, contentHash, now, now)
		if err != nil {
			return fmt.Errorf("insert task: %w", err)
		}

		// Parse and insert acceptance criteria if body contains them.
		criteria := frontmatter.ParseCriteria(input.Body)
		for i, c := range criteria {
			checked := 0
			if c.Checked {
				checked = 1
			}
			_, err := w.DB.Exec(`INSERT INTO task_criteria (task_id, position, text, checked)
				VALUES (?, ?, ?, ?)`, taskID, i+1, c.Text, checked)
			if err != nil {
				return fmt.Errorf("insert criterion: %w", err)
			}
		}

		result = &NewTaskResult{ID: taskID, Path: relPath}
		return nil
	})

	// Compute candidates outside the lock (read-only).
	if err == nil && result != nil {
		candidates, _ := w.Candidates(result.ID, 20)
		result.Candidates = candidates
	}

	return result, err
}

// UpdateTaskInput holds optional fields for updating a task.
type UpdateTaskInput struct {
	Title   *string
	Summary *string
	Body    *string
}

// UpdateTask updates mutable fields on a task.
func (w *Workspace) UpdateTask(taskID string, input UpdateTaskInput) error {
	if input.Body != nil {
		if err := validateNoH1(*input.Body); err != nil {
			return err
		}
	}
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		title := task.Title
		summary := task.Summary
		body := task.Body

		if input.Title != nil {
			title = *input.Title
		}
		if input.Summary != nil {
			summary = *input.Summary
		}
		if input.Body != nil {
			body = *input.Body
		}

		// Preserve system-managed frontmatter fields.
		absPath := filepath.Join(w.Root, task.Path)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read task file: %w", err)
		}
		doc, err := frontmatter.Parse(string(data))
		if err != nil {
			return fmt.Errorf("parse task: %w", err)
		}
		existingFM, err := frontmatter.DecodeTask(doc.RawFrontmatter)
		if err != nil {
			return err
		}

		fm := &frontmatter.TaskFrontmatter{
			Title:       title,
			Status:      task.Status,
			Summary:     summary,
			LinkedTasks: existingFM.LinkedTasks,
			DependsOn:   existingFM.DependsOn,
			Cites:       existingFM.Cites,
		}

		content, err := frontmatter.RenderTask(fm, body)
		if err != nil {
			return err
		}

		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return err
		}

		contentHash := hash.String(content)
		_, err = w.DB.Exec(`UPDATE tasks SET title=?, summary=?, body=?,
			updated_by=?, content_hash=?, updated_at=? WHERE id=?`,
			title, summary, body, userName, contentHash, now, taskID)
		if err != nil {
			return err
		}

		// Re-sync criteria if body changed.
		if input.Body != nil {
			criteria := frontmatter.ParseCriteria(body)
			w.DB.Exec("DELETE FROM task_criteria WHERE task_id = ?", taskID)
			for i, c := range criteria {
				checked := 0
				if c.Checked {
					checked = 1
				}
				w.DB.Exec(`INSERT INTO task_criteria (task_id, position, text, checked)
					VALUES (?, ?, ?, ?)`, taskID, i+1, c.Text, checked)
			}
		}

		return nil
	})
}

// MoveTask changes a task's status. Updates the position to the end of
// the target status column.
func (w *Workspace) MoveTask(taskID string, newStatus string) error {
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		if task.Status == newStatus {
			return nil // no-op
		}

		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		// Get max position in the target status.
		var maxPos int
		w.DB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM tasks WHERE status = ?",
			newStatus).Scan(&maxPos)

		// Update status in DB.
		_, err = w.DB.Exec(`UPDATE tasks SET status=?, position=?,
			updated_by=?, updated_at=? WHERE id=?`,
			newStatus, maxPos+1, userName, now, taskID)
		if err != nil {
			return fmt.Errorf("move task: %w", err)
		}

		// Update frontmatter.
		return w.rewriteTaskFrontmatter(task, func(fm *frontmatter.TaskFrontmatter) {
			fm.Status = newStatus
		})
	})
}

// RenameTask changes a task's title and updates its slug and filename.
func (w *Workspace) RenameTask(taskID string, newTitle string) error {
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		newSlug := Slugify(newTitle)
		newFileName := fmt.Sprintf("%s-%s.md", taskID, newSlug)
		taskDir := filepath.Dir(task.Path)
		newRelPath := filepath.Join(taskDir, newFileName)

		oldAbs := filepath.Join(w.Root, task.Path)
		newAbs := filepath.Join(w.Root, newRelPath)

		if err := os.Rename(oldAbs, newAbs); err != nil {
			return fmt.Errorf("rename task file: %w", err)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		// Rewrite frontmatter with new title.
		data, err := os.ReadFile(newAbs)
		if err != nil {
			return err
		}
		doc, err := frontmatter.Parse(string(data))
		if err != nil {
			return err
		}
		fm, err := frontmatter.DecodeTask(doc.RawFrontmatter)
		if err != nil {
			return err
		}
		fm.Title = newTitle
		content, err := frontmatter.RenderTask(fm, doc.Body)
		if err != nil {
			return err
		}
		if err := os.WriteFile(newAbs, []byte(content), 0o644); err != nil {
			return err
		}

		contentHash := hash.String(content)
		_, err = w.DB.Exec(`UPDATE tasks SET title=?, slug=?, path=?,
			updated_by=?, content_hash=?, updated_at=? WHERE id=?`,
			newTitle, newSlug, newRelPath, userName, contentHash, now, taskID)
		return err
	})
}

// DeleteTask soft-deletes a task to trash.
func (w *Workspace) DeleteTask(taskID string) error {
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		absPath := filepath.Join(w.Root, task.Path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read task for trash: %w", err)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		metaBytes, _ := json.Marshal(map[string]string{
			"id": task.ID, "title": task.Title, "spec_id": task.SpecID,
			"status": task.Status, "path": task.Path,
		})
		metadata := string(metaBytes)

		tx, err := w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
			VALUES ('task', ?, ?, ?, ?, ?, 'cli')`,
			task.ID, task.Path, content, metadata, now)
		if err != nil {
			return fmt.Errorf("insert task trash: %w", err)
		}

		// Delete citations referencing this task (not FK-cascaded).
		_, err = tx.Exec("DELETE FROM citations WHERE from_kind = 'task' AND from_id = ?", taskID)
		if err != nil {
			return fmt.Errorf("delete task citations: %w", err)
		}

		_, err = tx.Exec("DELETE FROM tasks WHERE id = ?", taskID)
		if err != nil {
			return fmt.Errorf("delete task: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		os.Remove(absPath)
		return nil
	})
}

// ReadTask reads a task by ID.
func (w *Workspace) ReadTask(taskID string) (*Task, error) {
	t := &Task{}
	err := w.DB.QueryRow(`SELECT id, slug, spec_id, title, status, summary, body, path, position,
		COALESCE(created_by, ''), COALESCE(updated_by, ''), created_at, updated_at
		FROM tasks WHERE id = ?`, taskID).Scan(
		&t.ID, &t.Slug, &t.SpecID, &t.Title, &t.Status, &t.Summary, &t.Body, &t.Path, &t.Position,
		&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("read task: %w", err)
	}
	return t, nil
}

// ListTasksFilter holds filter options for listing tasks.
type ListTasksFilter struct {
	SpecID    string
	Status    string
	LinkedTo  string
	DependsOn string
	CreatedBy string
	Limit     int
}

// ListTasks returns tasks matching the filter.
func (w *Workspace) ListTasks(filter ListTasksFilter) ([]Task, error) {
	query := `SELECT id, slug, spec_id, title, status, summary, body, path, position,
		COALESCE(created_by, ''), COALESCE(updated_by, ''), created_at, updated_at
		FROM tasks WHERE 1=1`
	var args []any

	if filter.SpecID != "" {
		query += " AND spec_id = ?"
		args = append(args, filter.SpecID)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.CreatedBy != "" {
		query += " AND created_by = ?"
		args = append(args, filter.CreatedBy)
	}
	if filter.LinkedTo != "" {
		query += ` AND id IN (
			SELECT to_task FROM task_links WHERE from_task = ?
			UNION SELECT from_task FROM task_links WHERE to_task = ?)`
		args = append(args, filter.LinkedTo, filter.LinkedTo)
	}
	if filter.DependsOn != "" {
		query += ` AND id IN (
			SELECT blocked_task FROM task_dependencies WHERE blocker_task = ?)`
		args = append(args, filter.DependsOn)
	}

	query += " ORDER BY position ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := w.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Slug, &t.SpecID, &t.Title, &t.Status, &t.Summary, &t.Body,
			&t.Path, &t.Position, &t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}
