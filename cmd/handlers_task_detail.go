// handlers_task_detail.go implements the GET /tasks/{id} page handler.
// Renders a single task with its criteria, linked tasks, dependencies, and
// metadata, reusing LoadTaskDetail() so the web page and the get-task CLI
// share one data path.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

// TaskDetailPageData is the view model passed to the task detail template.
type TaskDetailPageData struct {
	Task           *GetTaskResponse
	ParentSpec     *ListSpecItem // nil if the parent spec row is missing
	StatusLabel    string        // human-readable form of Task.Status
	BodyClean      string        // Task.Body with the ## Acceptance Criteria section stripped
	CompletedCount int
	TotalCriteria  int
}

// makeTaskDetailHandler returns an http.HandlerFunc for /tasks/{id}.
func makeTaskDetailHandler(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}

		db, _, err := OpenProjectDB()
		if err != nil {
			slog.Error("task detail: db", "error", err)
			http.Error(w, "database unavailable", http.StatusInternalServerError)
			return
		}
		defer func() { _ = db.Close() }()

		task, err := LoadTaskDetail(db, id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.NotFound(w, r)
				return
			}
			slog.Error("task detail: load", "error", err)
			http.Error(w, "failed to load task", http.StatusInternalServerError)
			return
		}

		parent, err := loadParentSpecSummary(db, task.SpecID)
		if err != nil {
			slog.Error("task detail: parent spec", "error", err)
			http.Error(w, "failed to load parent spec", http.StatusInternalServerError)
			return
		}

		data := TaskDetailPageData{
			Task:          task,
			ParentSpec:    parent,
			StatusLabel:   FromSlug(task.Status),
			BodyClean:     stripCriteriaSection(task.Body),
			TotalCriteria: len(task.Criteria),
		}
		for _, c := range task.Criteria {
			if c.Checked == 1 {
				data.CompletedCount++
			}
		}

		renderPage(w, r, freshPages(), "task_detail", &PageData{
			Title:   task.ID + " — " + task.Title,
			Active:  "tasks",
			DevMode: devMode,
			CSSHash: cssHash,
			JSHash:  jsHash,
			Data:    data,
		})
	}
}

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
