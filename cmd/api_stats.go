// api_stats.go implements GET /api/stats. Cheap project-wide counts for
// the welcome dashboard tiles. Tasks are reported as both total and done
// so the client can render an X / Y label plus a progress bar without a
// second round-trip.
package cmd

import (
	"log/slog"
	"net/http"
)

// apiStatsResponse is the payload returned by GET /api/stats.
type apiStatsResponse struct {
	TasksTotal int `json:"tasks_total"`
	TasksDone  int `json:"tasks_done"`
	Specs      int `json:"specs"`
	KBDocs     int `json:"kb_docs"`
}

// apiStatsHandler implements GET /api/stats. Counts come from the project
// SQLite database. Each query is independent, so a failure on one table
// (e.g. kb_docs missing on a fresh init) returns 500 rather than partial
// stats — the dashboard can fall back to its skeleton state until the
// project is fully initialized.
func apiStatsHandler(w http.ResponseWriter, _ *http.Request) {
	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api stats: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	var resp apiStatsResponse
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&resp.TasksTotal); err != nil {
		slog.Error("api stats: tasks_total", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to count tasks")
		return
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status = 'done'`).Scan(&resp.TasksDone); err != nil {
		slog.Error("api stats: tasks_done", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to count completed tasks")
		return
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM specs`).Scan(&resp.Specs); err != nil {
		slog.Error("api stats: specs", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to count specs")
		return
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM kb_docs`).Scan(&resp.KBDocs); err != nil {
		slog.Error("api stats: kb_docs", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to count kb docs")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
