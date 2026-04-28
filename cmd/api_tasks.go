// api_tasks.go implements every /api/tasks/* endpoint: list, kanban board,
// detail, move (kanban DnD), criterion toggle, depends_on replacement, and
// delete. Tasks are the only resource the SPA mutates, so most of this file
// is mutation handlers — each one rewrites the affected markdown file before
// returning so disk and DB stay in sync.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// ---------------------------------------------------------------------------
// GET /api/tasks
// ---------------------------------------------------------------------------

// apiTasksResponse is the payload returned by GET /api/tasks.
type apiTasksResponse struct {
	Items      []ListTaskItem `json:"items"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
}

// apiListTasksHandler implements GET /api/tasks. Reuses the buildTaskFilters
// helper from list_tasks.go so the SPA and CLI share one filtering path.
func apiListTasksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	specID := q.Get("spec_id")
	status := q.Get("status")
	page := parsePositiveIntParam(q.Get("page"), 1)
	pageSize := parsePositiveIntParam(q.Get("page_size"), DefaultPageSize)

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api tasks: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	where, args := buildTaskFilters(specID, status)

	var total int
	countSQL := "SELECT COUNT(*) FROM tasks" + where //nolint:gosec // where uses hardcoded column names
	if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		slog.Error("api tasks: count", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to count tasks")
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	offset := (page - 1) * pageSize

	querySQL := "SELECT id, spec_id, title, status, summary, position, created_at, updated_at FROM tasks" + where + " ORDER BY position, id LIMIT ? OFFSET ?" //nolint:gosec // where built from hardcoded columns
	queryArgs := append(args, pageSize, offset)                                                                                                               //nolint:gocritic // intentional copy

	rows, err := db.Query(querySQL, queryArgs...)
	if err != nil {
		slog.Error("api tasks: query", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	defer func() { _ = rows.Close() }()

	items := []ListTaskItem{}
	for rows.Next() {
		var t ListTaskItem
		if err := rows.Scan(&t.ID, &t.SpecID, &t.Title, &t.Status, &t.Summary, &t.Position, &t.CreatedAt, &t.UpdatedAt); err != nil {
			slog.Error("api tasks: scan", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to scan task")
			return
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		slog.Error("api tasks: iterate", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to iterate tasks")
		return
	}

	writeJSON(w, http.StatusOK, apiTasksResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
		TotalPages: totalPages,
	})
}

// ---------------------------------------------------------------------------
// GET /api/tasks/board
// ---------------------------------------------------------------------------

// apiBoardCard is a single kanban card in JSON form.
type apiBoardCard struct {
	ID       string `json:"id"`
	SpecID   string `json:"spec_id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Position int    `json:"position"`
}

// apiBoardColumn is one kanban column in JSON form.
type apiBoardColumn struct {
	Status string         `json:"status"`
	Label  string         `json:"label"`
	Tasks  []apiBoardCard `json:"tasks"`
}

// apiBoardResponse is the payload returned by GET /api/tasks/board and the
// POST /api/tasks/move success path.
type apiBoardResponse struct {
	Filter  string           `json:"filter"`
	Stages  []string         `json:"stages"`
	Columns []apiBoardColumn `json:"columns"`
}

// boardJSONFromData converts the internal boardData into the JSON payload.
// Extracted so both the read endpoint and the move endpoint can reuse it.
func boardJSONFromData(data *boardData, filter string) apiBoardResponse {
	stages := make([]string, 0, len(data.Columns))
	cols := make([]apiBoardColumn, 0, len(data.Columns))
	for _, c := range data.Columns {
		stages = append(stages, c.Status)
		cards := make([]apiBoardCard, 0, len(c.Tasks))
		for _, t := range c.Tasks {
			cards = append(cards, apiBoardCard(t))
		}
		cols = append(cols, apiBoardColumn{
			Status: c.Status,
			Label:  c.Label,
			Tasks:  cards,
		})
	}
	return apiBoardResponse{
		Filter:  filter,
		Stages:  stages,
		Columns: cols,
	}
}

// apiBoardHandler implements GET /api/tasks/board.
func apiBoardHandler(w http.ResponseWriter, r *http.Request) {
	filter := normalizeBoardFilter(r.URL.Query().Get("filter"))
	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api board: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		writeJSONError(w, http.StatusBadRequest, "project not initialized")
		return
	}

	data, err := loadBoard(db, proj.TaskStages, filter)
	if err != nil {
		slog.Error("api board: load", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load board")
		return
	}
	writeJSON(w, http.StatusOK, boardJSONFromData(data, filter))
}

