// Package web — handlers.go defines HTTP handler functions for the
// embedded web UI pages. Each handler queries the workspace and renders
// a template with BeerCSS components.
//
// Dialog form handlers use htmx: on validation error they return the
// form partial with HTTP 422 (htmx swaps it in place, preserving user
// input). On success they return HX-Redirect.
//
// Non-dialog form handlers (delete, move, criteria) use standard
// redirects via redirectWithError on failure.
package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stackific/specd/internal/workspace"
)

// validSpecTypes is the set of accepted spec type values.
var validSpecTypes = map[string]bool{
	"business":       true,
	"functional":     true,
	"non-functional": true,
}

// validTaskStatuses is the set of accepted task status values.
var validTaskStatuses = map[string]bool{
	"backlog":              true,
	"todo":                 true,
	"in_progress":          true,
	"blocked":              true,
	"pending_verification": true,
	"done":                 true,
	"cancelled":            true,
	"wontfix":              true,
}

// trimFormValue returns the trimmed form value, collapsing consecutive spaces.
func trimFormValue(r *http.Request, key string) string {
	v := strings.TrimSpace(r.FormValue(key))
	for strings.Contains(v, "  ") {
		v = strings.ReplaceAll(v, "  ", " ")
	}
	return v
}

// =====================================================================
// Form data types — used for htmx form partial re-rendering on error
// =====================================================================

// SpecFormData holds submitted values + error for the new-spec form partial.
type SpecFormData struct {
	Title   string
	Type    string
	Summary string
	Body    string
	Error   string
}

// EditSpecFormData holds submitted values + error for the edit-spec form partial.
type EditSpecFormData struct {
	ID      string
	Title   string
	Type    string
	Summary string
	Body    string
	Error   string
}

// TaskFormData holds submitted values + error for the new-task form partial
// (used on spec-detail page where spec_id is fixed).
type TaskFormData struct {
	SpecID  string
	Title   string
	Summary string
	Status  string
	Body    string
	Error   string
}

// BoardTaskFormData holds submitted values + error + specs list for the
// board new-task form partial (where spec_id is a dropdown).
type BoardTaskFormData struct {
	Specs   []workspace.Spec
	SpecID  string
	Title   string
	Summary string
	Status  string
	Body    string
	Error   string
}

// EditTaskFormData holds submitted values + error for the edit-task form partial.
type EditTaskFormData struct {
	ID      string
	Title   string
	Summary string
	Body    string
	Error   string
}

// KBFormData holds submitted values + error for the add-kb form partial.
type KBFormData struct {
	URL   string
	Title string
	Note  string
	Error string
}

// =====================================================================
// Board (kanban)
// =====================================================================

// BoardData holds tasks grouped by status for the kanban view.
type BoardData struct {
	Columns       []BoardColumn
	Specs         []workspace.Spec // for the spec filter dropdown
	FilterSpec    string           // currently selected spec ID, empty = all
	BoardTaskForm *BoardTaskFormData
}

// BoardColumn represents one kanban status column.
type BoardColumn struct {
	Status string
	Label  string
	Icon   string
	Tasks  []BoardTask
}

// BoardTask enriches a task with presentation metadata for kanban cards.
type BoardTask struct {
	workspace.Task
	SpecTitle       string
	CriteriaTotal   int
	CriteriaChecked int
	HasDeps         bool // task has at least one dependency
	IsBlocked       bool // at least one dep is not ready
	HasCitations    bool
}

// kanbanStatuses defines the column order and display labels.
var kanbanStatuses = []struct {
	Status string
	Label  string
	Icon   string
}{
	{"backlog", "Backlog", "inventory_2"},
	{"todo", "To Do", "checklist"},
	{"in_progress", "In Progress", "pending"},
	{"blocked", "Blocked", "block"},
	{"pending_verification", "Verification", "verified"},
	{"done", "Done", "check_circle"},
	{"cancelled", "Cancelled", "cancel"},
	{"wontfix", "Won't Fix", "do_not_disturb_on"},
}

