// api_specs.go implements GET /api/specs, GET /api/specs/{id}, and the
// DELETE endpoint that removes a single linked-spec relationship. Spec
// creation and edits still go through the CLI; the only mutation here is
// the bidirectional unlink so the SPA can show a "remove this link" button
// without round-tripping through `specd update-spec`.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// apiSpecsResponse is the payload returned by GET /api/specs.
type apiSpecsResponse struct {
	View       string         `json:"view"`
	Type       string         `json:"type"`
	Types      []string       `json:"types"`
	Items      []ListSpecItem `json:"items"`
	Groups     []SpecsGroup   `json:"groups"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
}

// apiListSpecsHandler implements GET /api/specs. Mirrors the query-parameter
// parsing done by makeSpecsHandler so the SPA can request the same data the
// HTML page renders, then strips the view-only pagination helpers.
func apiListSpecsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	view := q.Get("view")
	if !validSpecsViews[view] {
		view = SpecsViewGrouped
	}
	page := parsePositiveIntParam(q.Get("page"), 1)
	pageSize := parsePositiveIntParam(q.Get("page_size"), specsPageDefaultSize)
	if pageSize > specsPageMaxSize {
		pageSize = specsPageMaxSize
	}

	types := configuredSpecTypes()
	typeFilter := q.Get("type")
	if !isAllowedSpecType(typeFilter, types) {
		typeFilter = SpecsTypeAll
	}

	data := SpecsPageData{
		View:     view,
		Type:     typeFilter,
		Types:    types,
		Page:     page,
		PageSize: pageSize,
	}

	if err := loadSpecsPage(&data); err != nil {
		slog.Error("api specs", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load specs")
		return
	}

	if data.Items == nil {
		data.Items = []ListSpecItem{}
	}
	if data.Groups == nil {
		data.Groups = []SpecsGroup{}
	}

	writeJSON(w, http.StatusOK, apiSpecsResponse(data))
}

// apiSpecRef is a {id, title, summary} triple used to expand id-only
// references on the spec detail (linked_specs). Mirrors apiTaskRef in
// shape; the SPA renders id + title + a summary snippet so a row in the
// "Linked specs" card can stand on its own without a second fetch.
type apiSpecRef struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// apiSpecDetailResponse wraps GetSpecResponse with sibling *_refs fields
// resolved server-side. The embedded GetSpecResponse keeps every existing
// field at the JSON root so older SPA code that read `data.foo` directly
// continues to work — only the new `linked_specs_refs` is additive.
type apiSpecDetailResponse struct {
	*GetSpecResponse
	LinkedSpecsRefs []apiSpecRef `json:"linked_specs_refs"`
}

// loadSpecRefs returns one apiSpecRef per ID in input order. Missing IDs
// fall back to {ID, "", ""} so the UI can still render a broken link
// instead of dropping it. Single batch query — never a loop.
func loadSpecRefs(db *sql.DB, ids []string) ([]apiSpecRef, error) {
	out := make([]apiSpecRef, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := batchQuerySpecRefs(db, ids)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	type pair struct{ title, summary string }
	cache := make(map[string]pair, len(ids))
	for rows.Next() {
		var id, title, summary string
		if err := rows.Scan(&id, &title, &summary); err != nil {
			return nil, err
		}
		cache[id] = pair{title, summary}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		p := cache[id]
		out = append(out, apiSpecRef{ID: id, Title: p.title, Summary: p.summary})
	}
	return out, nil
}

// batchQuerySpecRefs runs a single SELECT for the given spec IDs. Extracted
// only to keep the placeholder/args glue out of loadSpecRefs's main flow.
func batchQuerySpecRefs(db *sql.DB, ids []string) (*sql.Rows, error) {
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	q := "SELECT id, title, summary FROM specs WHERE id IN (" + inPlaceholders(len(ids)) + ")" //nolint:gosec // inPlaceholders only emits "?,?,..." — no caller input
	return db.Query(q, args...)                                                                //nolint:gosec // q is the line above; args are bound
}

// apiGetSpecHandler implements GET /api/specs/{id}.
func apiGetSpecHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.ToUpper(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing spec id")
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api spec detail: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	spec, err := LoadSpecDetail(db, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, "spec not found")
			return
		}
		slog.Error("api spec detail: load", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load spec")
		return
	}

	refs, err := loadSpecRefs(db, spec.LinkedSpecs)
	if err != nil {
		slog.Error("api spec detail: linked refs", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve linked specs")
		return
	}
	writeJSON(w, http.StatusOK, apiSpecDetailResponse{
		GetSpecResponse: spec,
		LinkedSpecsRefs: refs,
	})
}

// apiUnlinkSpecHandler implements DELETE /api/specs/{id}/linked_specs/{linkedId}.
// Removes the link in BOTH directions (the spec_links table is bidirectional)
// and rewrites both spec markdown files so frontmatter `linked_specs:` matches
// the DB. Returns the freshly-loaded spec detail for the source ID so the
// SPA can re-render off the response.
func apiUnlinkSpecHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.ToUpper(r.PathValue("id"))
	linkedID := strings.ToUpper(r.PathValue("linkedId"))
	if id == "" || linkedID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing spec id")
		return
	}
	if id == linkedID {
		writeJSONError(w, http.StatusBadRequest, "spec cannot unlink itself")
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api spec unlink: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	// Verify both specs exist so we return clean 404s instead of silently
	// no-oping on the DELETE.
	for _, check := range []string{id, linkedID} {
		var exists int
		err := db.QueryRow("SELECT 1 FROM specs WHERE id = ?", check).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, fmt.Sprintf("spec not found: %s", check))
			return
		}
		if err != nil {
			slog.Error("api spec unlink: verify", "spec", check, "error", err) //nolint:gosec // check is id or linkedID, both validated above
			writeJSONError(w, http.StatusInternalServerError, "failed to verify spec")
			return
		}
	}

	// unlinkRelatedSpecs takes a comma-separated string and clears both
	// directions in a single call.
	if err := unlinkRelatedSpecs(db, id, linkedID); err != nil {
		slog.Error("api spec unlink: db update", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to unlink")
		return
	}

	// Rewrite both files so frontmatter matches the DB. A failure here is
	// non-fatal — the DB is the source of truth for sync, and the next
	// SyncCache will reconcile.
	for _, sid := range []string{id, linkedID} {
		if err := rewriteSpecFile(db, sid); err != nil {
			slog.Error("api spec unlink: rewrite", "spec", sid, "error", err) //nolint:gosec // sid is one of id/linkedID, both uppercased and existence-checked above
			w.Header().Add("X-Specd-Warning", "could not rewrite "+sid+".md")
		}
	}

	spec, err := LoadSpecDetail(db, id)
	if err != nil {
		slog.Error("api spec unlink: reload", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reload spec")
		return
	}
	refs, err := loadSpecRefs(db, spec.LinkedSpecs)
	if err != nil {
		slog.Error("api spec unlink: linked refs", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve linked specs")
		return
	}

	slog.Info("api.specs.unlink", "id", id, "unlinked", linkedID) //nolint:gosec // both IDs are validated above
	writeJSON(w, http.StatusOK, apiSpecDetailResponse{
		GetSpecResponse: spec,
		LinkedSpecsRefs: refs,
	})
}

// apiSetLinkedSpecsRequest is the body of PUT /api/specs/{id}/linked_specs.
// `LinkedSpecs` is the COMPLETE replacement set of related spec IDs. Send
// an empty array to clear all links. The handler computes the add/remove
// diff against the current rows so unrelated specs aren't disturbed.
type apiSetLinkedSpecsRequest struct {
	LinkedSpecs []string `json:"linked_specs"`
}

// normalizeLinkedSpecsInput uppercases, trims, and de-duplicates link target
// IDs. Returns an error on self-link or empty input slot. Pure helper —
// keeps the handler readable and testable in isolation.
func normalizeLinkedSpecsInput(specID string, raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		linkID := strings.ToUpper(strings.TrimSpace(item))
		if linkID == "" {
			continue
		}
		if linkID == specID {
			return nil, fmt.Errorf("spec cannot link to itself")
		}
		if seen[linkID] {
			continue
		}
		seen[linkID] = true
		out = append(out, linkID)
	}
	return out, nil
}

// diffLinkedSpecs returns the adds and removes needed to turn `current` into
// `next`. Order of `next` is preserved in the adds slice; removes preserve
// `current` order so log lines stay deterministic.
func diffLinkedSpecs(current, next []string) (toAdd, toRemove []string) {
	currentSet := make(map[string]bool, len(current))
	for _, c := range current {
		currentSet[c] = true
	}
	nextSet := make(map[string]bool, len(next))
	for _, n := range next {
		nextSet[n] = true
	}
	for _, n := range next {
		if !currentSet[n] {
			toAdd = append(toAdd, n)
		}
	}
	for _, c := range current {
		if !nextSet[c] {
			toRemove = append(toRemove, c)
		}
	}
	return toAdd, toRemove
}

// applyLinkedSpecsDiff inserts/deletes bidirectional spec_links rows for the
// given diff inside one transaction. Caller is responsible for ID
// validation; the schema's foreign keys catch stragglers.
func applyLinkedSpecsDiff(db *sql.DB, specID string, toAdd, toRemove []string) error {
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
	for _, target := range toAdd {
		if _, err := tx.Exec("INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)", specID, target); err != nil {
			return fmt.Errorf("insert link %s→%s: %w", specID, target, err)
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)", target, specID); err != nil {
			return fmt.Errorf("insert link %s→%s: %w", target, specID, err)
		}
	}
	for _, target := range toRemove {
		if _, err := tx.Exec("DELETE FROM spec_links WHERE from_spec = ? AND to_spec = ?", specID, target); err != nil {
			return fmt.Errorf("delete link %s→%s: %w", specID, target, err)
		}
		if _, err := tx.Exec("DELETE FROM spec_links WHERE from_spec = ? AND to_spec = ?", target, specID); err != nil {
			return fmt.Errorf("delete link %s→%s: %w", target, specID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true
	return nil
}

// apiSetLinkedSpecsHandler implements PUT /api/specs/{id}/linked_specs. The
// posted set is the new full list of related specs; the handler diffs against
// the current set, inserts/deletes bidirectional spec_links rows, then
// rewrites every affected spec.md so frontmatter `linked_specs:` matches the
// DB. Mirrors apiSetTaskDependsOnHandler in shape so the SPA can reuse the
// same picker pattern.
func apiSetLinkedSpecsHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.ToUpper(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing spec id")
		return
	}

	body, err := decodeJSON[apiSetLinkedSpecsRequest](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	next, err := normalizeLinkedSpecsInput(id, body.LinkedSpecs)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api set linked_specs: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	missing, verr := verifySpecsExist(db, append([]string{id}, next...))
	if verr != nil {
		slog.Error("api set linked_specs: verify", "error", verr)
		writeJSONError(w, http.StatusInternalServerError, "failed to verify spec ids")
		return
	}
	if missing == id {
		writeJSONError(w, http.StatusNotFound, "spec not found")
		return
	}
	if missing != "" {
		writeJSONError(w, http.StatusBadRequest, "unknown spec in linked_specs: "+missing)
		return
	}

	current, err := loadLinkedSpecs(db, id)
	if err != nil {
		slog.Error("api set linked_specs: load current", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load current links")
		return
	}
	toAdd, toRemove := diffLinkedSpecs(current, next)

	if err := applyLinkedSpecsDiff(db, id, toAdd, toRemove); err != nil {
		slog.Error("api set linked_specs: apply", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save linked_specs")
		return
	}

	// Rewrite every spec whose link list changed (subject + adds + removes).
	// File-rewrite failures are non-fatal — the next SyncCache reconciles —
	// surface them as a soft warning header.
	affected := append([]string{id}, toAdd...)
	affected = append(affected, toRemove...)
	for _, sid := range affected {
		if err := rewriteSpecFile(db, sid); err != nil {
			slog.Error("api set linked_specs: rewrite", "spec", sid, "error", err) //nolint:gosec // sid validated above
			w.Header().Add("X-Specd-Warning", "could not rewrite "+sid+".md")
		}
	}

	resp, status, err := buildSpecDetailResponse(db, id)
	if err != nil {
		slog.Error("api set linked_specs: reload", "error", err)
		writeJSONError(w, status, "failed to reload spec")
		return
	}
	slog.Info("api.specs.set_linked_specs", "id", id, "added", len(toAdd), "removed", len(toRemove)) //nolint:gosec // id validated above
	writeJSON(w, status, resp)
}

// buildSpecDetailResponse loads the full spec-detail payload for an id from
// an open DB connection. Returns (response, http-status, err) — caller
// surfaces the status code on error so 404s come through as 404s. Used by
// both GET /api/specs/{id} and PUT /api/specs/{id}/linked_specs so they
// emit the exact same shape.
func buildSpecDetailResponse(db *sql.DB, id string) (apiSpecDetailResponse, int, error) {
	spec, err := LoadSpecDetail(db, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return apiSpecDetailResponse{}, http.StatusNotFound, err
		}
		return apiSpecDetailResponse{}, http.StatusInternalServerError, fmt.Errorf("load spec: %w", err)
	}
	refs, err := loadSpecRefs(db, spec.LinkedSpecs)
	if err != nil {
		return apiSpecDetailResponse{}, http.StatusInternalServerError, fmt.Errorf("load linked refs: %w", err)
	}
	return apiSpecDetailResponse{
		GetSpecResponse: spec,
		LinkedSpecsRefs: refs,
	}, http.StatusOK, nil
}

// verifySpecsExist returns the first missing id (or "") so the caller can
// distinguish 404-on-subject from 400-on-target. Mirrors verifyTasksExist.
func verifySpecsExist(db *sql.DB, ids []string) (string, error) {
	for _, id := range ids {
		var exists int
		err := db.QueryRow("SELECT 1 FROM specs WHERE id = ?", id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return id, nil
		}
		if err != nil {
			return "", fmt.Errorf("verify spec %s: %w", id, err)
		}
	}
	return "", nil
}