// ---------------------------------------------------------------------------
// GET /api/tasks/{id}
// ---------------------------------------------------------------------------

// apiTaskRef is a {id, title, summary} triple used for resolved task
// references in the detail payload (depends_on / linked_tasks). The SPA
// renders all three so the user sees titles + a summary snippet, not raw
// IDs. Summary is included so the depends-on UI can echo the same
// title-plus-snippet shape as the spec detail's child tasks list, no extra
// fetch needed.
type apiTaskRef struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// apiTaskDetailResponse is the payload returned by GET /api/tasks/{id}.
type apiTaskDetailResponse struct {
	Task           *GetTaskResponse `json:"task"`
	ParentSpec     *ListSpecItem    `json:"parent_spec"`
	StatusLabel    string           `json:"status_label"`
	BodyClean      string           `json:"body_clean"`
	CompletedCount int              `json:"completed_count"`
	TotalCriteria  int              `json:"total_criteria"`
	DependsOnRefs  []apiTaskRef     `json:"depends_on_refs"`
	LinkedTaskRefs []apiTaskRef     `json:"linked_task_refs"`
}

// loadTaskRefs returns one apiTaskRef per ID in input order. Missing IDs
// fall back to {ID, "", ""} so the UI can still render the broken link
// instead of dropping it silently. The query is a single batch — never a
// loop — to keep the round-trip count to one per detail-page load.
func loadTaskRefs(db *sql.DB, ids []string) ([]apiTaskRef, error) {
	out := make([]apiTaskRef, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		title   string
		summary string
	}
	rowsByID := make(map[string]row, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	q := "SELECT id, title, summary FROM tasks WHERE id IN (" + inPlaceholders(len(ids)) + ")" //nolint:gosec // inPlaceholders only emits "?,?,..." — no caller input
	rows, err := db.Query(q, args...)                                                          //nolint:gosec // q is the line above; bound args are safe
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, title, summary string
		if err := rows.Scan(&id, &title, &summary); err != nil {
			return nil, err
		}
		rowsByID[id] = row{title: title, summary: summary}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		r := rowsByID[id]
		out = append(out, apiTaskRef{ID: id, Title: r.title, Summary: r.summary})
	}
	return out, nil
}

// buildTaskDetailResponse loads the full task-detail payload for a task id
// from an open DB connection. Returns (response, http-status, err) — caller
// surfaces the status code on error so 404s come through as 404s. Used by
// both GET /api/tasks/{id} and PUT /api/tasks/{id}/depends_on so they emit
// the exact same shape.
func buildTaskDetailResponse(db *sql.DB, id string) (apiTaskDetailResponse, int, error) {
	task, err := LoadTaskDetail(db, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return apiTaskDetailResponse{}, http.StatusNotFound, err
		}
		return apiTaskDetailResponse{}, http.StatusInternalServerError, fmt.Errorf("load task: %w", err)
	}

	parent, err := loadParentSpecSummary(db, task.SpecID)
	if err != nil {
		return apiTaskDetailResponse{}, http.StatusInternalServerError, fmt.Errorf("load parent spec: %w", err)
	}
	dependsOn, err := loadTaskRefs(db, task.DependsOn)
	if err != nil {
		return apiTaskDetailResponse{}, http.StatusInternalServerError, fmt.Errorf("load depends_on: %w", err)
	}
	linked, err := loadTaskRefs(db, task.LinkedTasks)
	if err != nil {
		return apiTaskDetailResponse{}, http.StatusInternalServerError, fmt.Errorf("load linked_tasks: %w", err)
	}

	resp := apiTaskDetailResponse{
		Task:           task,
		ParentSpec:     parent,
		StatusLabel:    FromSlug(task.Status),
		BodyClean:      stripCriteriaSection(task.Body),
		TotalCriteria:  len(task.Criteria),
		DependsOnRefs:  dependsOn,
		LinkedTaskRefs: linked,
	}
	for _, c := range task.Criteria {
		if c.Checked == 1 {
			resp.CompletedCount++
		}
	}
	return resp, http.StatusOK, nil
}

