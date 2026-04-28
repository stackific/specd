// api.go is the thin facade for the JSON API that backs the TanStack-Router
// SPA in frontend/. It owns two things only:
//
//  1. RegisterAPI — wires every /api/* route to the handler in its own file.
//     One file per resource (api_meta.go, api_specs.go, api_tasks.go,
//     api_kb.go, api_search.go, api_stats.go, api_settings.go). Add a new
//     resource file when introducing a new top-level path.
//  2. The shared JSON helpers (writeJSON, writeJSONError, decodeJSON) that
//     every handler uses. Keeping them here avoids per-resource imports of
//     "encoding/json" and a uniform error shape across the API.
//
// Mutating endpoints keep the markdown files on disk as ground truth — they
// reuse the existing rewrite helpers (rewriteTaskFile, etc.) so the next
// SyncCache is a no-op.
package cmd

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// RegisterAPI wires every JSON endpoint backing the SPA on mux. Group reads
// first, mutations second — keeps the diff readable as we add endpoints.
// Path patterns use Go 1.22+ method-and-pattern syntax; "{id}" captures are
// retrieved by the handler via r.PathValue.
func RegisterAPI(mux *http.ServeMux) {
	// Reads.
	mux.HandleFunc("GET /api/meta", apiMetaHandler)
	mux.HandleFunc("GET /api/stats", apiStatsHandler)
	mux.HandleFunc("GET /api/specs", apiListSpecsHandler)
	mux.HandleFunc("GET /api/specs/{id}", apiGetSpecHandler)
	mux.HandleFunc("GET /api/tasks", apiListTasksHandler)
	mux.HandleFunc("GET /api/tasks/board", apiBoardHandler)
	mux.HandleFunc("GET /api/tasks/{id}", apiTaskDetailHandler)
	mux.HandleFunc("GET /api/kb", apiKBListHandler)
	mux.HandleFunc("GET /api/kb/{id}", apiKBDetailHandler)
	mux.HandleFunc("GET /api/search", apiSearchHandler)

	// Mutations.
	mux.HandleFunc("POST /api/tasks/move", apiMoveTaskHandler)
	mux.HandleFunc("POST /api/tasks/{id}/criteria/{position}/toggle", apiToggleCriterionHandler)
	mux.HandleFunc("PUT /api/tasks/{id}/depends_on", apiSetTaskDependsOnHandler)
	mux.HandleFunc("DELETE /api/tasks/{id}", apiDeleteTaskHandler)
	mux.HandleFunc("PUT /api/specs/{id}/linked_specs", apiSetLinkedSpecsHandler)
	mux.HandleFunc("DELETE /api/specs/{id}/linked_specs/{linkedId}", apiUnlinkSpecHandler)
	mux.HandleFunc("POST /api/settings/default-route", apiSetDefaultRouteHandler)
}

// ---------------------------------------------------------------------------
// JSON helpers (shared by every handler)
// ---------------------------------------------------------------------------

const (
	errCodeBadRequest = "bad_request"
	errCodeNotFound   = "not_found"
	errCodeInternal   = "internal"
)

// writeJSON serializes payload as JSON and writes it with the given status.
// Encoding errors are logged; they cannot be surfaced once headers are flushed.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("api: encode response", "error", err)
	}
}

// writeJSONError writes a uniform `{"error":"...","code":"..."}` body with the
// given HTTP status. The code is derived from the status when possible so the
// SPA can switch on a stable string instead of HTTP numbers.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	code := errCodeInternal
	switch status {
	case http.StatusBadRequest:
		code = errCodeBadRequest
	case http.StatusNotFound:
		code = errCodeNotFound
	}
	writeJSON(w, status, map[string]string{"error": message, "code": code})
}

// decodeJSON reads a JSON body into a value of type T, capping the body via
// http.MaxBytesReader (gosec G120) and rejecting unknown fields so a typo'd
// payload field surfaces as a 400 instead of a silent no-op.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, MaxSettingsFormBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

// inPlaceholders returns "?,?,…" with `n` placeholders, joined by commas.
// Used by batch loaders that expand a known-size []string into a single
// `WHERE id IN (?,?,…)` query rather than looping per ID. Returns "" for
// n <= 0; the caller is expected to short-circuit before using it.
func inPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	s := strings.Repeat("?,", n)
	return s[:len(s)-1]
}
