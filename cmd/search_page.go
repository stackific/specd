// search_page.go implements the GET /search page handler. It reads the
// query string, runs hybrid BM25 + trigram search via Search(), merges and
// paginates the per-kind results, and renders the search page template.
// Form submissions reuse the same route via htmx, with hx-select narrowing
// the swap to the results region.
package cmd

import (
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// searchPageMaxPerKind is the upper bound of per-kind results we pull from
// the search backend before merging and paginating client-side. It is large
// enough to cover any reasonable page count for typical specd projects.
const searchPageMaxPerKind = 100

// searchPageDefaultSize is the default number of results per page on the
// search page. Smaller than DefaultPageSize because search rows are denser
// than list-spec rows.
const searchPageDefaultSize = 10

// searchPageMaxSize caps user-supplied page_size to keep render cost bounded.
const searchPageMaxSize = 50

// SearchPageData is the view model passed to the search page template.
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
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	PageNums   []int // 1-based page numbers to render in the nav
}

// validSearchKinds enumerates the kind filter values accepted by the page.
var validSearchKinds = map[string]bool{
	KindAll:  true,
	KindSpec: true,
	KindTask: true,
	KindKB:   true,
}

// makeSearchHandler returns an http.HandlerFunc that renders the search page.
// freshPages mirrors the dev-mode pattern from runServe so live reload works.
func makeSearchHandler(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
				slog.Error("search page", "error", err)
				http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			view.HasRun = true
			applyPagination(&view, results)
		}

		renderPage(w, r, freshPages(), "search", &PageData{
			Title:   "Search",
			Active:  "search",
			DevMode: devMode,
			CSSHash: cssHash,
			JSHash:  jsHash,
			Data:    view,
		})
	}
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

	view.HasPrev = view.Page > 1
	view.HasNext = view.Page < view.TotalPages
	view.PrevPage = view.Page - 1
	view.NextPage = view.Page + 1
	view.PageNums = pageWindow(view.Page, view.TotalPages, 5)
}

// pageWindow returns up to `window` page numbers centered on `current`,
// clamped to [1, total]. Useful for rendering numbered pagination.
func pageWindow(current, total, window int) []int {
	if total <= 0 {
		return nil
	}
	if total <= window {
		nums := make([]int, total)
		for i := range nums {
			nums[i] = i + 1
		}
		return nums
	}
	half := window / 2
	start := current - half
	if start < 1 {
		start = 1
	}
	end := start + window - 1
	if end > total {
		end = total
		start = end - window + 1
	}
	nums := make([]int, 0, window)
	for i := start; i <= end; i++ {
		nums = append(nums, i)
	}
	return nums
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