func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	filterSpec := r.URL.Query().Get("spec")

	allSpecs, _ := s.w.ListSpecs(workspace.ListSpecsFilter{})
	if allSpecs == nil {
		allSpecs = []workspace.Spec{}
	}
	specTitles := make(map[string]string, len(allSpecs))
	for _, sp := range allSpecs {
		specTitles[sp.ID] = sp.Title
	}

	data := BoardData{
		Specs:         allSpecs,
		FilterSpec:    filterSpec,
		BoardTaskForm: &BoardTaskFormData{Specs: allSpecs, Status: "backlog"},
	}

	for _, ks := range kanbanStatuses {
		filter := workspace.ListTasksFilter{Status: ks.Status}
		if filterSpec != "" {
			filter.SpecID = filterSpec
		}
		tasks, _ := s.w.ListTasks(filter)
		if tasks == nil {
			tasks = []workspace.Task{}
		}

		boardTasks := make([]BoardTask, 0, len(tasks))
		for _, t := range tasks {
			bt := BoardTask{
				Task:      t,
				SpecTitle: specTitles[t.SpecID],
			}

			if criteria, err := s.w.ListCriteria(t.ID); err == nil {
				bt.CriteriaTotal = len(criteria)
				for _, c := range criteria {
					if c.Checked {
						bt.CriteriaChecked++
					}
				}
			}

			if deps, err := s.w.GetTaskDeps(t.ID); err == nil && len(deps) > 0 {
				bt.HasDeps = true
				for _, d := range deps {
					if !d.Ready {
						bt.IsBlocked = true
						break
					}
				}
			}

			if cites, err := s.w.GetCitations(t.ID); err == nil && len(cites) > 0 {
				bt.HasCitations = true
			}

			boardTasks = append(boardTasks, bt)
		}

		data.Columns = append(data.Columns, BoardColumn{
			Status: ks.Status,
			Label:  ks.Label,
			Icon:   ks.Icon,
			Tasks:  boardTasks,
		})
	}

	s.renderPage(w, r, "board", PageData{
		Title:  "Board",
		Active: "board",
		Data:   data,
	})
}

// handleMoveTask handles POST /tasks/{id}/move to change a task's status.
func (s *Server) handleMoveTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	newStatus := trimFormValue(r, "status")
	taskPath := fmt.Sprintf("/tasks/%s", taskID)

	if newStatus == "" {
		redirectWithError(w, r, taskPath, "Status is required")
		return
	}
	if !validTaskStatuses[newStatus] {
		redirectWithError(w, r, taskPath, "Invalid status value")
		return
	}

	if err := s.w.MoveTask(taskID, newStatus); err != nil {
		redirectWithError(w, r, taskPath, err.Error())
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", taskPath)
		return
	}
	http.Redirect(w, r, taskPath, http.StatusSeeOther)
}

// handleReorderTask handles POST /tasks/{id}/reorder to change position.
func (s *Server) handleReorderTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	posStr := r.FormValue("position")

	taskPath := fmt.Sprintf("/tasks/%s", taskID)
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		redirectWithError(w, r, taskPath, "Invalid position")
		return
	}

	if err := s.w.ReorderTask(taskID, workspace.ReorderInput{
		Mode:     workspace.ReorderTo,
		Position: pos,
	}); err != nil {
		redirectWithError(w, r, taskPath, err.Error())
		return
	}

	s.renderBoardPartial(w, r)
}

// renderBoardPartial renders the board content block as an htmx partial,
// without mutating the incoming request headers.
func (s *Server) renderBoardPartial(w http.ResponseWriter, r *http.Request) {
	filterSpec := r.URL.Query().Get("spec")

	allSpecs, _ := s.w.ListSpecs(workspace.ListSpecsFilter{})
	if allSpecs == nil {
		allSpecs = []workspace.Spec{}
	}
	specTitles := make(map[string]string, len(allSpecs))
	for _, sp := range allSpecs {
		specTitles[sp.ID] = sp.Title
	}

	data := BoardData{
		Specs:         allSpecs,
		FilterSpec:    filterSpec,
		BoardTaskForm: &BoardTaskFormData{Specs: allSpecs, Status: "backlog"},
	}

	for _, ks := range kanbanStatuses {
		filter := workspace.ListTasksFilter{Status: ks.Status}
		if filterSpec != "" {
			filter.SpecID = filterSpec
		}
		tasks, _ := s.w.ListTasks(filter)
		if tasks == nil {
			tasks = []workspace.Task{}
		}

		boardTasks := make([]BoardTask, 0, len(tasks))
		for _, t := range tasks {
			bt := BoardTask{Task: t, SpecTitle: specTitles[t.SpecID]}
			if criteria, err := s.w.ListCriteria(t.ID); err == nil {
				bt.CriteriaTotal = len(criteria)
				for _, c := range criteria {
					if c.Checked {
						bt.CriteriaChecked++
					}
				}
			}
			if deps, err := s.w.GetTaskDeps(t.ID); err == nil && len(deps) > 0 {
				bt.HasDeps = true
				for _, d := range deps {
					if !d.Ready {
						bt.IsBlocked = true
						break
					}
				}
			}
			if cites, err := s.w.GetCitations(t.ID); err == nil && len(cites) > 0 {
				bt.HasCitations = true
			}
			boardTasks = append(boardTasks, bt)
		}

		data.Columns = append(data.Columns, BoardColumn{
			Status: ks.Status, Label: ks.Label, Icon: ks.Icon, Tasks: boardTasks,
		})
	}

	pd := PageData{Title: "Board", Active: "board", CSSFile: s.cssFile, Data: data}
	tmpl := s.pages["board"]
	tmpl.ExecuteTemplate(w, "content", pd)
}

// MoveRequest represents the JSON body for a task move via drag-and-drop.
type MoveRequest struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// ReorderRequest represents the JSON body for a task reorder via drag-and-drop.
type ReorderRequest struct {
	TaskID   string `json:"task_id"`
	Position int    `json:"position"`
}

