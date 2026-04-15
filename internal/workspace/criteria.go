// Package workspace — criteria.go implements acceptance criteria CRUD.
// Criteria are parsed from the body and stored in SQLite with markdown sync.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackific/specd/internal/hash"
)

// CriterionRow represents a stored acceptance criterion.
type CriterionRow struct {
	TaskID   string `json:"task_id"`
	Position int    `json:"position"`
	Text     string `json:"text"`
	Checked  bool   `json:"checked"`
}

// ListCriteria returns all acceptance criteria for a task, ordered by position.
func (w *Workspace) ListCriteria(taskID string) ([]CriterionRow, error) {
	rows, err := w.DB.Query(
		"SELECT task_id, position, text, checked FROM task_criteria WHERE task_id = ? ORDER BY position",
		taskID)
	if err != nil {
		return nil, fmt.Errorf("list criteria: %w", err)
	}
	defer rows.Close()

	var criteria []CriterionRow
	for rows.Next() {
		var c CriterionRow
		var checked int
		if err := rows.Scan(&c.TaskID, &c.Position, &c.Text, &checked); err != nil {
			return nil, err
		}
		c.Checked = checked == 1
		criteria = append(criteria, c)
	}
	return criteria, rows.Err()
}

// AddCriterion appends a new unchecked criterion to the task.
// Updates both SQLite and the markdown file.
func (w *Workspace) AddCriterion(taskID, text string) (*CriterionRow, error) {
	var result *CriterionRow

	err := w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		// Find the next position.
		var maxPos int
		err = w.DB.QueryRow(
			"SELECT COALESCE(MAX(position), 0) FROM task_criteria WHERE task_id = ?",
			taskID).Scan(&maxPos)
		if err != nil {
			return err
		}
		pos := maxPos + 1

		// Insert into SQLite.
		_, err = w.DB.Exec(
			"INSERT INTO task_criteria (task_id, position, text, checked) VALUES (?, ?, ?, 0)",
			taskID, pos, text)
		if err != nil {
			return fmt.Errorf("insert criterion: %w", err)
		}

		// Update the markdown file.
		if err := w.rewriteTaskCriteria(task); err != nil {
			return err
		}

		result = &CriterionRow{TaskID: taskID, Position: pos, Text: text, Checked: false}
		return nil
	})

	return result, err
}

// CheckCriterion marks a criterion as checked (done).
// Updates both SQLite and the markdown file.
func (w *Workspace) CheckCriterion(taskID string, position int) error {
	return w.setCriterionChecked(taskID, position, true)
}

// UncheckCriterion marks a criterion as unchecked.
// Updates both SQLite and the markdown file.
func (w *Workspace) UncheckCriterion(taskID string, position int) error {
	return w.setCriterionChecked(taskID, position, false)
}

// setCriterionChecked sets the checked state of a criterion and rewrites the markdown.
func (w *Workspace) setCriterionChecked(taskID string, position int, checked bool) error {
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		checkedInt := 0
		if checked {
			checkedInt = 1
		}

		res, err := w.DB.Exec(
			"UPDATE task_criteria SET checked = ? WHERE task_id = ? AND position = ?",
			checkedInt, taskID, position)
		if err != nil {
			return fmt.Errorf("update criterion: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("criterion %d not found on %s", position, taskID)
		}

		return w.rewriteTaskCriteria(task)
	})
}

// RemoveCriterion deletes a criterion and renumbers subsequent positions.
// Updates both SQLite and the markdown file.
func (w *Workspace) RemoveCriterion(taskID string, position int) error {
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		res, err := w.DB.Exec(
			"DELETE FROM task_criteria WHERE task_id = ? AND position = ?",
			taskID, position)
		if err != nil {
			return fmt.Errorf("delete criterion: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("criterion %d not found on %s", position, taskID)
		}

		// Renumber positions to stay contiguous.
		_, err = w.DB.Exec(
			"UPDATE task_criteria SET position = position - 1 WHERE task_id = ? AND position > ?",
			taskID, position)
		if err != nil {
			return fmt.Errorf("renumber criteria: %w", err)
		}

		return w.rewriteTaskCriteria(task)
	})
}

// rewriteTaskCriteria reads current criteria from SQLite and rewrites the
// "## Acceptance criteria" section in the task's markdown file, then updates
// the content hash in SQLite.
func (w *Workspace) rewriteTaskCriteria(task *Task) error {
	absPath := filepath.Join(w.Root, task.Path)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read task file: %w", err)
	}
	content := string(data)

	// Load current criteria from SQLite.
	criteria, err := w.ListCriteria(task.ID)
	if err != nil {
		return err
	}

	// Build the new criteria section.
	var section strings.Builder
	section.WriteString("## Acceptance criteria\n\n")
	for _, c := range criteria {
		if c.Checked {
			section.WriteString("- [x] " + c.Text + "\n")
		} else {
			section.WriteString("- [ ] " + c.Text + "\n")
		}
	}

	// Replace the existing section, or append it.
	newContent := replaceCriteriaSection(content, section.String())

	if err := os.WriteFile(absPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("write task file: %w", err)
	}

	// Update content hash so the watcher doesn't treat this as an external edit.
	contentHash := hash.String(newContent)
	_, err = w.DB.Exec(
		"UPDATE tasks SET content_hash = ? WHERE id = ?",
		contentHash, task.ID)
	return err
}

// replaceCriteriaSection replaces the "## Acceptance criteria" section in
// content with newSection. If the section doesn't exist, it appends it.
func replaceCriteriaSection(content, newSection string) string {
	const header = "## Acceptance criteria"
	idx := strings.Index(content, header)
	if idx < 0 {
		// Append the section.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + newSection
	}

	before := content[:idx]

	// Find the end of the section: next ## heading or end of file.
	rest := content[idx+len(header):]
	// Skip past the header line itself.
	if nlIdx := strings.IndexByte(rest, '\n'); nlIdx >= 0 {
		rest = rest[nlIdx+1:]
	}

	endIdx := -1
	for i := 0; i < len(rest); {
		nlPos := strings.IndexByte(rest[i:], '\n')
		var lineStart int
		if i == 0 {
			lineStart = 0
		} else {
			lineStart = i
		}

		if strings.HasPrefix(rest[lineStart:], "## ") {
			endIdx = lineStart
			break
		}

		if nlPos < 0 {
			break
		}
		i = i + nlPos + 1
	}

	if endIdx >= 0 {
		return before + newSection + rest[endIdx:]
	}
	return before + newSection
}
