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

// taskCriteriaTestFS returns a minimal template FS that defines just the
// criteria-article partial so the toggle handler can render its response.
func taskCriteriaTestFS() fstest.MapFS {
	fs := testTemplateFS()
	fs["pages/task_detail.html"] = &fstest.MapFile{Data: []byte(
		`{{define "task-criteria-article"}}` +
			`<article id="task-criteria-article">` +
			`<span id="count">{{.CompletedCount}}/{{.TotalCriteria}}</span>` +
			`<ul>{{range .Task.Criteria}}` +
			`<li>{{.Position}}|{{.Text}}|{{.Checked}}</li>` +
			`{{end}}</ul>` +
			`</article>{{end}}` +
			// content block must exist or parseTemplates rejects the file.
			`{{define "content"}}<section></section>{{end}}`,
	)}
	return fs
}

// setupTaskCriteriaProject seeds a project with a single task whose body
// contains a 3-criterion `## Acceptance Criteria` block, persisted to a real
// markdown file so rewriteTaskFile can round-trip through the filesystem.
func setupTaskCriteriaProject(t *testing.T) (db *sql.DB, taskPath string, pages map[string]*template.Template) {
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

	specDir := filepath.Join(tmp, "specd", "specs", "spec-1")
	if err := os.MkdirAll(specDir, 0o750); err != nil {
		t.Fatal(err)
	}
	taskPath = filepath.Join(specDir, "TASK-1.md")
	body := "## Description\n\nDoes things.\n\n## Acceptance Criteria\n\n- [ ] One\n- [ ] Two\n- [ ] Three"
	taskMD := buildTaskMarkdown("TASK-1", "SPEC-1", "Wire it up", "summary", "backlog", "tester", "2025-01-01T00:00:00Z", 0, nil, nil, body)
	if err := os.WriteFile(taskPath, []byte(taskMD), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	d, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, d, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, position, created_at, updated_at)
		VALUES ('SPEC-1', 'S', 'business', 's', 'b', 'p', 'h', 0, '2025-01-01', '2025-01-01')`)
	if _, err := d.Exec(`INSERT INTO tasks (id, spec_id, title, status, summary, body, path, content_hash, position, created_by, created_at, updated_at)
		VALUES ('TASK-1', 'SPEC-1', 'Wire it up', 'backlog', 'summary', ?, ?, 'h', 0, 'tester', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`, body, taskPath); err != nil {
		t.Fatal(err)
	}
	mustExec(t, d, `INSERT INTO task_criteria (task_id, position, text, checked) VALUES ('TASK-1', 1, 'One', 0)`)
	mustExec(t, d, `INSERT INTO task_criteria (task_id, position, text, checked) VALUES ('TASK-1', 2, 'Two', 0)`)
	mustExec(t, d, `INSERT INTO task_criteria (task_id, position, text, checked) VALUES ('TASK-1', 3, 'Three', 0)`)

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close(); _ = os.Chdir(origDir) })

	pages, err = parseTemplates(taskCriteriaTestFS())
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	return d, taskPath, pages
}

func newCriteriaToggleHandler(pages map[string]*template.Template) http.HandlerFunc {
	return makeTaskCriteriaToggleHandler(func() map[string]*template.Template { return pages })
}

func readCriteriaChecked(t *testing.T, db *sql.DB, taskID string, pos int) int {
	t.Helper()
	var v int
	if err := db.QueryRow("SELECT checked FROM task_criteria WHERE task_id = ? AND position = ?", taskID, pos).Scan(&v); err != nil {
		t.Fatalf("read checked: %v", err)
	}
	return v
}

func TestFlipTaskCriterionToggles(t *testing.T) {
	db, _, _ := setupTaskCriteriaProject(t)

	if err := flipTaskCriterion(db, "TASK-1", 2); err != nil {
		t.Fatalf("flip: %v", err)
	}
	if got := readCriteriaChecked(t, db, "TASK-1", 2); got != 1 {
		t.Errorf("after first flip want checked=1, got %d", got)
	}
	if err := flipTaskCriterion(db, "TASK-1", 2); err != nil {
		t.Fatalf("flip back: %v", err)
	}
	if got := readCriteriaChecked(t, db, "TASK-1", 2); got != 0 {
		t.Errorf("after second flip want checked=0, got %d", got)
	}
}

func TestFlipTaskCriterionMissingRow(t *testing.T) {
	db, _, _ := setupTaskCriteriaProject(t)

	err := flipTaskCriterion(db, "TASK-1", 99)
	if err == nil {
		t.Fatal("expected error for missing position")
	}
	if err != sql.ErrNoRows { //nolint:errorlint // QueryRow returns sentinel directly here
		t.Errorf("want sql.ErrNoRows, got %v", err)
	}
}

func TestCriteriaToggleHandlerHappyPath(t *testing.T) {
	db, taskPath, pages := setupTaskCriteriaProject(t)
	h := newCriteriaToggleHandler(pages)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/TASK-1/criteria/2/toggle", http.NoBody)
	req.SetPathValue("id", "TASK-1")
	req.SetPathValue("position", "2")
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if got := readCriteriaChecked(t, db, "TASK-1", 2); got != 1 {
		t.Errorf("DB criterion should be checked, got %d", got)
	}

	// File on disk must reflect the new state.
	fileBytes, err := os.ReadFile(taskPath) //nolint:gosec // taskPath comes from t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	body := string(fileBytes)
	if !strings.Contains(body, "- [x] Two") {
		t.Errorf("markdown file should contain '- [x] Two'; got:\n%s", body)
	}
	if !strings.Contains(body, "- [ ] One") || !strings.Contains(body, "- [ ] Three") {
		t.Errorf("other criteria should remain unchecked; got:\n%s", body)
	}

	// Response is the criteria-article partial with the new count + state.
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `id="task-criteria-article"`) {
		t.Errorf("response should be the partial; got:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "1/3") {
		t.Errorf("count chip should show 1/3; got:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "2|Two|1") {
		t.Errorf("toggled criterion should render with checked=1; got:\n%s", bodyStr)
	}
}

func TestCriteriaToggleHandlerToggleBack(t *testing.T) {
	db, _, pages := setupTaskCriteriaProject(t)
	h := newCriteriaToggleHandler(pages)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/TASK-1/criteria/1/toggle", http.NoBody)
		req.SetPathValue("id", "TASK-1")
		req.SetPathValue("position", "1")
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("iteration %d: status %d", i, w.Code)
		}
	}
	if got := readCriteriaChecked(t, db, "TASK-1", 1); got != 0 {
		t.Errorf("after two toggles want checked=0, got %d", got)
	}
}

func TestCriteriaToggleHandlerInvalidTaskID(t *testing.T) {
	_, _, pages := setupTaskCriteriaProject(t)
	h := newCriteriaToggleHandler(pages)

	for _, id := range []string{"", "FOO-1"} { // wrong prefix or empty
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/X/criteria/1/toggle", http.NoBody)
		req.SetPathValue("id", id)
		req.SetPathValue("position", "1")
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("id=%q want 400, got %d", id, w.Code)
		}
	}

	// Lowercase task id is normalised to upper-case and accepted.
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/criteria/1/toggle", http.NoBody)
	req.SetPathValue("id", "task-1")
	req.SetPathValue("position", "1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code == http.StatusBadRequest {
		t.Errorf("lowercase task id should be normalised, got 400 (%s)", w.Body.String())
	}
}

func TestCriteriaToggleHandlerInvalidPosition(t *testing.T) {
	_, _, pages := setupTaskCriteriaProject(t)
	h := newCriteriaToggleHandler(pages)

	for _, pos := range []string{"", "abc", "0", "-1"} {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/TASK-1/criteria/X/toggle", http.NoBody)
		req.SetPathValue("id", "TASK-1")
		req.SetPathValue("position", pos)
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("position=%q want 400, got %d", pos, w.Code)
		}
	}
}

func TestCriteriaToggleHandlerNotFound(t *testing.T) {
	_, _, pages := setupTaskCriteriaProject(t)
	h := newCriteriaToggleHandler(pages)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/TASK-1/criteria/99/toggle", http.NoBody)
	req.SetPathValue("id", "TASK-1")
	req.SetPathValue("position", "99")
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("missing position should 404, got %d", w.Code)
	}
}

func TestCriteriaToggleHandlerHashUpdates(t *testing.T) {
	db, _, pages := setupTaskCriteriaProject(t)
	h := newCriteriaToggleHandler(pages)

	var hashBefore string
	if err := db.QueryRow("SELECT content_hash FROM tasks WHERE id = ?", "TASK-1").Scan(&hashBefore); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/TASK-1/criteria/1/toggle", http.NoBody)
	req.SetPathValue("id", "TASK-1")
	req.SetPathValue("position", "1")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}

	var hashAfter string
	if err := db.QueryRow("SELECT content_hash FROM tasks WHERE id = ?", "TASK-1").Scan(&hashAfter); err != nil {
		t.Fatal(err)
	}
	if hashAfter == "" || hashAfter == hashBefore {
		t.Errorf("content_hash should refresh; before=%q after=%q", hashBefore, hashAfter)
	}
}
