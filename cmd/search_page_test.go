package cmd

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// searchPageTestFS returns a minimal template FS that exercises the
// SearchPageData fields the real template depends on.
func searchPageTestFS() fstest.MapFS {
	fs := testTemplateFS()
	fs["pages/search.html"] = &fstest.MapFile{Data: []byte(
		`{{define "content"}}<section>` +
			`<form id="search-form">` +
			`<input name="q" value="{{.Data.Query}}">` +
			`<input name="kind" value="{{.Data.Kind}}">` +
			`</form>` +
			`<div id="search-results">` +
			`{{if .Data.HasRun}}` +
			`{{if eq .Data.TotalCount 0}}<p>NO_RESULTS</p>{{else}}` +
			`<p>TOTAL:{{.Data.TotalCount}};PAGE:{{.Data.Page}}/{{.Data.TotalPages}}</p>` +
			`{{range .Data.Items}}<article>{{.Kind}}:{{.ID}}:{{.Title}}</article>{{end}}` +
			`{{if .Data.HasPrev}}<a id="prev" href="?page={{.Data.PrevPage}}">prev</a>{{end}}` +
			`{{if .Data.HasNext}}<a id="next" href="?page={{.Data.NextPage}}">next</a>{{end}}` +
			`{{end}}{{end}}` +
			`</div></section>{{end}}`,
	)}
	return fs
}

// setupSearchPageProject creates a fresh project, chdirs into it, seeds the
// given number of specs (each containing the word "authentication" in the
// body so they all match), and returns the parsed pages map.
func setupSearchPageProject(t *testing.T, specCount int) map[string]*template.Template {
	t.Helper()
	tmp := t.TempDir()
	specTypes := []string{"business"}
	taskStages := []string{"backlog", "done"}
	if err := InitDB(tmp, specTypes, taskStages); err != nil {
		t.Fatal(err)
	}
	if err := SaveProjectConfig(tmp, &ProjectConfig{
		Dir:        "specd",
		SpecTypes:  specTypes,
		TaskStages: taskStages,
	}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= specCount; i++ {
		mustExec(t, db, fmt.Sprintf(`INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
			VALUES ('SPEC-%d', 'Authentication Spec %d', 'business', 'authentication summary %d', 'authentication body %d', 'p%d', 'h%d', '2025-01-01', '2025-01-01')`,
			i, i, i, i, i, i))
	}
	_ = db.Close()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	pages, err := parseTemplates(searchPageTestFS())
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	return pages
}

func newSearchHandler(pages map[string]*template.Template) http.HandlerFunc {
	return makeSearchHandler(func() map[string]*template.Template { return pages }, false, "", "")
}

func TestSearchPageEmptyQuery(t *testing.T) {
	pages := setupSearchPageProject(t, 1)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `id="search-form"`) {
		t.Error("expected form on empty query")
	}
	if strings.Contains(body, "NO_RESULTS") || strings.Contains(body, "TOTAL:") {
		t.Error("expected no results section to render before a query")
	}
}

func TestSearchPageHappyPath(t *testing.T) {
	pages := setupSearchPageProject(t, 1)
	h := newSearchHandler(pages)

	q := url.Values{"q": {"authentication"}, "kind": {"spec"}}
	req := httptest.NewRequest(http.MethodGet, "/search?"+q.Encode(), http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "spec:SPEC-1:Authentication Spec 1") {
		t.Errorf("expected SPEC-1 in results, got %s", body)
	}
}

func TestSearchPageInvalidKindFallsBackToAll(t *testing.T) {
	pages := setupSearchPageProject(t, 1)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=auth&kind=bogus", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `value="all"`) {
		t.Errorf("expected kind to fall back to all, got %s", body)
	}
}

func TestSearchPageNoMatches(t *testing.T) {
	pages := setupSearchPageProject(t, 1)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=zzznotfoundzzz", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "NO_RESULTS") {
		t.Error("expected NO_RESULTS marker for empty result set")
	}
}

func TestSearchPagePaginationFirstPage(t *testing.T) {
	// 25 specs / searchPageDefaultSize (10) → 3 pages.
	pages := setupSearchPageProject(t, 25)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=authentication&kind=spec", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:25;PAGE:1/3") {
		t.Errorf("expected TOTAL:25;PAGE:1/3, body: %s", body)
	}
	if !strings.Contains(body, `id="next"`) {
		t.Error("expected next link on first page")
	}
	if strings.Contains(body, `id="prev"`) {
		t.Error("did not expect prev link on first page")
	}
	if got := strings.Count(body, "<article>"); got != searchPageDefaultSize {
		t.Errorf("expected %d items on page 1, got %d", searchPageDefaultSize, got)
	}
}

func TestSearchPagePaginationLastPage(t *testing.T) {
	pages := setupSearchPageProject(t, 25)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=authentication&kind=spec&page=3", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:25;PAGE:3/3") {
		t.Errorf("expected page 3/3, body: %s", body)
	}
	if !strings.Contains(body, `id="prev"`) {
		t.Error("expected prev link on last page")
	}
	if strings.Contains(body, `id="next"`) {
		t.Error("did not expect next link on last page")
	}
	// Last page has 25 - 20 = 5 items.
	if got := strings.Count(body, "<article>"); got != 5 {
		t.Errorf("expected 5 items on last page, got %d", got)
	}
}

func TestSearchPagePaginationOutOfRangeClamped(t *testing.T) {
	pages := setupSearchPageProject(t, 25)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=authentication&kind=spec&page=99", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if !strings.Contains(w.Body.String(), "TOTAL:25;PAGE:3/3") {
		t.Errorf("expected page clamped to last page, body: %s", w.Body.String())
	}
}

func TestSearchPagePageSizeOverride(t *testing.T) {
	pages := setupSearchPageProject(t, 7)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=authentication&kind=spec&page_size=3", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:7;PAGE:1/3") {
		t.Errorf("expected page_size=3 to yield 3 pages, body: %s", body)
	}
	if got := strings.Count(body, "<article>"); got != 3 {
		t.Errorf("expected 3 items on page 1, got %d", got)
	}
}

func TestSearchPagePageSizeCapped(t *testing.T) {
	pages := setupSearchPageProject(t, 5)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=authentication&kind=spec&page_size=9999", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	// page_size > searchPageMaxSize is clamped; with only 5 results we expect 1 page.
	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:5;PAGE:1/1") {
		t.Errorf("expected single-page render after cap, body: %s", body)
	}
}

func TestSearchPageHtmxRendersPartial(t *testing.T) {
	pages := setupSearchPageProject(t, 1)
	h := newSearchHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/search?q=authentication&kind=spec", http.NoBody)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("htmx response should not include the full document")
	}
	if !strings.Contains(body, `id="search-results"`) {
		t.Error("htmx response must contain #search-results so hx-select can target it")
	}
}

func TestPageWindow(t *testing.T) {
	cases := []struct {
		current, total, max int
		want                []int
	}{
		{1, 3, 5, []int{1, 2, 3}},
		{1, 10, 5, []int{1, 2, 3, 4, 5}},
		{5, 10, 5, []int{3, 4, 5, 6, 7}},
		{10, 10, 5, []int{6, 7, 8, 9, 10}},
		{1, 0, 5, nil},
	}
	for _, c := range cases {
		got := pageWindow(c.current, c.total, c.max)
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("pageWindow(%d,%d,%d) = %v, want %v", c.current, c.total, c.max, got, c.want)
		}
	}
}
