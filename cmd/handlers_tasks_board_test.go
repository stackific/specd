package cmd

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -- Pure-function unit tests -----------------------------------------------

func TestOrderedStagesPreferredFirst(t *testing.T) {
	// Configured set is intentionally out of order to confirm the preferred
	// list dictates layout when the stages are present.
	configured := []string{"backlog", "done", "in_progress", "blocked", "todo", "pending_verification"}
	got := orderedStages(configured)
	want := []string{"Backlog", "Todo", "In progress", "Blocked", "Pending Verification", "Done"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %q, want %q (%v)", i, got[i], want[i], got)
		}
	}
}

func TestOrderedStagesAppendsUnknownTrailingStages(t *testing.T) {
	configured := []string{"backlog", "todo", "in_progress", "done", "cancelled", "wont_fix"}
	got := orderedStages(configured)
	want := []string{"Backlog", "Todo", "In progress", "Done", "Cancelled", "Wont Fix"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOrderedStagesSkipsAbsentOptional(t *testing.T) {
	// Project that opted out of all optional stages: only required remain.
	configured := []string{"backlog", "todo", "in_progress", "done"}
	got := orderedStages(configured)
	want := []string{"Backlog", "Todo", "In progress", "Done"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, label := range want {
		if got[i] != label {
			t.Errorf("position %d: got %q, want %q", i, got[i], label)
		}
	}
}

func TestCompletedStageSlugsAtAndAfterDone(t *testing.T) {
	stages := []string{"Backlog", "Todo", "In progress", "Blocked", "Pending Verification", "Done", "Cancelled", "Wont Fix"}
	got := completedStageSlugs(stages)
	wantTrue := []string{"done", "cancelled", "wont_fix"}
	wantFalse := []string{"backlog", "todo", "in_progress", "blocked", "pending_verification"}
	for _, s := range wantTrue {
		if !got[s] {
			t.Errorf("expected %q to be completed", s)
		}
	}
	for _, s := range wantFalse {
		if got[s] {
			t.Errorf("expected %q to NOT be completed", s)
		}
	}
}

func TestCompletedStageSlugsCustomTrailingStage(t *testing.T) {
	// User keeps Done and adds a custom optional stage after it. The custom
	// stage must be picked up positionally — no hardcoded slug list.
	stages := []string{"Backlog", "Todo", "In progress", "Done", "Archived"}
	got := completedStageSlugs(stages)
	if !got["done"] {
		t.Error("done must be completed")
	}
	if !got["archived"] {
		t.Error("custom trailing stage 'archived' must be completed")
	}
	if got["todo"] {
		t.Error("todo must NOT be completed")
	}
}

func TestCompletedStageSlugsWithoutDone(t *testing.T) {
	// Defensive: a stages list without Done yields an empty completed set,
	// not a panic.
	got := completedStageSlugs([]string{"Backlog", "Todo"})
	if len(got) != 0 {
		t.Errorf("expected empty set, got %v", got)
	}
}

func TestNormalizeBoardFilter(t *testing.T) {
	cases := map[string]string{
		"":             BoardFilterAll,
		"all":          BoardFilterAll,
		"incomplete":   BoardFilterIncomplete,
		"garbage":      BoardFilterAll,
		"INCOMPLETE":   BoardFilterAll, // case-sensitive on purpose
		"  all       ": BoardFilterAll, // not trimmed; safer to reject
	}
	for in, want := range cases {
		if got := normalizeBoardFilter(in); got != want {
			t.Errorf("normalizeBoardFilter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidStatus(t *testing.T) {
	stages := []string{"Backlog", "Todo", "In progress", "Done"}
	if !validStatus(stages, "backlog") {
		t.Error("backlog must be valid")
	}
	if !validStatus(stages, "in_progress") {
		t.Error("in_progress must be valid")
	}
	if validStatus(stages, "wont_fix") {
		t.Error("unknown slug must be rejected")
	}
	if validStatus(stages, "") {
		t.Error("empty must be rejected")
	}
}

// -- moveTask integration tests ----------------------------------------------

// setupBoardProject seeds a project with three tasks under one spec, all in
// Backlog with positions 0, 1, 2.
func setupBoardProject(t *testing.T) (db *sql.DB, projDir string, taskFiles []string) {
	t.Helper()
	projDir = t.TempDir()
	specTypes := []string{"business"}
	stages := []string{"backlog", "todo", "in_progress", "done"}
	if err := InitDB(projDir, specTypes, stages); err != nil {
		t.Fatal(err)
	}
	if err := SaveProjectConfig(projDir, &ProjectConfig{
		Dir:        "specd",
		SpecTypes:  specTypes,
		TaskStages: stages,
	}); err != nil {
		t.Fatal(err)
	}

	specsDir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.MkdirAll(specsDir, 0o755); err != nil { //nolint:gosec // test dir
		t.Fatal(err)
	}
	specPath := filepath.Join(specsDir, "spec.md")
	if err := os.WriteFile(specPath, []byte("---\nid: SPEC-1\ntype: business\nsummary: s\nposition: 0\ncreated_by: t\ncreated_at: 2025-01-01T00:00:00Z\nupdated_at: 2025-01-01T00:00:00Z\n---\n\n# S\n\nb"), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	var err error
	db, err = sql.Open("sqlite", filepath.Join(projDir, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO specs (id, title, type, summary, body, path, content_hash, position, created_at, updated_at)
		VALUES ('SPEC-1', 'S', 'business', 's', 'b', ?, 'h', 0, '2025-01-01', '2025-01-01')`, specPath); err != nil {
		t.Fatal(err)
	}

	taskFiles = []string{}
	for i, id := range []string{"TASK-1", "TASK-2", "TASK-3"} {
		path := filepath.Join(specsDir, id+".md")
		md := buildTaskMarkdown(id, "SPEC-1", "T"+id, "s", "backlog", "t", "2025-01-01T00:00:00Z", i, nil, nil, "body")
		if werr := os.WriteFile(path, []byte(md), 0o644); werr != nil { //nolint:gosec // test
			t.Fatal(werr)
		}
		if _, ierr := db.Exec(`INSERT INTO tasks (id, spec_id, title, status, summary, body, path, position, created_by, content_hash, created_at, updated_at)
			VALUES (?, 'SPEC-1', ?, 'backlog', 's', 'body', ?, ?, 't', 'h', '2025-01-01', '2025-01-01')`,
			id, "T"+id, path, i); ierr != nil {
			t.Fatal(ierr)
		}
		taskFiles = append(taskFiles, path)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, projDir, taskFiles
}

// readPositions returns id→position for a given status, ordered by position.
func readPositions(t *testing.T, db *sql.DB, status string) map[string]int {
	t.Helper()
	rows, err := db.Query("SELECT id, position FROM tasks WHERE status = ? ORDER BY position, id", status)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var pos int
		if err := rows.Scan(&id, &pos); err != nil {
			t.Fatal(err)
		}
		out[id] = pos
	}
	return out
}

func TestMoveTaskWithinColumnReorders(t *testing.T) {
	db, _, _ := setupBoardProject(t)

	// Move TASK-1 (pos 0) to position 2 in the same backlog column.
	moved, err := moveTask(db, "TASK-1", "backlog", 2)
	if err != nil {
		t.Fatalf("moveTask: %v", err)
	}
	if len(moved) == 0 {
		t.Fatal("moved set should not be empty")
	}

	got := readPositions(t, db, "backlog")
	want := map[string]int{"TASK-2": 0, "TASK-3": 1, "TASK-1": 2}
	for id, p := range want {
		if got[id] != p {
			t.Errorf("%s: got pos %d, want %d (got=%v)", id, got[id], p, got)
		}
	}
}

func TestMoveTaskAcrossColumnsRenumbersBoth(t *testing.T) {
	db, _, _ := setupBoardProject(t)

	// Move TASK-2 from backlog (pos 1) to todo (new column, pos 0).
	if _, err := moveTask(db, "TASK-2", "todo", 0); err != nil {
		t.Fatalf("moveTask: %v", err)
	}

	src := readPositions(t, db, "backlog")
	dst := readPositions(t, db, "todo")
	if got := len(src); got != 2 {
		t.Errorf("backlog should retain 2 tasks, got %d (%v)", got, src)
	}
	if src["TASK-1"] != 0 || src["TASK-3"] != 1 {
		t.Errorf("backlog renumber wrong: %v", src)
	}
	if dst["TASK-2"] != 0 {
		t.Errorf("todo: TASK-2 should be at 0, got %d (%v)", dst["TASK-2"], dst)
	}
}

func TestMoveTaskUnknownIDReturnsErrTaskNotFound(t *testing.T) {
	db, _, _ := setupBoardProject(t)
	_, err := moveTask(db, "TASK-999", "todo", 0)
	if !errors.Is(err, errTaskNotFound) {
		t.Errorf("expected errTaskNotFound, got %v", err)
	}
}

func TestMoveTaskOutOfRangePositionClamps(t *testing.T) {
	db, _, _ := setupBoardProject(t)
	// Move TASK-1 to position 99 in todo — should clamp to end of column.
	if _, err := moveTask(db, "TASK-1", "todo", 99); err != nil {
		t.Fatalf("moveTask: %v", err)
	}
	dst := readPositions(t, db, "todo")
	if dst["TASK-1"] != 0 {
		t.Errorf("clamped pos should be 0 (only task in column), got %d", dst["TASK-1"])
	}
}

// -- loadBoard filter behaviour ---------------------------------------------

func TestLoadBoardFilterIncompleteHidesCompletedTasks(t *testing.T) {
	db, _, _ := setupBoardProject(t)

	// Move TASK-3 to done so we have a task in a completed column.
	if _, err := moveTask(db, "TASK-3", "done", 0); err != nil {
		t.Fatal(err)
	}

	stages := []string{"backlog", "todo", "in_progress", "done"}

	all, err := loadBoard(db, stages, BoardFilterAll)
	if err != nil {
		t.Fatal(err)
	}
	inc, err := loadBoard(db, stages, BoardFilterIncomplete)
	if err != nil {
		t.Fatal(err)
	}

	// Both filters keep all configured columns.
	if len(all.Columns) != len(inc.Columns) {
		t.Errorf("incomplete should not hide columns: all=%d inc=%d", len(all.Columns), len(inc.Columns))
	}

	// "all" shows TASK-3 in done; "incomplete" hides it.
	doneTasks := func(b *boardData) []string {
		for _, c := range b.Columns {
			if c.Status == "done" {
				ids := []string{}
				for _, t := range c.Tasks {
					ids = append(ids, t.ID)
				}
				return ids
			}
		}
		return nil
	}
	if got := doneTasks(all); len(got) != 1 || got[0] != "TASK-3" {
		t.Errorf("all filter: done column should have TASK-3, got %v", got)
	}
	if got := doneTasks(inc); len(got) != 0 {
		t.Errorf("incomplete filter: done column should be empty, got %v", got)
	}
}

// -- stripCriteriaSection ---------------------------------------------------

func TestStripCriteriaSectionRemovesCriteriaBlock(t *testing.T) {
	body := strings.Join([]string{
		"Some intro paragraph.",
		"",
		"## Acceptance Criteria",
		"",
		"- [ ] one",
		"- [x] two",
		"",
		"## Notes",
		"",
		"Trailing notes.",
	}, "\n")
	got := stripCriteriaSection(body)
	if strings.Contains(got, "Acceptance Criteria") {
		t.Errorf("criteria heading should be removed, got:\n%s", got)
	}
	if strings.Contains(got, "- [ ] one") || strings.Contains(got, "- [x] two") {
		t.Errorf("criteria items should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Some intro paragraph.") {
		t.Errorf("intro should be kept, got:\n%s", got)
	}
	if !strings.Contains(got, "Trailing notes.") {
		t.Errorf("section after criteria should resume, got:\n%s", got)
	}
}

func TestStripCriteriaSectionStripsDescriptionHeading(t *testing.T) {
	body := strings.Join([]string{
		"## Description",
		"",
		"Body paragraph here.",
	}, "\n")
	got := stripCriteriaSection(body)
	if strings.Contains(got, "## Description") {
		t.Errorf("description heading should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Body paragraph here.") {
		t.Errorf("description body should be preserved, got:\n%s", got)
	}
}

func TestStripCriteriaSectionEmpty(t *testing.T) {
	if got := stripCriteriaSection(""); got != "" {
		t.Errorf("empty input should yield empty, got %q", got)
	}
	if got := stripCriteriaSection("   \n\n\n"); got != "" {
		t.Errorf("whitespace-only should yield empty, got %q", got)
	}
}

// -- searchResultHref --------------------------------------------------------

func TestSearchResultHref(t *testing.T) {
	cases := []struct {
		kind, id, want string
	}{
		{KindSpec, "SPEC-1", "/specs/SPEC-1"},
		{KindTask, "TASK-42", "/tasks/TASK-42"},
		{KindKB, "KB-7", "/kb/KB-7"},
		{"unknown", "X-1", "/search"},
	}
	for _, c := range cases {
		if got := searchResultHref(c.kind, c.id); got != c.want {
			t.Errorf("searchResultHref(%q, %q) = %q, want %q", c.kind, c.id, got, c.want)
		}
	}
}

// -- LoadTaskDetail ---------------------------------------------------------

func TestLoadTaskDetailHappyPath(t *testing.T) {
	db, _, _ := setupBoardProject(t)
	got, err := LoadTaskDetail(db, "TASK-1")
	if err != nil {
		t.Fatalf("LoadTaskDetail: %v", err)
	}
	if got.ID != "TASK-1" {
		t.Errorf("got ID %q, want TASK-1", got.ID)
	}
	if got.SpecID != "SPEC-1" {
		t.Errorf("got SpecID %q, want SPEC-1", got.SpecID)
	}
	if got.Status != "backlog" {
		t.Errorf("got Status %q, want backlog", got.Status)
	}
}

func TestLoadTaskDetailMissingReturnsWrappedNotFound(t *testing.T) {
	db, _, _ := setupBoardProject(t)
	_, err := LoadTaskDetail(db, "TASK-9999")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected wrapped sql.ErrNoRows, got %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got %q", err.Error())
	}
}

// Sanity check that buildTaskMarkdown writes the position field — guards
// against accidental regressions to the hardcoded `position: 0` we replaced.
func TestBuildTaskMarkdownIncludesPosition(t *testing.T) {
	md := buildTaskMarkdown("TASK-1", "SPEC-1", "T", "s", "backlog", "u", "2025-01-01T00:00:00Z", 7, nil, nil, "body")
	if !strings.Contains(md, "position: 7") {
		t.Errorf("buildTaskMarkdown should write 'position: 7', got:\n%s", md)
	}
	if strings.Contains(md, "position: 0") && !strings.Contains(md, "position: 7") {
		t.Error("position should not silently fall back to 0")
	}
}

// Ensure the moveTask renumbering writes back to the task .md file via
// rewriteTaskFile — i.e., the filesystem stays in sync after a move. This
// guards the AGENTS.md "frontmatter ↔ DB ↔ UI sync" rule.
func TestRewriteTaskFileAfterMoveUpdatesFrontmatter(t *testing.T) {
	db, _, _ := setupBoardProject(t)

	if _, err := moveTask(db, "TASK-1", "todo", 0); err != nil {
		t.Fatal(err)
	}
	if err := rewriteTaskFile(db, "TASK-1"); err != nil {
		t.Fatalf("rewriteTaskFile: %v", err)
	}

	var path string
	if err := db.QueryRow("SELECT path FROM tasks WHERE id = 'TASK-1'").Scan(&path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // test
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "status: todo") {
		t.Errorf("file should reflect new status, got:\n%s", body)
	}
	if !strings.Contains(body, "position: 0") {
		t.Errorf("file should reflect new position, got:\n%s", body)
	}
}
