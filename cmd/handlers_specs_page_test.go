package cmd

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// specsPageTestFS returns a minimal template FS exercising SpecsPageData.
func specsPageTestFS() fstest.MapFS {
	fs := testTemplateFS()
	fs["pages/specs.html"] = &fstest.MapFile{Data: []byte(
		`{{define "content"}}<section>` +
			`{{with .Data}}` +
			`<p id="meta">VIEW:{{.View}};TOTAL:{{.TotalCount}};PAGE:{{.Page}}/{{.TotalPages}}</p>` +
			`{{if eq .View "flat"}}` +
			`<table id="flat">{{range .Items}}<tr><td>{{.ID}}</td><td>{{.Type}}</td><td>{{.Title}}</td></tr>{{end}}</table>` +
			`{{else}}` +
			`{{range .Groups}}<section class="group" data-type="{{.Type}}">` +
			`{{range .Items}}<article>{{.ID}}:{{.Title}}</article>{{end}}` +
			`</section>{{end}}` +
			`{{end}}` +
			`{{if .HasPrev}}<a id="prev" href="?page={{.PrevPage}}">prev</a>{{end}}` +
			`{{if .HasNext}}<a id="next" href="?page={{.NextPage}}">next</a>{{end}}` +
			`{{end}}` +
			`</section>{{end}}`,
	)}
	return fs
}

// setupSpecsPageProject seeds the project with specs of mixed types and
// returns parsed templates wired for the specs page tests.
func setupSpecsPageProject(t *testing.T, items []specSeed) map[string]*template.Template {
	t.Helper()
	tmp := t.TempDir()
	specTypes := []string{"business", "functional", "non_functional"}
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
	for i, s := range items {
		mustExec(t, db, fmt.Sprintf(
			`INSERT INTO specs (id, title, type, summary, body, path, content_hash, position, created_at, updated_at)
			 VALUES ('%s', '%s', '%s', '%s', 'body', 'p%d', 'h%d', %d, '2025-01-01', '2025-01-01')`,
			s.ID, s.Title, s.Type, s.Summary, i, i, i,
		))
	}
	_ = db.Close()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	pages, err := parseTemplates(specsPageTestFS())
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	return pages
}

type specSeed struct {
	ID, Title, Type, Summary string
}

func newSpecsHandler(pages map[string]*template.Template) http.HandlerFunc {
	return makeSpecsHandler(func() map[string]*template.Template { return pages }, false, "", "")
}

func TestSpecsPageDefaultsToGrouped(t *testing.T) {
	pages := setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "B One", Type: "business", Summary: "s1"},
		{ID: "SPEC-2", Title: "F One", Type: "functional", Summary: "s2"},
	})
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "VIEW:grouped") {
		t.Errorf("expected default view=grouped: %s", body)
	}
	if !strings.Contains(body, `data-type="business"`) || !strings.Contains(body, `data-type="functional"`) {
		t.Errorf("expected per-type groups: %s", body)
	}
}

func TestSpecsPageFlatView(t *testing.T) {
	pages := setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "T1", Type: "business", Summary: "s"},
		{ID: "SPEC-2", Title: "T2", Type: "functional", Summary: "s"},
	})
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs?view=flat", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "VIEW:flat") {
		t.Errorf("expected view=flat: %s", body)
	}
	if !strings.Contains(body, `id="flat"`) {
		t.Errorf("expected flat table render: %s", body)
	}
	if strings.Contains(body, `class="group"`) {
		t.Errorf("flat view should not render type groups: %s", body)
	}
}

func TestSpecsPageInvalidViewFallsBackToGrouped(t *testing.T) {
	pages := setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "T1", Type: "business", Summary: "s"},
	})
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs?view=evil", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	if !strings.Contains(w.Body.String(), "VIEW:grouped") {
		t.Errorf("expected fallback to grouped view")
	}
}

func TestSpecsPagePagination(t *testing.T) {
	var seeds []specSeed
	for i := 1; i <= 25; i++ {
		seeds = append(seeds, specSeed{
			ID:      fmt.Sprintf("SPEC-%d", i),
			Title:   fmt.Sprintf("Title %d", i),
			Type:    "business",
			Summary: "s",
		})
	}
	pages := setupSpecsPageProject(t, seeds)
	h := newSpecsHandler(pages)

	// Page 1 with page_size=10: HasNext, no HasPrev.
	req := httptest.NewRequest(http.MethodGet, "/specs?page=1&page_size=10", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:25;PAGE:1/3") {
		t.Errorf("expected page 1 of 3 with 25 items: %s", body)
	}
	if !strings.Contains(body, `id="next"`) || strings.Contains(body, `id="prev"`) {
		t.Errorf("expected next without prev on page 1: %s", body)
	}

	// Last page: HasPrev, no HasNext.
	req = httptest.NewRequest(http.MethodGet, "/specs?page=3&page_size=10", http.NoBody)
	w = httptest.NewRecorder()
	h(w, req)
	body = w.Body.String()
	if !strings.Contains(body, `id="prev"`) || strings.Contains(body, `id="next"`) {
		t.Errorf("expected prev without next on last page: %s", body)
	}
}

func TestSpecsPageEmpty(t *testing.T) {
	pages := setupSpecsPageProject(t, nil)
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:0") {
		t.Errorf("expected 0 totals: %s", body)
	}
}

