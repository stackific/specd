package workspace

import (
	"database/sql"
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
	ID   string `json:"id"`
	Path string `json:"path"`
}

// NewTask creates a new task under the given spec.
func (w *Workspace) NewTask(input NewTaskInput) (*NewTaskResult, error) {
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

	return result, err
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