// handleDragMove handles POST /api/board/move for drag-and-drop status changes.
func (s *Server) handleDragMove(w http.ResponseWriter, r *http.Request) {
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.TaskID == "" || req.Status == "" {
		http.Error(w, "missing task_id or status", http.StatusBadRequest)
		return
	}
	if !validTaskStatuses[req.Status] {
		http.Error(w, "invalid status value", http.StatusBadRequest)
		return
	}

	if err := s.w.MoveTask(req.TaskID, req.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderBoardPartial(w, r)
}

// handleDragReorder handles POST /api/board/reorder for drag-and-drop reordering.
func (s *Server) handleDragReorder(w http.ResponseWriter, r *http.Request) {
	var req ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.TaskID == "" {
		http.Error(w, "missing task_id", http.StatusBadRequest)
		return
	}

	if err := s.w.ReorderTask(req.TaskID, workspace.ReorderInput{
		Mode:     workspace.ReorderTo,
		Position: req.Position,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderBoardPartial(w, r)
}

// =====================================================================
// Specs list
// =====================================================================

// SpecsData holds the spec list grouped by type with progress info.
type SpecsData struct {
	Groups   []SpecGroup
	SpecForm *SpecFormData
}

// SpecGroup holds specs of a single type for grouped rendering.
type SpecGroup struct {
	Type  string
	Label string
	Specs []SpecWithProgress
}

// SpecWithProgress pairs a spec with its task progress.
type SpecWithProgress struct {
	Spec     workspace.Spec
	Progress *workspace.SpecProgress
}

// specTypeOrder defines display order and labels for spec type groups.
var specTypeOrder = []struct{ Type, Label string }{
	{"functional", "Functional"},
	{"business", "Business"},
	{"non-functional", "Non-functional"},
}

func (s *Server) handleSpecs(w http.ResponseWriter, r *http.Request) {
	specs, _ := s.w.ListSpecs(workspace.ListSpecsFilter{})
	if specs == nil {
		specs = []workspace.Spec{}
	}

	// Build per-type items map.
	byType := make(map[string][]SpecWithProgress)
	for _, spec := range specs {
		progress, _ := s.w.GetSpecProgress(spec.ID)
		byType[spec.Type] = append(byType[spec.Type], SpecWithProgress{Spec: spec, Progress: progress})
	}

	// Assemble groups in display order, skipping empty types.
	var groups []SpecGroup
	for _, st := range specTypeOrder {
		if items, ok := byType[st.Type]; ok && len(items) > 0 {
			groups = append(groups, SpecGroup{Type: st.Type, Label: st.Label, Specs: items})
		}
	}

	s.renderPage(w, r, "specs", PageData{
		Title:  "Specs",
		Active: "specs",
		Data: SpecsData{
			Groups:   groups,
			SpecForm: &SpecFormData{Type: "functional"},
		},
	})
}

// handleCreateSpec handles POST /specs to create a new spec.
// On error, returns the form partial with 422 for htmx to swap in place.
func (s *Server) handleCreateSpec(w http.ResponseWriter, r *http.Request) {
	form := SpecFormData{
		Title:   trimFormValue(r, "title"),
		Type:    trimFormValue(r, "type"),
		Summary: trimFormValue(r, "summary"),
		Body:    strings.TrimSpace(r.FormValue("body")),
	}

	// Validate.
	if form.Title == "" {
		form.Error = "Title is required"
	} else if len(form.Title) < 2 {
		form.Error = "Title must be at least 2 characters"
	} else if !validSpecTypes[form.Type] {
		form.Error = "Invalid spec type"
	} else if form.Body == "" {
		form.Error = "Body is required — describe the spec in enough detail for an AI agent to act on it"
	} else if len(form.Body) < 20 {
		form.Error = "Body must be at least 20 characters"
	}

	if form.Error != "" {
		s.renderFormPartial(w, "specs", "new-spec-form", &form)
		return
	}

	result, err := s.w.NewSpec(workspace.NewSpecInput{
		Title:   form.Title,
		Type:    form.Type,
		Summary: form.Summary,
		Body:    form.Body,
	})
	if err != nil {
		form.Error = err.Error()
		s.renderFormPartial(w, "specs", "new-spec-form", &form)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/specs/%s", result.ID))
}

// =====================================================================
// Spec detail
// =====================================================================

// SpecDetailData holds a single spec with its tasks, links, citations, and form state.
type SpecDetailData struct {
	Spec         *workspace.Spec
	Tasks        []SpecDetailTask
	Links        []workspace.LinkedSpec
	Citations    []workspace.CitationDetail
	Progress     *workspace.SpecProgress
	AllSpecs     []workspace.Spec
	EditSpecForm *EditSpecFormData
	TaskForm     *TaskFormData
}

// SpecDetailTask enriches a task with criteria progress for the spec detail view.
type SpecDetailTask struct {
	workspace.Task
	CriteriaTotal   int
	CriteriaChecked int
}

func (s *Server) handleSpecDetail(w http.ResponseWriter, r *http.Request) {
	specID := r.PathValue("id")
	spec, err := s.w.ReadSpec(specID)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "Spec not found")
		return
	}

	tasks, _ := s.w.ListTasks(workspace.ListTasksFilter{SpecID: specID})
	links, _ := s.w.GetSpecLinks(specID)
	citations, _ := s.w.GetCitations(specID)
	progress, _ := s.w.GetSpecProgress(specID)
	allSpecs, _ := s.w.ListSpecs(workspace.ListSpecsFilter{})

	// Enrich tasks with criteria progress.
	detailTasks := make([]SpecDetailTask, 0, len(tasks))
	for _, t := range tasks {
		dt := SpecDetailTask{Task: t}
		if criteria, err := s.w.ListCriteria(t.ID); err == nil {
			dt.CriteriaTotal = len(criteria)
			for _, c := range criteria {
				if c.Checked {
					dt.CriteriaChecked++
				}
			}
		}
		detailTasks = append(detailTasks, dt)
	}

	s.renderPage(w, r, "spec-detail", PageData{
		Title:  spec.Title,
		Active: "specs",
		Data: SpecDetailData{
			Spec:      spec,
			Tasks:     detailTasks,
			Links:     links,
			Citations: citations,
			Progress:  progress,
			AllSpecs:  allSpecs,
			EditSpecForm: &EditSpecFormData{
				ID: spec.ID, Title: spec.Title, Type: spec.Type,
				Summary: spec.Summary, Body: spec.Body,
			},
			TaskForm: &TaskFormData{SpecID: specID, Status: "backlog"},
		},
	})
}

// handleUpdateSpec handles POST /specs/{id}/update to edit spec fields.
func (s *Server) handleUpdateSpec(w http.ResponseWriter, r *http.Request) {
	specID := r.PathValue("id")
	form := EditSpecFormData{
		ID:      specID,
		Title:   trimFormValue(r, "title"),
		Type:    trimFormValue(r, "type"),
		Summary: trimFormValue(r, "summary"),
		Body:    strings.TrimSpace(r.FormValue("body")),
	}

	// Validate.
	if form.Title == "" {
		form.Error = "Title is required"
	} else if len(form.Title) < 2 {
		form.Error = "Title must be at least 2 characters"
	} else if !validSpecTypes[form.Type] {
		form.Error = "Invalid spec type"
	} else if form.Body == "" {
		form.Error = "Body is required"
	} else if len(form.Body) < 20 {
		form.Error = "Body must be at least 20 characters"
	}

	if form.Error != "" {
		s.renderFormPartial(w, "spec-detail", "edit-spec-form", &form)
		return
	}

	input := workspace.UpdateSpecInput{
		Title: &form.Title, Type: &form.Type,
		Summary: &form.Summary, Body: &form.Body,
	}
	if err := s.w.UpdateSpec(specID, input); err != nil {
		form.Error = err.Error()
		s.renderFormPartial(w, "spec-detail", "edit-spec-form", &form)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/specs/%s", specID))
}

// handleDeleteSpec handles POST /specs/{id}/delete to soft-delete a spec.
func (s *Server) handleDeleteSpec(w http.ResponseWriter, r *http.Request) {
	specID := r.PathValue("id")

	if err := s.w.DeleteSpec(specID); err != nil {
		redirectWithError(w, r, fmt.Sprintf("/specs/%s", specID), err.Error())
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/specs")
		return
	}
	http.Redirect(w, r, "/specs", http.StatusSeeOther)
}

// =====================================================================
// Task detail
// =====================================================================

// TaskDetailData holds a single task with criteria, links, deps, and citations.
type TaskDetailData struct {
	Task         *workspace.Task
	Criteria     []workspace.CriterionRow
	Links        []workspace.LinkedTask
	Deps         []workspace.TaskDependency
	Citations    []workspace.CitationDetail
	ParentSpec   *workspace.Spec
	Statuses     []StatusOption
	EditTaskForm *EditTaskFormData
}

// StatusOption is a status value + label for dropdown rendering.
type StatusOption struct {
	Value    string
	Label    string
	Selected bool
}

// allStatuses returns the 8 task statuses with labels.
func allStatuses(current string) []StatusOption {
	defs := []struct{ V, L string }{
		{"backlog", "Backlog"},
		{"todo", "To Do"},
		{"in_progress", "In Progress"},
		{"blocked", "Blocked"},
		{"pending_verification", "Verification"},
		{"done", "Done"},
		{"cancelled", "Cancelled"},
		{"wontfix", "Won't Fix"},
	}
	opts := make([]StatusOption, len(defs))
	for i, d := range defs {
		opts[i] = StatusOption{Value: d.V, Label: d.L, Selected: d.V == current}
	}
	return opts
}

func (s *Server) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	task, err := s.w.ReadTask(taskID)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "Task not found")
		return
	}

	criteria, _ := s.w.ListCriteria(taskID)
	links, _ := s.w.GetTaskLinks(taskID)
	deps, _ := s.w.GetTaskDeps(taskID)
	citations, _ := s.w.GetCitations(taskID)
	parentSpec, _ := s.w.ReadSpec(task.SpecID)

	s.renderPage(w, r, "task-detail", PageData{
		Title:  task.Title,
		Active: "board",
		Data: TaskDetailData{
			Task:       task,
			Criteria:   criteria,
			Links:      links,
			Deps:       deps,
			Citations:  citations,
			ParentSpec: parentSpec,
			Statuses:   allStatuses(task.Status),
			EditTaskForm: &EditTaskFormData{
				ID: task.ID, Title: task.Title,
				Summary: task.Summary, Body: task.Body,
			},
		},
	})
}

