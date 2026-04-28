// loaders_tasks.go contains task detail/criteria data loaders shared by the
// JSON API. The HTML page handlers were removed in the SPA migration.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// LoadTaskDetail hydrates a GetTaskResponse for the given ID, including
// criteria, links, and dependencies. Returns an error wrapping
// sql.ErrNoRows if the task does not exist.
func LoadTaskDetail(db *sql.DB, taskID string) (*GetTaskResponse, error) {
	var resp GetTaskResponse
	var updatedBy *string
	err := db.QueryRow(`
		SELECT id, spec_id, title, status, summary, body, path, position,
		       created_by, updated_by, content_hash, created_at, updated_at
		FROM tasks WHERE id = ?`, taskID).Scan(
		&resp.ID, &resp.SpecID, &resp.Title, &resp.Status, &resp.Summary,
		&resp.Body, &resp.Path, &resp.Position,
		&resp.CreatedBy, &updatedBy, &resp.ContentHash,
		&resp.CreatedAt, &resp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task %s not found: %w", taskID, err)
		}
		return nil, fmt.Errorf("loading task %s: %w", taskID, err)
	}
	if updatedBy != nil {
		resp.UpdatedBy = *updatedBy
	}
	if resp.LinkedTasks, err = loadLinkedTasks(db, taskID); err != nil {
		return nil, err
	}
	if resp.DependsOn, err = loadTaskDependsOn(db, taskID); err != nil {
		return nil, err
	}
	if resp.Criteria, err = loadGetTaskCriteria(db, taskID); err != nil {
		return nil, err
	}
	return &resp, nil
}

// loadParentSpecSummary returns the parent spec for a breadcrumb. Returns
// (nil, nil) if the spec row is missing — sync should keep the foreign key in
// step, but handlers shouldn't crash on lag.
func loadParentSpecSummary(db *sql.DB, specID string) (*ListSpecItem, error) {
	var s ListSpecItem
	err := db.QueryRow(`
		SELECT id, title, type, summary
		FROM specs WHERE id = ?`, specID).Scan(&s.ID, &s.Title, &s.Type, &s.Summary)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // intentional: missing parent renders without a breadcrumb link
	}
	if err != nil {
		return nil, fmt.Errorf("loading parent spec %s: %w", specID, err)
	}
	return &s, nil
}

// stripCriteriaSection prepares a task body for rendering on the detail page:
//
//   - Removes the leading `# Title` H1. The page header already renders the
//     title; leaving it in the body would duplicate it as a huge heading.
//   - Removes the `## Acceptance Criteria` heading and the bullets that follow
//     it. The detail page renders criteria as proper checkboxes elsewhere;
//     leaving the raw markdown would duplicate them after goldmark runs.
//   - Removes any `## Description` heading line. The card itself communicates
//     the section, so the heading is redundant.
func stripCriteriaSection(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var kept []string
	skipping := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			// Drop the H1 title line.
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			skipping = strings.EqualFold(heading, "Acceptance Criteria")
			if skipping {
				continue
			}
			if strings.EqualFold(heading, "Description") {
				continue
			}
		}
		if !skipping {
			kept = append(kept, line)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// parseToggleParams pulls and validates the {id} + {position} path values.
func parseToggleParams(r *http.Request) (taskID string, position int, ok bool) {
	taskID = strings.ToUpper(r.PathValue("id"))
	if !strings.HasPrefix(taskID, IDPrefixTask) {
		return "", 0, false
	}
	pos, err := strconv.Atoi(r.PathValue("position"))
	if err != nil || pos < 1 {
		return "", 0, false
	}
	return taskID, pos, true
}

// flipTaskCriterion reads the current checked state for (taskID, position),
// flips it, and writes it back. Returns sql.ErrNoRows if the row doesn't exist.
func flipTaskCriterion(db *sql.DB, taskID string, position int) error {
	var current int
	err := db.QueryRow(
		"SELECT checked FROM task_criteria WHERE task_id = ? AND position = ?",
		taskID, position,
	).Scan(&current)
	if err != nil {
		return err
	}

	next := 1 - current
	username := ResolveActiveUsername()
	var checkedBy *string
	if next == 1 && username != "" {
		checkedBy = &username
	}

	res, err := db.Exec(
		`UPDATE task_criteria SET checked = ?, checked_by = ? WHERE task_id = ? AND position = ?`,
		next, checkedBy, taskID, position,
	)
	if err != nil {
		return fmt.Errorf("updating criterion: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
