package cmd

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// specDetailTestFS returns a minimal template FS that exercises the fields
// LoadSpecDetail/SpecDetailPageData expose.
func specDetailTestFS() fstest.MapFS {
	fs := testTemplateFS()
	fs["pages/spec_detail.html"] = &fstest.MapFile{Data: []byte(
		`{{define "content"}}<section>` +
			`{{with .Data.Spec}}` +
			`<h3 id="title">{{.ID}}:{{.Title}}</h3>` +
			`<p id="type">{{.Type}}</p>` +
			`<p id="summary">{{.Summary}}</p>` +
			`<ul id="claims">{{range .Claims}}<li>{{.Position}}:{{.Text}}</li>{{end}}</ul>` +
			`<ul id="linked">{{range .LinkedSpecs}}<li>{{.}}</li>{{end}}</ul>` +
			`<ul id="tasks">{{range .Tasks}}<li>{{.ID}}:{{.Title}}:{{.Status}}</li>{{end}}</ul>` +
			`{{end}}` +
			`</section>{{end}}`,
	)}
	return fs
}

// setupSpecDetailProject seeds a project with one spec, two claims, one
// linked spec, and two child tasks; chdirs in.
func setupSpecDetailProject(t *testing.T) map[string]*template.Template {
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
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, position, created_by, created_at, updated_at)
		VALUES ('SPEC-1', 'Hero Spec', 'business', 'A summary', 'body text', 'p1', 'h1', 0, 'tester', '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, position, created_by, created_at, updated_at)
		VALUES ('SPEC-2', 'Other Spec', 'business', 's', 'b', 'p2', 'h2', 1, 'tester', '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO spec_claims (spec_id, position, text) VALUES ('SPEC-1', 1, 'must do A')`)
	mustExec(t, db, `INSERT INTO spec_claims (spec_id, position, text) VALUES ('SPEC-1', 2, 'should do B')`)
	mustExec(t, db, `INSERT INTO spec_links (from_spec, to_spec) VALUES ('SPEC-1', 'SPEC-2')`)
	mustExec(t, db, `INSERT INTO tasks (id, spec_id, title, status, summary, body, path, content_hash, position, created_at, updated_at)
		VALUES ('TASK-1', 'SPEC-1', 'Wire it up', 'backlog', 't1 summary', 'b', 'p1', 'h1', 0, '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO tasks (id, spec_id, title, status, summary, body, path, content_hash, position, created_at, updated_at)
		VALUES ('TASK-2', 'SPEC-1', 'Test it', 'done', 't2 summary', 'b', 'p2', 'h2', 1, '2025-01-01', '2025-01-01')`)
	_ = db.Close()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	pages, err := parseTemplates(specDetailTestFS())
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	return pages
}

func newSpecDetailHandler(pages map[string]*template.Template) http.HandlerFunc {
	return makeSpecDetailHandler(func() map[string]*template.Template { return pages }, false, "", "")
}

func TestLoadSpecDetailFound(t *testing.T) {
	setupSpecDetailProject(t)
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	spec, err := LoadSpecDetail(db, "SPEC-1")
	if err != nil {
		t.Fatalf("LoadSpecDetail: %v", err)
	}
	if spec.ID != "SPEC-1" || spec.Title != "Hero Spec" || spec.Type != "business" {
		t.Errorf("unexpected scalars: %+v", spec)
	}
	if len(spec.Claims) != 2 || spec.Claims[0].Text != "must do A" {
		t.Errorf("claims wrong: %+v", spec.Claims)
	}
	if len(spec.LinkedSpecs) != 1 || spec.LinkedSpecs[0] != "SPEC-2" {
		t.Errorf("linked specs wrong: %+v", spec.LinkedSpecs)
	}
	if len(spec.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d: %+v", len(spec.Tasks), spec.Tasks)
	}
	if spec.Tasks[0].ID != "TASK-1" || spec.Tasks[1].Status != "done" {
		t.Errorf("tasks wrong: %+v", spec.Tasks)
	}
}

func TestLoadSpecDetailNotFound(t *testing.T) {
	setupSpecDetailProject(t)
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	_, err = LoadSpecDetail(db, "SPEC-NOPE")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestSpecDetailHandlerOK(t *testing.T) {
	pages := setupSpecDetailProject(t)
	h := newSpecDetailHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs/SPEC-1", http.NoBody)
	req.SetPathValue("id", "SPEC-1")
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"SPEC-1:Hero Spec", "must do A", "SPEC-2", "TASK-1:Wire it up:backlog", "TASK-2:Test it:done"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q. got:\n%s", want, body)
		}
	}
}

func TestSpecDetailHandlerNotFound(t *testing.T) {
	pages := setupSpecDetailProject(t)
	h := newSpecDetailHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs/SPEC-99", http.NoBody)
	req.SetPathValue("id", "SPEC-99")
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSpecDetailHandlerEmptyID(t *testing.T) {
	pages := setupSpecDetailProject(t)
	h := newSpecDetailHandler(pages)

	req := httptest.NewRequest(http.MethodGet, "/specs/", http.NoBody)
	// PathValue defaults to "" when not set — exactly what we want to test.
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for empty id, got %d", w.Code)
	}
}

func TestSpecDetailHandlerSetsTitleHeader(t *testing.T) {
	// setupSpecDetailProject seeds the DB and chdirs in; we re-parse a fresh
	// template set here to verify the handler renders title/ID into the body
	// (the test FS doesn't render a <title> element).
	setupSpecDetailProject(t)
	pages, err := parseTemplates(specDetailTestFS())
	if err != nil {
		t.Fatal(err)
	}
	h := makeSpecDetailHandler(func() map[string]*template.Template { return pages }, false, "", "")

	req := httptest.NewRequest(http.MethodGet, "/specs/SPEC-1", http.NoBody)
	req.SetPathValue("id", "SPEC-1")
	w := httptest.NewRecorder()
	h(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "SPEC-1") || !strings.Contains(body, "Hero Spec") {
		t.Errorf("expected ID + title rendered: %s", body)
	}
}