// handleCreateTask handles POST /tasks to create a new task.
// Renders the appropriate form partial on error (board or spec-detail variant).
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	specID := trimFormValue(r, "spec_id")
	title := trimFormValue(r, "title")
	summary := trimFormValue(r, "summary")
	body := strings.TrimSpace(r.FormValue("body"))
	status := trimFormValue(r, "status")
	if status == "" {
		status = "backlog"
	}

	// Validate.
	var errMsg string
	if specID == "" {
		errMsg = "Parent spec is required"
	} else if title == "" {
		errMsg = "Title is required"
	} else if len(title) < 2 {
		errMsg = "Title must be at least 2 characters"
	} else if !validTaskStatuses[status] {
		errMsg = "Invalid status value"
	} else if body == "" {
		errMsg = "Body is required — describe the task in enough detail for an AI agent to act on it"
	} else if len(body) < 20 {
		errMsg = "Body must be at least 20 characters"
	}

	if errMsg != "" {
		// Determine which form partial to re-render.
		// If the referer is the board page, render the board variant.
		referer := r.Referer()
		if referer == "" || strings.HasSuffix(referer, "/") || strings.Contains(referer, "/?") {
			allSpecs, _ := s.w.ListSpecs(workspace.ListSpecsFilter{})
			form := &BoardTaskFormData{
				Specs: allSpecs, SpecID: specID,
				Title: title, Summary: summary, Status: status, Body: body,
				Error: errMsg,
			}
			s.renderFormPartial(w, "board", "board-new-task-form", form)
		} else {
			form := &TaskFormData{
				SpecID: specID, Title: title, Summary: summary,
				Status: status, Body: body, Error: errMsg,
			}
			s.renderFormPartial(w, "spec-detail", "new-task-form", form)
		}
		return
	}

	result, err := s.w.NewTask(workspace.NewTaskInput{
		SpecID: specID, Title: title, Summary: summary,
		Body: body, Status: status,
	})
	if err != nil {
		// Server error — same partial logic.
		referer := r.Referer()
		if referer == "" || strings.HasSuffix(referer, "/") || strings.Contains(referer, "/?") {
			allSpecs, _ := s.w.ListSpecs(workspace.ListSpecsFilter{})
			form := &BoardTaskFormData{
				Specs: allSpecs, SpecID: specID,
				Title: title, Summary: summary, Status: status, Body: body,
				Error: err.Error(),
			}
			s.renderFormPartial(w, "board", "board-new-task-form", form)
		} else {
			form := &TaskFormData{
				SpecID: specID, Title: title, Summary: summary,
				Status: status, Body: body, Error: err.Error(),
			}
			s.renderFormPartial(w, "spec-detail", "new-task-form", form)
		}
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/tasks/%s", result.ID))
}