// apiTaskDetailHandler implements GET /api/tasks/{id}.
func apiTaskDetailHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.ToUpper(r.PathValue("id"))
	if !strings.HasPrefix(id, IDPrefixTask) {
		writeJSONError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api task detail: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	resp, status, err := buildTaskDetailResponse(db, id)
	if err != nil {
		if status == http.StatusNotFound {
			writeJSONError(w, status, "task not found")
			return
		}
		slog.Error("api task detail", "error", err)
		writeJSONError(w, status, "failed to load task")
		return
	}
	writeJSON(w, status, resp)
}

// ---------------------------------------------------------------------------
// POST /api/tasks/move
// ---------------------------------------------------------------------------

// apiMoveTaskRequest is the JSON body of POST /api/tasks/move.
type apiMoveTaskRequest struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Position int    `json:"position"`
	Filter   string `json:"filter"`
}

// apiMoveTaskHandler implements POST /api/tasks/move. Validates, moves,
// rewrites the affected task .md files, and returns the updated board.
func apiMoveTaskHandler(w http.ResponseWriter, r *http.Request) {
	body, err := decodeJSON[apiMoveTaskRequest](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.ID == "" || body.Status == "" {
		writeJSONError(w, http.StatusBadRequest, "id and status are required")
		return
	}
	if body.Position < 0 {
		writeJSONError(w, http.StatusBadRequest, "position must be a non-negative integer")
		return
	}
	filter := normalizeBoardFilter(body.Filter)

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api move: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		writeJSONError(w, http.StatusBadRequest, "project not initialized")
		return
	}
	if !validStatus(proj.TaskStages, body.Status) {
		writeJSONError(w, http.StatusBadRequest, "unknown status: "+body.Status)
		return
	}

	moved, err := moveTask(db, body.ID, body.Status, body.Position)
	if err != nil {
		if errors.Is(err, errTaskNotFound) {
			writeJSONError(w, http.StatusNotFound, "task not found")
			return
		}
		slog.Error("api move: moveTask", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to move task")
		return
	}

	for _, id := range moved {
		if err := rewriteTaskFile(db, id); err != nil {
			slog.Error("api move: rewrite", "id", id, "error", err)
			w.Header().Set("X-Specd-Warning", "could not rewrite "+id+".md")
		}
	}

	data, err := loadBoard(db, proj.TaskStages, filter)
	if err != nil {
		slog.Error("api move: reload board", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reload board")
		return
	}

	slog.Info("api.tasks.move", "id", body.ID, "status", body.Status, "position", body.Position)
	writeJSON(w, http.StatusOK, boardJSONFromData(data, filter))
}

// ---------------------------------------------------------------------------
// POST /api/tasks/{id}/criteria/{position}/toggle
// ---------------------------------------------------------------------------

// apiToggleCriterionHandler implements
// POST /api/tasks/{id}/criteria/{position}/toggle. Returns the freshly-loaded
// GetTaskResponse as JSON.
func apiToggleCriterionHandler(w http.ResponseWriter, r *http.Request) {
	taskID, pos, ok := parseToggleParams(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid task id or criterion position")
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api toggle: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	if err := flipTaskCriterion(db, taskID, pos); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "criterion not found")
			return
		}
		slog.Error("api toggle: flip", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to toggle criterion")
		return
	}
	if err := rewriteTaskFile(db, taskID); err != nil {
		slog.Error("api toggle: rewrite", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to rewrite task file")
		return
	}

	task, err := LoadTaskDetail(db, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, "task not found")
			return
		}
		slog.Error("api toggle: reload", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reload task")
		return
	}

	slog.Info("api.tasks.criteria.toggle", "id", taskID, "position", pos) //nolint:gosec // taskID validated as TASK-N by parseToggleParams
	writeJSON(w, http.StatusOK, task)
}

// ---------------------------------------------------------------------------
// PUT /api/tasks/{id}/depends_on
// ---------------------------------------------------------------------------

// apiSetTaskDependsOnRequest is the body of PUT /api/tasks/{id}/depends_on.
// `DependsOn` is the COMPLETE replacement set of blocker task IDs. Send an
// empty array to clear all dependencies.
type apiSetTaskDependsOnRequest struct {
	DependsOn []string `json:"depends_on"`
}

// normalizeDependsOnInput uppercases, trims, and de-duplicates blocker task
// IDs from a request body. Returns an error if any ID is malformed or equals
// the subject taskID (self-dependency).
func normalizeDependsOnInput(taskID string, raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		id := strings.ToUpper(strings.TrimSpace(item))
		if id == "" {
			continue
		}
		if !strings.HasPrefix(id, IDPrefixTask) {
			return nil, fmt.Errorf("invalid task id in depends_on: %s", item)
		}
		if id == taskID {
			return nil, fmt.Errorf("task cannot depend on itself")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

// replaceTaskDependencies rewrites the task_dependencies rows for blocked_task
// = taskID inside a single transaction. The caller is responsible for ID
// validation; rows referencing tasks that don't exist will fail the insert
// foreign-key constraint and the whole transaction rolls back.
func replaceTaskDependencies(db *sql.DB, taskID string, deps []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.Exec("DELETE FROM task_dependencies WHERE blocked_task = ?", taskID); err != nil {
		return fmt.Errorf("clear depends_on: %w", err)
	}
	for _, blocker := range deps {
		if _, err := tx.Exec(
			"INSERT INTO task_dependencies(blocker_task, blocked_task) VALUES (?, ?)",
			blocker, taskID,
		); err != nil {
			return fmt.Errorf("insert depends_on %s: %w", blocker, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true
	return nil
}

// verifyTasksExist returns nil if every id has a row in tasks; otherwise the
// first missing id is surfaced so the caller can return a 400.
func verifyTasksExist(db *sql.DB, ids []string) (string, error) {
	for _, id := range ids {
		var exists int
		if err := db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", id).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return id, nil
			}
			return "", fmt.Errorf("verify task %s: %w", id, err)
		}
	}
	return "", nil
}

// apiSetTaskDependsOnHandler implements PUT /api/tasks/{id}/depends_on. The
// posted set replaces the rows in task_dependencies for the given task; the
// markdown file is rewritten and the full task detail is returned so the SPA
// can update its view without an extra fetch. Self-dependencies and unknown
// IDs are rejected; duplicate IDs in the body are de-duplicated.
func apiSetTaskDependsOnHandler(w http.ResponseWriter, r *http.Request) {
	taskID := strings.ToUpper(r.PathValue("id"))
	if !strings.HasPrefix(taskID, IDPrefixTask) {
		writeJSONError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	body, err := decodeJSON[apiSetTaskDependsOnRequest](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	deps, err := normalizeDependsOnInput(taskID, body.DependsOn)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api depends_on: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	missing, err := verifyTasksExist(db, append([]string{taskID}, deps...))
	if err != nil {
		slog.Error("api depends_on: verify", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to verify task ids")
		return
	}
	if missing == taskID {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}
	if missing != "" {
		writeJSONError(w, http.StatusBadRequest, "unknown task in depends_on: "+missing)
		return
	}

	if err := replaceTaskDependencies(db, taskID, deps); err != nil {
		slog.Error("api depends_on: replace", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save depends_on")
		return
	}

	if err := rewriteTaskFile(db, taskID); err != nil {
		slog.Error("api depends_on: rewrite", "error", err)
		// File rewrite failure isn't fatal — DB is already updated. Surface
		// a soft warning header so the SPA can show a toast if it wants.
		w.Header().Set("X-Specd-Warning", "depends_on saved but markdown rewrite failed")
	}

	resp, status, err := buildTaskDetailResponse(db, taskID)
	if err != nil {
		slog.Error("api depends_on: reload", "error", err)
		writeJSONError(w, status, "failed to reload task")
		return
	}
	slog.Info("api.tasks.depends_on.set", "id", taskID, "count", len(deps)) //nolint:gosec // taskID validated above
	writeJSON(w, status, resp)
}

// ---------------------------------------------------------------------------
// DELETE /api/tasks/{id}
// ---------------------------------------------------------------------------

// apiDeleteTaskHandler implements DELETE /api/tasks/{id}. Removes the task
// from the database (FK cascades clean up criteria, links, dependencies) and
// deletes its markdown file from disk. Returns the same payload as the CLI
// `specd delete-task` so the SPA can confirm what got removed.
func apiDeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := strings.ToUpper(r.PathValue("id"))
	if !strings.HasPrefix(taskID, IDPrefixTask) {
		writeJSONError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api delete task: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	resp, err := DeleteTask(db, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			writeJSONError(w, http.StatusNotFound, "task not found")
			return
		}
		slog.Error("api delete task: delete", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	slog.Info("api.tasks.delete", "id", resp.ID, "spec_id", resp.SpecID) //nolint:gosec // taskID validated above
	writeJSON(w, http.StatusOK, resp)
}
