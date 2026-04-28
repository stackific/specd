// handlers_tasks_page.go implements the GET /tasks page handler. Reads the
// `filter` query parameter (all | incomplete) and threads it into PageData
// so the rendered shell wires the kanban initial-fetch and the active tab
// to the URL — mirroring how /specs treats `?view=` as the source of truth.
package cmd

import (
	"html/template"
	"net/http"
)

// TasksPageData is the view model passed to the tasks page template.
type TasksPageData struct {
	Filter string // "all" or "incomplete" — drives the active tab and the board fetch
}

// makeTasksHandler returns an http.HandlerFunc that renders /tasks.
func makeTasksHandler(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := normalizeBoardFilter(r.URL.Query().Get("filter"))
		renderPage(w, r, freshPages(), "tasks", &PageData{
			Title:   "Tasks",
			Active:  "tasks",
			DevMode: devMode,
			CSSHash: cssHash,
			JSHash:  jsHash,
			Data:    TasksPageData{Filter: filter},
		})
	}
}