// handleUpdateTask handles POST /tasks/{id}/update to edit task fields.
func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	form := EditTaskFormData{
		ID:      taskID,
		Title:   trimFormValue(r, "title"),
		Summary: trimFormValue(r, "summary"),
		Body:    strings.TrimSpace(r.FormValue("body")),
	}

	// Validate.
	if form.Title == "" {
		form.Error = "Title is required"
	} else if len(form.Title) < 2 {
		form.Error = "Title must be at least 2 characters"
	} else if form.Body == "" {
		form.Error = "Body is required"
	} else if len(form.Body) < 20 {
		form.Error = "Body must be at least 20 characters"
	}

	if form.Error != "" {
		s.renderFormPartial(w, "task-detail", "edit-task-form", &form)
		return
	}

	input := workspace.UpdateTaskInput{
		Title: &form.Title, Summary: &form.Summary, Body: &form.Body,
	}
	if err := s.w.UpdateTask(taskID, input); err != nil {
		form.Error = err.Error()
		s.renderFormPartial(w, "task-detail", "edit-task-form", &form)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/tasks/%s", taskID))
}

// handleDeleteTask handles POST /tasks/{id}/delete to soft-delete a task.
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	task, err := s.w.ReadTask(taskID)
	if err != nil {
		redirectWithError(w, r, "/", err.Error())
		return
	}

	if err := s.w.DeleteTask(taskID); err != nil {
		redirectWithError(w, r, fmt.Sprintf("/tasks/%s", taskID), err.Error())
		return
	}

	redirect := fmt.Sprintf("/specs/%s", task.SpecID)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirect)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// =====================================================================
