// handlers_tasks_board.go implements the kanban board API used by the /tasks
// page. Two endpoints are exposed:
//
//	GET  /api/tasks/board   — renders the board partial (one column per
//	                           configured task stage, cards ordered by
//	                           position).
//	POST /api/tasks/move    — moves a task to a new column / index, renumbers
//	                           positions in the affected columns, and rewrites
//	                           every affected TASK-N.md so the markdown files
//	                           remain ground truth.
//
// Column ordering: Backlog, Todo, In progress, then (when enabled) Blocked,
// Pending Verification, Done, then any remaining configured stages in their
// configured order.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// boardColumn is a single kanban column rendered into the partial.
type boardColumn struct {
	Status string      // slug stored in tasks.status
	Label  string      // display label
	Tasks  []boardCard // ordered by position
}

// boardCard is one card on the board.
type boardCard struct {
	ID       string
	SpecID   string
	Title    string
	Summary  string
	Position int
}

// boardData is the template data for the board partial.
type boardData struct {
	Columns []boardColumn
}

// preferredColumnOrder is the user-requested kanban column ordering. Stages
// not present in the project's configured task stages are skipped. Any
// configured stages not listed here are appended in their configured order.
var preferredColumnOrder = []string{
	"Backlog",
	"Todo",
	"In progress",
	"Blocked",
	"Pending Verification",
	"Done",
}

// Filter values for the kanban board.
const (
	BoardFilterAll        = "all"
	BoardFilterIncomplete = "incomplete"
)

// completedStageSlugs returns the set of stage slugs considered "completed"
// for the current project — i.e. every stage at or after Done in the kanban
// column order. Done is a required stage (see RequiredTaskStages), so this
// set always contains at least one entry. Optional stages (e.g. Cancelled,
// Wont Fix) are included when they appear after Done in the order, and skipped
// automatically when the user opts out of them at init time. No hardcoded
// optional-stage slugs.
func completedStageSlugs(stages []string) map[string]bool {
	completed := make(map[string]bool)
	doneSlug := ToSlug("Done")
	found := false
	for _, label := range stages {
		slug := ToSlug(label)
		if slug == doneSlug {
			found = true
		}
		if found {
			completed[slug] = true
		}
	}
	return completed
}

// orderedStages returns the configured task stage labels in kanban order.
func orderedStages(configured []string) []string {
	have := make(map[string]bool, len(configured))
	for _, s := range configured {
		have[ToSlug(s)] = true
	}

	var ordered []string
	used := make(map[string]bool, len(configured))
	for _, label := range preferredColumnOrder {
		slug := ToSlug(label)
		if have[slug] {
			ordered = append(ordered, label)
			used[slug] = true
		}
	}
	for _, slug := range configured {
		if !used[ToSlug(slug)] {
			ordered = append(ordered, FromSlug(ToSlug(slug)))
		}
	}
	return ordered
}

// loadBoard reads every task from the DB and groups them by status into the
// configured column order. When filter == BoardFilterIncomplete, tasks whose
// status is at or after Done in the kanban order are omitted, but the
// columns themselves remain visible (rendered empty) so the board layout is
// stable across filter toggles. "Completed" is derived positionally from the
// configured stages, not from a hardcoded slug list — projects that opt out
// of Cancelled or Wont Fix at init time still work correctly.
func loadBoard(db *sql.DB, configured []string, filter string) (*boardData, error) {
	stages := orderedStages(configured)
	hideCompletedTasks := filter == BoardFilterIncomplete
	completed := completedStageSlugs(stages)

	bySlug := make(map[string]*boardColumn, len(stages))
	out := &boardData{Columns: make([]boardColumn, 0, len(stages))}
	for _, label := range stages {
		out.Columns = append(out.Columns, boardColumn{
			Status: ToSlug(label),
			Label:  label,
		})
	}
	for i := range out.Columns {
		bySlug[out.Columns[i].Status] = &out.Columns[i]
	}

	rows, err := db.Query(`
		SELECT id, spec_id, title, summary, status, position
		FROM tasks
		ORDER BY status, position, id`)
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var c boardCard
		var status string
		if err := rows.Scan(&c.ID, &c.SpecID, &c.Title, &c.Summary, &status, &c.Position); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		col, ok := bySlug[status]
		if !ok {
			// Task whose status is not in the configured stages — skip silently.
			continue
		}
		if hideCompletedTasks && completed[status] {
			continue
		}
		col.Tasks = append(col.Tasks, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tasks: %w", err)
	}

	return out, nil
}

