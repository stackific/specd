// handlers_task_criteria.go implements POST /api/tasks/{id}/criteria/{position}/toggle.
// Flips the checked state of one acceptance criterion in the database, rewrites
// the TASK-N.md markdown file from DB state (so the file remains ground truth),
// and re-renders the criteria article partial for htmx to swap into the page.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// makeTaskCriteriaToggleHandler returns the http.HandlerFunc for the toggle
// endpoint. It validates the path values, flips the criterion atomically in
// SQL, rewrites the markdown file, and renders the criteria-article partial.
func makeTaskCriteriaToggleHandler(freshPages func() map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID, pos, ok := parseToggleParams(r)
		if !ok {
			http.Error(w, "invalid task id or criterion position", http.StatusBadRequest)
			return
		}
		toggleAndRender(w, r, freshPages, taskID, pos)
	}
}

// parseToggleParams pulls and validates the {id} + {position} path values.
func parseToggleParams(r *http.Request) (taskID string, position int, ok bool) {
	taskID = strings.ToUpper(r.PathValue("id"))
	if !strings.HasPrefix(taskID, "TASK-") {
		return "", 0, false
	}
	pos, err := strconv.Atoi(r.PathValue("position"))
	if err != nil || pos < 1 {
		return "", 0, false
	}
	return taskID, pos, true
}

// toggleAndRender flips the criterion, rewrites the markdown, and renders the
// criteria-article partial. Split out from the handler closure to keep its
// cognitive complexity within the linter's threshold.
func toggleAndRender(w http.ResponseWriter, r *http.Request, freshPages func() map[string]*template.Template, taskID string, pos int) {
	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("task criteria toggle: db", "error", err)
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}
	defer func() { _ = db.Close() }()

	if err := flipTaskCriterion(db, taskID, pos); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("task criteria toggle: flip", "error", err)
		http.Error(w, "failed to toggle criterion", http.StatusInternalServerError)
		return
	}

	if err := rewriteTaskFile(db, taskID); err != nil {
		slog.Error("task criteria toggle: rewrite", "error", err)
		http.Error(w, "failed to rewrite task file", http.StatusInternalServerError)
		return
	}

	data, err := buildCriteriaPartialData(db, taskID)
	if err != nil {
		slog.Error("task criteria toggle: reload", "error", err)
		http.Error(w, "failed to reload task", http.StatusInternalServerError)
		return
	}

	tmpl, ok := freshPages()["task_detail"]
	if !ok {
		http.Error(w, "task_detail template missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "task-criteria-article", data); err != nil {
		slog.Error("task criteria toggle: render", "error", err)
	}
}

// buildCriteriaPartialData reloads the task and packages the fields the
// criteria-article partial expects.
func buildCriteriaPartialData(db *sql.DB, taskID string) (TaskDetailPageData, error) {
	task, err := LoadTaskDetail(db, taskID)
	if err != nil {
		return TaskDetailPageData{}, err
	}
	data := TaskDetailPageData{
		Task:          task,
		TotalCriteria: len(task.Criteria),
	}
	for _, c := range task.Criteria {
		if c.Checked == 1 {
			data.CompletedCount++
		}
	}
	return data, nil
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