// Criteria — non-dialog forms, use redirectWithError
// =====================================================================

// handleCheckCriterion handles POST /tasks/{id}/criteria/{pos}/check.
func (s *Server) handleCheckCriterion(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	taskPath := fmt.Sprintf("/tasks/%s", taskID)
	pos, err := strconv.Atoi(r.PathValue("pos"))
	if err != nil {
		redirectWithError(w, r, taskPath, "Invalid criterion position")
		return
	}

	if err := s.w.CheckCriterion(taskID, pos); err != nil {
		redirectWithError(w, r, taskPath, err.Error())
		return
	}

	s.reloadTaskDetail(w, r, taskID)
}

// handleUncheckCriterion handles POST /tasks/{id}/criteria/{pos}/uncheck.
func (s *Server) handleUncheckCriterion(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	taskPath := fmt.Sprintf("/tasks/%s", taskID)
	pos, err := strconv.Atoi(r.PathValue("pos"))
	if err != nil {
		redirectWithError(w, r, taskPath, "Invalid criterion position")
		return
	}

	if err := s.w.UncheckCriterion(taskID, pos); err != nil {
		redirectWithError(w, r, taskPath, err.Error())
		return
	}

	s.reloadTaskDetail(w, r, taskID)
}

// handleAddCriterion handles POST /tasks/{id}/criteria to add a new criterion.
func (s *Server) handleAddCriterion(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	taskPath := fmt.Sprintf("/tasks/%s", taskID)
	text := trimFormValue(r, "text")

	if text == "" {
		redirectWithError(w, r, taskPath, "Criterion text is required")
		return
	}
	if len(text) < 2 {
		redirectWithError(w, r, taskPath, "Criterion text must be at least 2 characters")
		return
	}

	if _, err := s.w.AddCriterion(taskID, text); err != nil {
		redirectWithError(w, r, taskPath, err.Error())
		return
	}

	s.reloadTaskDetail(w, r, taskID)
}

// handleRemoveCriterion handles POST /tasks/{id}/criteria/{pos}/remove.
func (s *Server) handleRemoveCriterion(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	taskPath := fmt.Sprintf("/tasks/%s", taskID)
	pos, err := strconv.Atoi(r.PathValue("pos"))
	if err != nil {
		redirectWithError(w, r, taskPath, "Invalid criterion position")
		return
	}

	if err := s.w.RemoveCriterion(taskID, pos); err != nil {
		redirectWithError(w, r, taskPath, err.Error())
		return
	}

	s.reloadTaskDetail(w, r, taskID)
}

// reloadTaskDetail re-renders the task detail page as an htmx partial.
func (s *Server) reloadTaskDetail(w http.ResponseWriter, r *http.Request, taskID string) {
	task, err := s.w.ReadTask(taskID)
	if err != nil {
		redirectWithError(w, r, "/", err.Error())
		return
	}

	criteria, _ := s.w.ListCriteria(taskID)
	links, _ := s.w.GetTaskLinks(taskID)
	deps, _ := s.w.GetTaskDeps(taskID)
	citations, _ := s.w.GetCitations(taskID)
	parentSpec, _ := s.w.ReadSpec(task.SpecID)

	r.Header.Set("HX-Request", "true")
	s.renderPage(w, r, "task-detail", PageData{
		Title:  task.Title,
		Active: "board",
		Data: TaskDetailData{
			Task:       task,
			Criteria:   criteria,
			Links:      links,
			Deps:       deps,
			Citations:  citations,
			ParentSpec: parentSpec,
			Statuses:   allStatuses(task.Status),
			EditTaskForm: &EditTaskFormData{
				ID: task.ID, Title: task.Title,
				Summary: task.Summary, Body: task.Body,
			},
		},
	})
}

// =====================================================================
// KB browser
// =====================================================================

// KBData holds KB document list, search results, and add form state.
type KBData struct {
	Docs          []workspace.KBDoc
	KBForm        *KBFormData
	Query         string
	FilterType    string
	SearchResults []workspace.KBSearchResult
}

func (s *Server) handleKB(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	filterType := r.URL.Query().Get("type")

	filter := workspace.KBListFilter{}
	if filterType != "" {
		filter.SourceType = filterType
	}

	docs, _ := s.w.KBList(filter)
	if docs == nil {
		docs = []workspace.KBDoc{}
	}

	var searchResults []workspace.KBSearchResult
	if query != "" {
		searchResults, _ = s.w.KBSearch(query, 20)
		if searchResults == nil {
			searchResults = []workspace.KBSearchResult{}
		}
	}

	s.renderPage(w, r, "kb", PageData{
		Title:  "Knowledge Base",
		Active: "kb",
		Data: KBData{
			Docs:          docs,
			KBForm:        &KBFormData{},
			Query:         query,
			FilterType:    filterType,
			SearchResults: searchResults,
		},
	})
}