// renderBoard executes the "board" template into w.
func renderBoard(w http.ResponseWriter, pages map[string]*template.Template, data *boardData) {
	// Any page template clones the shared partials; "tasks" is guaranteed
	// to exist since the page is registered in serve.go.
	tmpl, ok := pages["tasks"]
	if !ok {
		http.Error(w, "tasks template missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "board", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleGetTasksBoard handles GET /api/tasks/board.
func handleGetTasksBoard(pages map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, _, err := OpenProjectDB()
		if err != nil {
			http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = db.Close() }()

		proj, err := LoadProjectConfig(".")
		if err != nil || proj == nil {
			http.Error(w, "project not initialized", http.StatusBadRequest)
			return
		}

		data, err := loadBoard(db, proj.TaskStages, normalizeBoardFilter(r.URL.Query().Get("filter")))
		if err != nil {
			http.Error(w, "loading board: "+err.Error(), http.StatusInternalServerError)
			return
		}

		renderBoard(w, pages, data)
	}
}

// normalizeBoardFilter coerces a query/form filter value to a known constant.
func normalizeBoardFilter(s string) string {
	if s == BoardFilterIncomplete {
		return BoardFilterIncomplete
	}
	return BoardFilterAll
}

// MaxMoveBodyBytes caps the size of the move request body. Three short fields,
// 1 KB is more than enough.
const MaxMoveBodyBytes = 1024

// moveRequest is the parsed form input of POST /api/tasks/move.
type moveRequest struct {
	ID       string
	Status   string
	Position int
	Filter   string // current board filter so the re-render mirrors the client view
}

// parseMoveRequest reads and validates form input.
func parseMoveRequest(r *http.Request) (*moveRequest, int, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, MaxMoveBodyBytes)
	if err := r.ParseForm(); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("bad form: %w", err)
	}
	id := r.FormValue("id")
	status := r.FormValue("status")
	posStr := r.FormValue("position")
	if id == "" || status == "" || posStr == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("id, status, and position are required")
	}
	pos, err := strconv.Atoi(posStr)
	if err != nil || pos < 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("position must be a non-negative integer")
	}
	return &moveRequest{
		ID:       id,
		Status:   status,
		Position: pos,
		Filter:   normalizeBoardFilter(r.FormValue("filter")),
	}, 0, nil
}

// validStatus reports whether the requested status slug is a configured stage.
func validStatus(stages []string, slug string) bool {
	for _, label := range stages {
		if ToSlug(label) == slug {
			return true
		}
	}
	return false
}

// handleMoveTask handles POST /api/tasks/move. The request body is form-encoded
// with: id (TASK-N), status (slug), position (0-based index in destination).
func handleMoveTask(pages map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, code, err := parseMoveRequest(r)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		performMove(w, pages, req)
	}
}