func TestGroupSpecsByType(t *testing.T) {
	in := []ListSpecItem{
		{ID: "SPEC-1", Type: "business"},
		{ID: "SPEC-2", Type: "business"},
		{ID: "SPEC-3", Type: "functional"},
		{ID: "SPEC-4", Type: "non_functional"},
		{ID: "SPEC-5", Type: "non_functional"},
	}
	totals := map[string]int{"business": 5, "functional": 2, "non_functional": 9}
	groups := groupSpecsByType(in, totals)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if groups[0].Type != "business" || len(groups[0].Items) != 2 || groups[0].Total != 5 {
		t.Errorf("group 0: %+v", groups[0])
	}
	if groups[1].Type != "functional" || len(groups[1].Items) != 1 || groups[1].Total != 2 {
		t.Errorf("group 1: %+v", groups[1])
	}
	if groups[2].Type != "non_functional" || len(groups[2].Items) != 2 || groups[2].Total != 9 {
		t.Errorf("group 2: %+v", groups[2])
	}
}

func TestGroupSpecsByTypeEmpty(t *testing.T) {
	if got := groupSpecsByType(nil, nil); got != nil {
		t.Errorf("expected nil for empty input, got %+v", got)
	}
}

func TestIsAllowedSpecType(t *testing.T) {
	types := []string{"business", "functional"}
	for _, ok := range []string{"all", "business", "functional"} {
		if !isAllowedSpecType(ok, types) {
			t.Errorf("expected %q allowed", ok)
		}
	}
	for _, bad := range []string{"", "evil", "non_functional", "ALL"} {
		if isAllowedSpecType(bad, types) {
			t.Errorf("expected %q rejected", bad)
		}
	}
}

func TestSpecsPageTypeFilterIncludesOnlyMatching(t *testing.T) {
	pages := setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "B", Type: "business", Summary: "s"},
		{ID: "SPEC-2", Title: "F", Type: "functional", Summary: "s"},
		{ID: "SPEC-3", Title: "N", Type: "non_functional", Summary: "s"},
	})
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs?type=business", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:1") {
		t.Errorf("expected TOTAL:1 with type=business, got: %s", body)
	}
	if !strings.Contains(body, `data-type="business"`) {
		t.Errorf("expected business group rendered: %s", body)
	}
	for _, otherType := range []string{`data-type="functional"`, `data-type="non_functional"`} {
		if strings.Contains(body, otherType) {
			t.Errorf("filter leaked %q: %s", otherType, body)
		}
	}
}

func TestSpecsPageTypeFilterAllReturnsEverything(t *testing.T) {
	pages := setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "B", Type: "business", Summary: "s"},
		{ID: "SPEC-2", Title: "F", Type: "functional", Summary: "s"},
	})
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs?type=all", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:2") {
		t.Errorf("expected TOTAL:2 with type=all, got: %s", body)
	}
}

func TestSpecsPageTypeFilterInvalidFallsBackToAll(t *testing.T) {
	pages := setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "B", Type: "business", Summary: "s"},
		{ID: "SPEC-2", Title: "F", Type: "functional", Summary: "s"},
	})
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs?type=evil", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "TOTAL:2") {
		t.Errorf("expected invalid type to fall back to all (TOTAL:2): %s", body)
	}
}

func TestLoadSpecTypeTotalsFiltered(t *testing.T) {
	setupSpecsPageProject(t, []specSeed{
		{ID: "SPEC-1", Title: "B1", Type: "business", Summary: "s"},
		{ID: "SPEC-2", Title: "B2", Type: "business", Summary: "s"},
		{ID: "SPEC-3", Title: "F1", Type: "functional", Summary: "s"},
	})

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	all, err := loadSpecTypeTotals(db, SpecsTypeAll)
	if err != nil {
		t.Fatalf("loadSpecTypeTotals all: %v", err)
	}
	if all["business"] != 2 || all["functional"] != 1 {
		t.Errorf("unexpected totals when unfiltered: %+v", all)
	}

	only, err := loadSpecTypeTotals(db, "business")
	if err != nil {
		t.Fatalf("loadSpecTypeTotals filtered: %v", err)
	}
	if only["business"] != 2 {
		t.Errorf("expected business=2 when filtered, got: %+v", only)
	}
	if _, present := only["functional"]; present {
		t.Errorf("expected functional missing when filtered to business: %+v", only)
	}
}

func TestSpecsPageTypeFilterPersistsAcrossPagination(t *testing.T) {
	var seeds []specSeed
	for i := 1; i <= 12; i++ {
		seeds = append(seeds, specSeed{
			ID:      fmt.Sprintf("SPEC-%d", i),
			Title:   fmt.Sprintf("T%d", i),
			Type:    "business",
			Summary: "s",
		})
	}
	// Add one off-type spec so a working filter must drop it from totals.
	seeds = append(seeds, specSeed{ID: "SPEC-X", Title: "F", Type: "functional", Summary: "s"})
	pages := setupSpecsPageProject(t, seeds)
	h := newSpecsHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs?type=business&page=2&page_size=5", http.NoBody)
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	// 12 business specs / 5 per page = 3 pages, page 2 of 3.
	if !strings.Contains(body, "TOTAL:12;PAGE:2/3") {
		t.Errorf("expected page 2/3 of 12 filtered specs, got: %s", body)
	}
	if strings.Contains(body, "SPEC-X") {
		t.Errorf("functional spec leaked into business filter: %s", body)
	}
}