// handleAddKB handles POST /kb to add a new KB document via file upload or URL.
func (s *Server) handleAddKB(w http.ResponseWriter, r *http.Request) {
	form := KBFormData{
		Title: trimFormValue(r, "title"),
		Note:  trimFormValue(r, "note"),
		URL:   trimFormValue(r, "url"),
	}

	var source string

	if form.URL != "" {
		source = form.URL
	} else {
		file, header, err := r.FormFile("file")
		if err != nil {
			form.Error = "Please select a file or enter a URL"
			s.renderFormPartial(w, "kb", "add-kb-form", &form)
			return
		}
		defer file.Close()

		tmpDir := os.TempDir()
		tmpPath := filepath.Join(tmpDir, header.Filename)
		out, err := os.Create(tmpPath)
		if err != nil {
			form.Error = "Failed to process upload"
			s.renderFormPartial(w, "kb", "add-kb-form", &form)
			return
		}
		if _, err := io.Copy(out, file); err != nil {
			out.Close()
			os.Remove(tmpPath)
			form.Error = "Failed to process upload"
			s.renderFormPartial(w, "kb", "add-kb-form", &form)
			return
		}
		out.Close()
		defer os.Remove(tmpPath)
		source = tmpPath
	}

	if _, err := s.w.KBAdd(workspace.KBAddInput{
		Source: source,
		Title:  form.Title,
		Note:   form.Note,
	}); err != nil {
		form.Error = err.Error()
		s.renderFormPartial(w, "kb", "add-kb-form", &form)
		return
	}

	w.Header().Set("HX-Redirect", "/kb")
}

// handleDeleteKB handles POST /kb/{id}/delete to remove a KB doc.
func (s *Server) handleDeleteKB(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("id")

	if err := s.w.KBRemove(kbID); err != nil {
		redirectWithError(w, r, "/kb", err.Error())
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/kb")
		return
	}
	http.Redirect(w, r, "/kb", http.StatusSeeOther)
}

// =====================================================================
// KB detail (reader)
// =====================================================================

// KBDetailData holds a single KB document with its chunks for the detail view.
type KBDetailData struct {
	Doc         workspace.KBDoc
	Chunks      []workspace.KBChunk
	TotalChunks int
	ActiveChunk int // 0-based chunk to highlight on load (-1 = none)
	Connections []workspace.ChunkConnectionResult
	Citations   []KBCitingItem
}

// KBCitingItem is a spec or task that cites this KB document.
type KBCitingItem struct {
	Kind  string // "spec" or "task"
	ID    string
	Title string
}

// handleKBDetail renders the KB detail page for a single document.
func (s *Server) handleKBDetail(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("id")
	result, err := s.w.KBRead(kbID, nil)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "KB document not found")
		return
	}

	activeChunk := -1
	if c := r.URL.Query().Get("chunk"); c != "" {
		if pos, err := strconv.Atoi(c); err == nil && pos >= 0 && pos < len(result.Chunks) {
			activeChunk = pos
		}
	}

	connections, _ := s.w.KBConnections(kbID, nil, 20)
	if connections == nil {
		connections = []workspace.ChunkConnectionResult{}
	}

	citations := s.findCitingItems(kbID)

	s.renderPage(w, r, "kb-detail", PageData{
		Title:  result.Doc.Title,
		Active: "kb",
		Data: KBDetailData{
			Doc:         result.Doc,
			Chunks:      result.Chunks,
			TotalChunks: len(result.Chunks),
			ActiveChunk: activeChunk,
			Connections: connections,
			Citations:   citations,
		},
	})
}

// findCitingItems returns specs and tasks that cite the given KB document.
func (s *Server) findCitingItems(kbDocID string) []KBCitingItem {
	rows, err := s.w.DB.Query(`
		SELECT DISTINCT c.from_kind, c.from_id,
			CASE c.from_kind
				WHEN 'spec' THEN (SELECT title FROM specs WHERE id = c.from_id)
				WHEN 'task' THEN (SELECT title FROM tasks WHERE id = c.from_id)
			END AS title
		FROM citations c
		WHERE c.kb_doc_id = ?
		ORDER BY c.from_kind, c.from_id`, kbDocID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []KBCitingItem
	for rows.Next() {
		var item KBCitingItem
		var title *string
		if err := rows.Scan(&item.Kind, &item.ID, &title); err != nil {
			continue
		}
		if title != nil {
			item.Title = *title
		}
		items = append(items, item)
	}
	return items
}

// =====================================================================
// KB API endpoints (JSON, for client-side reader)
// =====================================================================

// handleAPIKBDoc returns KB document metadata as JSON.
func (s *Server) handleAPIKBDoc(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("id")
	result, err := s.w.KBRead(kbID, nil)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Doc)
}

// handleAPIKBChunks returns all chunks for a KB document as JSON.
func (s *Server) handleAPIKBChunks(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("id")
	result, err := s.w.KBRead(kbID, nil)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Chunks)
}