// performMove executes the validated move request: opens the DB, validates the
// status against project config, applies the move, rewrites affected files,
// and re-renders the board.
func performMove(w http.ResponseWriter, pages map[string]*template.Template, req *moveRequest) {
	db, _, err := OpenProjectDB()
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = db.Close() }()

	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		http.Error(w, "project not initialized", http.StatusBadRequest)
		return
	}
	if !validStatus(proj.TaskStages, req.Status) {
		http.Error(w, "unknown status: "+req.Status, http.StatusBadRequest)
		return
	}

	moved, err := moveTask(db, req.ID, req.Status, req.Position)
	if err != nil {
		writeMoveError(w, err)
		return
	}

	rewriteMovedFiles(w, db, moved)

	data, err := loadBoard(db, proj.TaskStages, req.Filter)
	if err != nil {
		http.Error(w, "loading board: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderBoard(w, pages, data)
}

// writeMoveError maps a moveTask error to an HTTP error response.
func writeMoveError(w http.ResponseWriter, err error) {
	if errors.Is(err, errTaskNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	http.Error(w, "moving task: "+err.Error(), http.StatusInternalServerError)
}

// rewriteMovedFiles rewrites every affected task's markdown file so the
// frontmatter on disk (status, position, updated_at) stays in sync with the
// DB. IDs come from the DB and are not user input.
func rewriteMovedFiles(w http.ResponseWriter, db *sql.DB, moved []string) {
	for _, id := range moved {
		if err := rewriteTaskFile(db, id); err != nil {
			slog.Error("rewriting task file after move", "id", id, "error", err) //nolint:gosec // id is a DB-derived TASK-N identifier, not user input
			w.Header().Set("X-Specd-Warning", "could not rewrite "+id+".md")
		}
	}
}

var errTaskNotFound = errors.New("task not found")

// moveTask updates a task's status and renumbers positions in both the
// source and destination columns. Returns the IDs of every task whose
// position or status changed (so callers can rewrite their .md files).
func moveTask(db *sql.DB, taskID, newStatus string, newPos int) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var oldStatus string
	if err := tx.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID).Scan(&oldStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errTaskNotFound
		}
		return nil, fmt.Errorf("reading task: %w", err)
	}

	// Update the moved task's status first.
	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()
	if oldStatus != newStatus {
		if _, err := tx.Exec(
			"UPDATE tasks SET status = ?, updated_by = ?, updated_at = ? WHERE id = ?",
			newStatus, username, now, taskID,
		); err != nil {
			return nil, fmt.Errorf("updating status: %w", err)
		}
	}

	// Renumber the destination column. Pull all task IDs (excluding the
	// moved task), insert the moved task at newPos, then write 0..n-1.
	destIDs, err := taskIDsByStatus(tx, newStatus, taskID)
	if err != nil {
		return nil, err
	}
	if newPos > len(destIDs) {
		newPos = len(destIDs)
	}
	destIDs = append(destIDs[:newPos], append([]string{taskID}, destIDs[newPos:]...)...)

	changed := make(map[string]struct{})
	if err := writePositions(tx, destIDs, changed); err != nil {
		return nil, err
	}

	// If the task moved between columns, also renumber the source column.
	if oldStatus != newStatus {
		srcIDs, err := taskIDsByStatus(tx, oldStatus, "")
		if err != nil {
			return nil, err
		}
		if err := writePositions(tx, srcIDs, changed); err != nil {
			return nil, err
		}
	}

	// The moved task itself always counts as changed.
	changed[taskID] = struct{}{}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	out := make([]string, 0, len(changed))
	for id := range changed {
		out = append(out, id)
	}
	return out, nil
}

// taskIDsByStatus returns task IDs in the given column ordered by position.
// If excludeID is non-empty, that ID is omitted from the result.
func taskIDsByStatus(tx *sql.Tx, status, excludeID string) ([]string, error) {
	rows, err := tx.Query(
		"SELECT id FROM tasks WHERE status = ? ORDER BY position, id",
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("querying status %s: %w", status, err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning id: %w", err)
		}
		if id == excludeID {
			continue
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// writePositions assigns dense positions 0..n-1 to ids in order. Records
// every id whose position changed in the changed set.
func writePositions(tx *sql.Tx, ids []string, changed map[string]struct{}) error {
	for i, id := range ids {
		var current int
		if err := tx.QueryRow("SELECT position FROM tasks WHERE id = ?", id).Scan(&current); err != nil {
			return fmt.Errorf("reading position for %s: %w", id, err)
		}
		if current == i {
			continue
		}
		if _, err := tx.Exec("UPDATE tasks SET position = ? WHERE id = ?", i, id); err != nil {
			return fmt.Errorf("updating position for %s: %w", id, err)
		}
		changed[id] = struct{}{}
	}
	return nil
}
