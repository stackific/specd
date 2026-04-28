// api_search.go implements GET /api/search. Hybrid BM25 + trigram search
// across specs, tasks, and KB docs, with kind filtering and pagination.
// The search engine itself lives in search.go; this file only wraps it for
// the SPA.
package cmd

import (
	"log/slog"
	"net/http"
	"strings"
)

// apiSearchResponse is the payload returned by GET /api/search.
type apiSearchResponse struct {
	Query      string         `json:"query"`
	Kind       string         `json:"kind"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
	Items      []SearchResult `json:"items"`
	PageSpecs  []SearchResult `json:"page_specs"`
	PageTasks  []SearchResult `json:"page_tasks"`
	PageKB     []SearchResult `json:"page_kb"`
}

// apiSearchHandler implements GET /api/search. Mirrors the merge/paginate
// behavior of the legacy search page via the shared applyPagination helper.
func apiSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	kind := q.Get("kind")
	if !validSearchKinds[kind] {
		kind = KindAll
	}
	page := parsePositiveIntParam(q.Get("page"), 1)
	pageSize := parsePositiveIntParam(q.Get("page_size"), searchPageDefaultSize)
	if pageSize > searchPageMaxSize {
		pageSize = searchPageMaxSize
	}

	view := SearchPageData{
		Query:    query,
		Kind:     kind,
		Page:     page,
		PageSize: pageSize,
	}

	if query != "" {
		results, err := runSearchForPage(query, kind)
		if err != nil {
			slog.Error("api search", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "search failed")
			return
		}
		view.HasRun = true
		applyPagination(&view, results)
	}

	writeJSON(w, http.StatusOK, apiSearchResponse{
		Query:      view.Query,
		Kind:       view.Kind,
		Page:       view.Page,
		PageSize:   view.PageSize,
		TotalCount: view.TotalCount,
		TotalPages: view.TotalPages,
		Items:      orEmptyResults(view.Items),
		PageSpecs:  orEmptyResults(view.PageSpecs),
		PageTasks:  orEmptyResults(view.PageTasks),
		PageKB:     orEmptyResults(view.PageKB),
	})
}

// orEmptyResults returns an empty slice when results is nil so the JSON
// payload is `[]` instead of `null`.
func orEmptyResults(results []SearchResult) []SearchResult {
	if results == nil {
		return []SearchResult{}
	}
	return results
}