// handleAPIKBChunk returns a single chunk by position as JSON.
func (s *Server) handleAPIKBChunk(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("id")
	posStr := r.PathValue("position")
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		http.Error(w, "invalid position", http.StatusBadRequest)
		return
	}

	result, err := s.w.KBRead(kbID, &pos)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if len(result.Chunks) == 0 {
		http.Error(w, "chunk not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Chunks[0])
}

// handleAPIKBRaw serves the raw (or cleaned) source bytes of a KB document.
// Path traversal protection: path is resolved via SQLite lookup, never from
// user input. Only files under specd/kb/ are served.
func (s *Server) handleAPIKBRaw(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("id")
	doc, err := s.w.KBRead(kbID, nil)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Determine which file to serve.
	var servePath string
	var contentType string

	switch doc.Doc.SourceType {
	case "html":
		// Serve the sanitized clean sidecar, never the original.
		if doc.Doc.CleanPath == nil {
			http.Error(w, "no cleaned HTML available", http.StatusNotFound)
			return
		}
		servePath = filepath.Join(s.w.Root, *doc.Doc.CleanPath)
		contentType = "text/html; charset=utf-8"
	case "pdf":
		servePath = filepath.Join(s.w.Root, doc.Doc.Path)
		contentType = "application/pdf"
	case "md":
		servePath = filepath.Join(s.w.Root, doc.Doc.Path)
		contentType = "text/plain; charset=utf-8"
	case "txt":
		servePath = filepath.Join(s.w.Root, doc.Doc.Path)
		contentType = "text/plain; charset=utf-8"
	default:
		http.Error(w, "unsupported source type", http.StatusBadRequest)
		return
	}

	// Security: verify the resolved path is under specd/kb/.
	absPath, err := filepath.Abs(servePath)
	if err != nil {
		http.Error(w, "path resolution error", http.StatusInternalServerError)
		return
	}
	kbDir := s.w.KBDir()
	if !strings.HasPrefix(absPath, kbDir+string(filepath.Separator)) && absPath != kbDir {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, absPath)
}

// =====================================================================
// Search
// =====================================================================

// SearchData holds search query and results.
type SearchData struct {
	Query   string
	Results *workspace.SearchResults
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var results *workspace.SearchResults
	if query != "" {
		results, _ = s.w.Search(query, "all", 20)
	}

	s.renderPage(w, r, "search", PageData{
		Title:  "Search",
		Active: "search",
		Data:   SearchData{Query: query, Results: results},
	})
}

// =====================================================================
// Status
// =====================================================================

// StatusData wraps the workspace status result with lint issues.
type StatusData struct {
	Status     *workspace.StatusResult
	LintIssues []workspace.LintIssue
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, _ := s.w.Status(true)

	var lintIssues []workspace.LintIssue
	if lint, err := s.w.Lint(); err == nil {
		lintIssues = lint.Issues
	}

	s.renderPage(w, r, "status", PageData{
		Title:  "Status",
		Active: "status",
		Data:   StatusData{Status: status, LintIssues: lintIssues},
	})
}

// =====================================================================
// Rejected files
// =====================================================================

// RejectedData holds rejected files list.
type RejectedData struct {
	Files []workspace.RejectedFile
}

func (s *Server) handleRejected(w http.ResponseWriter, r *http.Request) {
	files, _ := s.w.ListRejectedFiles()
	if files == nil {
		files = []workspace.RejectedFile{}
	}

	s.renderPage(w, r, "rejected", PageData{
		Title:  "Rejected Files",
		Active: "status",
		Data:   RejectedData{Files: files},
	})
}

// =====================================================================
// Trash
// =====================================================================

// TrashData holds trash items.
type TrashData struct {
	Items []workspace.TrashItem
}

func (s *Server) handleTrash(w http.ResponseWriter, r *http.Request) {
	items, _ := s.w.ListTrash(workspace.TrashListFilter{})
	if items == nil {
		items = []workspace.TrashItem{}
	}

	s.renderPage(w, r, "trash", PageData{
		Title:  "Trash",
		Active: "trash",
		Data:   TrashData{Items: items},
	})
}

// handleRestoreTrash handles POST /trash/{id}/restore.
func (s *Server) handleRestoreTrash(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		redirectWithError(w, r, "/trash", "Invalid trash ID")
		return
	}

	if _, err := s.w.RestoreTrash(id); err != nil {
		redirectWithError(w, r, "/trash", err.Error())
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/trash")
		return
	}
	http.Redirect(w, r, "/trash", http.StatusSeeOther)
}

// handlePurgeTrash handles POST /trash/purge to permanently remove items.
func (s *Server) handlePurgeTrash(w http.ResponseWriter, r *http.Request) {
	olderThan := r.FormValue("older_than")

	if olderThan != "" {
		if _, err := s.w.PurgeTrash(olderThan); err != nil {
			redirectWithError(w, r, "/trash", err.Error())
			return
		}
	} else {
		if _, err := s.w.PurgeAllTrash(); err != nil {
			redirectWithError(w, r, "/trash", err.Error())
			return
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/trash")
		return
	}
	http.Redirect(w, r, "/trash", http.StatusSeeOther)
}
