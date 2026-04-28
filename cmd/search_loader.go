// search_loader.go contains the merge/paginate logic that backs /api/search.
// The legacy GET /search HTML handler was removed; only the shared helpers
// remain.
package cmd

import (
	"sort"
	"strconv"
)

// searchPageMaxPerKind is the upper bound of per-kind results we pull from
// the search backend before merging and paginating.
const searchPageMaxPerKind = 100

// searchPageDefaultSize is the default number of results per page on the
// search page.
const searchPageDefaultSize = 10

// searchPageMaxSize caps user-supplied page_size to keep render cost bounded.
const searchPageMaxSize = 50

// SearchPageData is the view model returned by applyPagination and serialized
// by /api/search.
type SearchPageData struct {
	Query      string
	Kind       string
	HasRun     bool
	Items      []SearchResult // current page of merged, ranked results
	PageSpecs  []SearchResult // current-page items where Kind == spec
	PageTasks  []SearchResult // current-page items where Kind == task
	PageKB     []SearchResult // current-page items where Kind == kb
	Page       int
	PageSize   int
	TotalCount int
	TotalPages int
}

// validSearchKinds enumerates the kind filter values accepted by the page.
var validSearchKinds = map[string]bool{
	KindAll:  true,
	KindSpec: true,
	KindTask: true,
	KindKB:   true,
}

// parsePositiveIntParam parses s as a positive integer, returning fallback
// on empty input or any parse error.
func parsePositiveIntParam(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

// applyPagination merges per-kind results, sorts them by relevance, and
// populates the pagination fields on view.
func applyPagination(view *SearchPageData, results *SearchResults) {
	merged := make([]SearchResult, 0, len(results.Specs)+len(results.Tasks)+len(results.KB))
	merged = append(merged, results.Specs...)
	merged = append(merged, results.Tasks...)
	merged = append(merged, results.KB...)

	// Stable sort: BM25 hits (positive Score) before trigram (Score == 0),
	// then by score descending within each group. Stable sort preserves
	// per-kind ordering as a tiebreaker.
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	view.TotalCount = len(merged)
	if view.TotalCount == 0 {
		view.Items = nil
		view.TotalPages = 0
		return
	}

	view.TotalPages = (view.TotalCount + view.PageSize - 1) / view.PageSize
	if view.Page > view.TotalPages {
		view.Page = view.TotalPages
	}

	start := (view.Page - 1) * view.PageSize
	end := start + view.PageSize
	if end > view.TotalCount {
		end = view.TotalCount
	}
	view.Items = merged[start:end]
	for _, r := range view.Items {
		switch r.Kind {
		case KindSpec:
			view.PageSpecs = append(view.PageSpecs, r)
		case KindTask:
			view.PageTasks = append(view.PageTasks, r)
		case KindKB:
			view.PageKB = append(view.PageKB, r)
		}
	}
}

// runSearchForPage opens the project DB, resolves weights from the project
// config (falling back to defaults), and runs the search with a generous
// per-kind cap so the handler has enough data to paginate.
func runSearchForPage(query, kind string) (*SearchResults, error) {
	db, _, err := OpenProjectDB()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	weights := defaultSearchWeights()
	if proj, perr := LoadProjectConfig("."); perr == nil && proj != nil {
		if proj.SearchWeights.Title != 0 || proj.SearchWeights.Summary != 0 || proj.SearchWeights.Body != 0 {
			weights = proj.SearchWeights
		}
	}

	return Search(db, query, kind, searchPageMaxPerKind, "", weights)
}
