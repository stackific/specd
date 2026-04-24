package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// setupSyncProject creates an initialized specd project in a temp dir,
// returns the project dir and a function to open its DB.
func setupSyncProject(t *testing.T) (projDir string, openDB func() *sql.DB) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	projDir = filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}

	specTypes := []string{"business", "functional"}
	taskStages := []string{"backlog", "done"}

	// Create project config.
	if err := SaveProjectConfig(projDir, &ProjectConfig{
		Folder:           "specd",
		SpecTypes:        specTypes,
		TaskStages:       taskStages,
		TopSearchResults: 5,
		SearchWeights:    defaultSearchWeights(),
	}); err != nil {
		t.Fatal(err)
	}

	// Create specd folder and specs subfolder.
	specsDir := filepath.Join(projDir, "specd", SpecsSubdir)
	if err := os.MkdirAll(specsDir, 0o755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}

	// Initialize the database at the project root.
	if err := InitDB(projDir, specTypes, taskStages); err != nil {
		t.Fatal(err)
	}

	// Save global config so the guard doesn't block.
	if err := SaveGlobalConfig(&GlobalConfig{Username: "sync-tester"}); err != nil {
		t.Fatal(err)
	}

	openDB = func() *sql.DB {
		db, err := sql.Open("sqlite", filepath.Join(projDir, CacheDBFile))
		if err != nil {
			t.Fatal(err)
		}
		return db
	}

	return projDir, openDB
}

// writeSpecFile writes a spec.md file with the given frontmatter and body.
func writeSpecFile(t *testing.T, projDir, dirName, id, title, specType, summary, body string) {
	t.Helper()
	dir := filepath.Join(projDir, "specd", SpecsSubdir, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}

	md := buildSpecMarkdown(id, title, summary, specType, "sync-tester", "2025-01-01T00:00:00Z", nil, body)
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(md), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
}

// TestSyncInsertNewSpec verifies that a spec.md on disk that isn't in the
// DB gets inserted during sync.
func TestSyncInsertNewSpec(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	// Write a spec file on disk.
	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Authentication", "business",
		"OAuth2 login", "Implement OAuth2 authentication")

	// chdir and run sync.
	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache: %v", err)
	}

	// Verify it was inserted.
	db := openDB()
	defer func() { _ = db.Close() }()

	var title string
	err := db.QueryRow("SELECT title FROM specs WHERE id = 'SPEC-1'").Scan(&title)
	if err != nil {
		t.Fatalf("spec not found in DB after sync: %v", err)
	}
	if title != "User Authentication" {
		t.Errorf("expected title %q, got %q", "User Authentication", title)
	}
}

// TestSyncUpdateChangedSpec verifies that editing a spec.md on disk
// triggers an update in the DB during sync.
func TestSyncUpdateChangedSpec(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Write initial spec and sync.
	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Authentication", "business",
		"OAuth2 login", "Version 1 body")

	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache initial: %v", err)
	}

	// Verify initial body hash.
	db := openDB()
	var hash1 string
	_ = db.QueryRow("SELECT content_hash FROM specs WHERE id = 'SPEC-1'").Scan(&hash1)
	_ = db.Close()

	// Modify the spec body on disk.
	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Authentication", "business",
		"OAuth2 login", "Version 2 body with changes")

	// Sync again.
	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache update: %v", err)
	}

	// Verify the hash changed.
	db = openDB()
	defer func() { _ = db.Close() }()

	var hash2, body string
	_ = db.QueryRow("SELECT content_hash, body FROM specs WHERE id = 'SPEC-1'").Scan(&hash2, &body)

	if hash1 == hash2 {
		t.Error("content_hash should have changed after editing spec on disk")
	}
	if !strings.Contains(body, "Version 2") {
		t.Errorf("expected updated body, got %q", body)
	}
}

// TestSyncDeleteRemovedSpec verifies that deleting a spec.md from disk
// removes it from the DB during sync.
func TestSyncDeleteRemovedSpec(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Write spec and sync.
	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Authentication", "business",
		"OAuth2 login", "Body content")

	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache initial: %v", err)
	}

	// Delete the spec from disk.
	specDir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.RemoveAll(specDir); err != nil {
		t.Fatal(err)
	}

	// Sync again.
	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache after delete: %v", err)
	}

	// Verify it was deleted from DB.
	db := openDB()
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-1'").Scan(&count)
	if count != 0 {
		t.Error("spec should have been deleted from DB after removing from disk")
	}
}

// TestSyncNoChangeSkips verifies that sync doesn't update specs whose
// content hash hasn't changed.
func TestSyncNoChangeSkips(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Authentication", "business",
		"OAuth2 login", "Stable body")

	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	// Read the updated_at after first sync.
	db := openDB()
	var updatedAt1 string
	_ = db.QueryRow("SELECT updated_at FROM specs WHERE id = 'SPEC-1'").Scan(&updatedAt1)
	_ = db.Close()

	// Sync again without changing the file.
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	// updated_at should not change.
	db = openDB()
	defer func() { _ = db.Close() }()

	var updatedAt2 string
	_ = db.QueryRow("SELECT updated_at FROM specs WHERE id = 'SPEC-1'").Scan(&updatedAt2)

	if updatedAt1 != updatedAt2 {
		t.Errorf("updated_at changed without file modification: %q → %q", updatedAt1, updatedAt2)
	}
}

// TestSyncLinkedSpecsCreatesLinks verifies that linked_specs in frontmatter
// creates bidirectional spec_links rows during sync.
func TestSyncLinkedSpecsCreatesLinks(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")

	md := "---\nid: SPEC-2\ntype: functional\nsummary: Tokens\nlinked_specs:\n  - SPEC-1\n---\n\n# Sessions\n\nBody"
	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-2")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(md), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	// Both directions should exist.
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links WHERE from_spec = 'SPEC-2' AND to_spec = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Error("expected link SPEC-2 → SPEC-1")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links WHERE from_spec = 'SPEC-1' AND to_spec = 'SPEC-2'").Scan(&count)
	if count != 1 {
		t.Error("expected reverse link SPEC-1 → SPEC-2")
	}
}

// TestSyncSkipsInvalidSpec verifies that a spec.md with missing required
// fields is silently skipped without affecting other valid specs.
func TestSyncSkipsInvalidSpec(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Valid spec.
	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")

	// Invalid spec — missing summary.
	invalidMD := "---\nid: SPEC-BAD\ntype: business\n---\n\n# Bad Spec\n\nBody"
	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-bad")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(invalidMD), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	// Sync should succeed — invalid spec skipped.
	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache should not fail on invalid spec: %v", err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	// Valid spec should be present.
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Error("valid spec SPEC-1 should be in DB")
	}

	// Invalid spec should NOT be present.
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-BAD'").Scan(&count)
	if count != 0 {
		t.Error("invalid spec SPEC-BAD should have been skipped")
	}
}

// TestParseSpecMarkdown verifies the frontmatter parser.
func TestParseSpecMarkdown(t *testing.T) {
	body := "# User Authentication\n\n## Overview\n\nImplement OAuth2."

	content := fmt.Sprintf(`---
id: SPEC-42
type: functional
summary: OAuth2 login flow
created_by: alice
created_at: 2025-01-01T00:00:00Z
updated_at: 2025-06-15T12:00:00Z
---

%s
`, body)

	ds, err := parseSpecMarkdown(content, "specd/specs/spec-42/spec.md")
	if err != nil {
		t.Fatalf("parseSpecMarkdown: %v", err)
	}

	if ds.ID != "SPEC-42" {
		t.Errorf("ID: got %q, want SPEC-42", ds.ID)
	}
	if ds.Title != "User Authentication" {
		t.Errorf("Title: got %q", ds.Title)
	}
	if ds.Type != "functional" {
		t.Errorf("Type: got %q", ds.Type)
	}
	if ds.Summary != "OAuth2 login flow" {
		t.Errorf("Summary: got %q", ds.Summary)
	}
	if ds.CreatedBy != "alice" {
		t.Errorf("CreatedBy: got %q", ds.CreatedBy)
	}
	// Hash is computed from the full file content (frontmatter + body).
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	if ds.ContentHash != expectedHash {
		t.Errorf("ContentHash: got %q, want %q", ds.ContentHash, expectedHash)
	}
	if ds.Path != "specd/specs/spec-42/spec.md" {
		t.Errorf("Path: got %q", ds.Path)
	}
}

// TestParseSpecMarkdownWithLinkedSpecs verifies linked_specs YAML list parsing.
func TestParseSpecMarkdownWithLinkedSpecs(t *testing.T) {
	content := "---\nid: SPEC-5\ntype: business\nsummary: A feature\nlinked_specs:\n  - SPEC-1\n  - SPEC-3\n---\n\n# Feature X\n\nBody text"

	ds, err := parseSpecMarkdown(content, "test.md")
	if err != nil {
		t.Fatalf("parseSpecMarkdown: %v", err)
	}
	if len(ds.LinkedSpecs) != 2 {
		t.Fatalf("expected 2 linked specs, got %d", len(ds.LinkedSpecs))
	}
	if ds.LinkedSpecs[0] != "SPEC-1" || ds.LinkedSpecs[1] != "SPEC-3" {
		t.Errorf("linked specs: got %v", ds.LinkedSpecs)
	}
}

// TestParseSpecMarkdownMissingFields verifies validation rejects specs
// with missing required fields.
func TestParseSpecMarkdownMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"missing title (no H1)", "---\nid: SPEC-1\ntype: business\nsummary: S\n---\n\nBody without heading"},
		{"missing type", "---\nid: SPEC-1\nsummary: S\n---\n\n# T\n\nBody"},
		{"missing summary", "---\nid: SPEC-1\ntype: business\n---\n\n# T\n\nBody"},
	}
	for _, tt := range tests {
		_, err := parseSpecMarkdown(tt.content, "test.md")
		if err == nil {
			t.Errorf("%s: expected validation error, got nil", tt.name)
		}
	}
}

// TestExtractH1Title verifies title extraction from the first H1 heading.
func TestExtractH1Title(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"# User Authentication\n\n## Overview\n\nDetails.", "User Authentication"},
		{"## No H1 here\n\nBody.", ""},
		{"# First\n\n# Second\n\nBody.", "First"},  // only first H1 counts
		{"#NoSpace\n\n# With Space", "With Space"}, // #NoSpace is not a valid H1
	}
	for _, tt := range tests {
		got := extractH1Title(tt.body)
		if got != tt.want {
			t.Errorf("extractH1Title(%q) = %q, want %q", tt.body[:20], got, tt.want)
		}
	}
}

// TestExtractClaims verifies parsing of acceptance criteria claims.
func TestExtractClaims(t *testing.T) {
	body := `# Title

## Overview

Some overview text.

## Acceptance Criteria

- The system must authenticate users via OAuth2
- The system should support Google and GitHub providers
- Users may choose to stay logged in

## Notes

This section is not part of acceptance criteria.
`
	claims := extractClaims(body)
	if len(claims) != 3 {
		t.Fatalf("expected 3 claims, got %d: %v", len(claims), claims)
	}
	if claims[0] != "The system must authenticate users via OAuth2" {
		t.Errorf("claim 0: got %q", claims[0])
	}
	if claims[2] != "Users may choose to stay logged in" {
		t.Errorf("claim 2: got %q", claims[2])
	}
}

// TestExtractClaimsNoSection verifies no claims when section is absent.
func TestExtractClaimsNoSection(t *testing.T) {
	body := "# Title\n\n## Overview\n\nNo acceptance criteria here."
	claims := extractClaims(body)
	if len(claims) != 0 {
		t.Errorf("expected 0 claims, got %d", len(claims))
	}
}

// TestSyncSpecClaims verifies that claims from ## Acceptance Criteria
// are synced to the spec_claims table and FTS index.
func TestSyncSpecClaims(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Write a spec with acceptance criteria claims.
	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	md := `---
id: SPEC-1
type: business
summary: OAuth2 login
created_by: tester
created_at: 2025-01-01T00:00:00Z
updated_at: 2025-01-01T00:00:00Z
---

# User Authentication

## Overview

Implement OAuth2.

## Acceptance Criteria

- The system must redirect to the OAuth2 consent screen
- The system should create new users on first login
- Users may use remember-me functionality
`
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(md), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache: %v", err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	// Verify claims in DB.
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_claims WHERE spec_id = 'SPEC-1'").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 claims, got %d", count)
	}

	// Verify claim text.
	var text string
	_ = db.QueryRow("SELECT text FROM spec_claims WHERE spec_id = 'SPEC-1' AND position = 1").Scan(&text)
	if text != "The system must redirect to the OAuth2 consent screen" {
		t.Errorf("claim 1 text: got %q", text)
	}

	// Verify FTS index works for claims search.
	var ftsCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM spec_claims_fts WHERE spec_claims_fts MATCH '"redirect" "consent"'`).Scan(&ftsCount)
	if ftsCount == 0 {
		t.Error("expected FTS to find claims matching 'redirect consent'")
	}
}

// TestExtractClaimsWithH3InsideSection verifies that H3-H6 headings inside
// the Acceptance Criteria section do NOT end the section.
func TestExtractClaimsWithH3InsideSection(t *testing.T) {
	body := `# Title

## Acceptance Criteria

### Must-have

- The system must authenticate users
- The system must log all attempts

### Nice-to-have

- Users may customize their dashboard

## Notes

Not part of criteria.
`
	claims := extractClaims(body)
	if len(claims) != 3 {
		t.Fatalf("expected 3 claims (H3 should not break section), got %d: %v", len(claims), claims)
	}
	if claims[2] != "Users may customize their dashboard" {
		t.Errorf("claim 3: got %q", claims[2])
	}
}

// ── Task sync tests ──────────────────────────────────────────────────

// writeTaskFile writes a TASK-*.md file inside a spec directory.
func writeTaskFile(t *testing.T, projDir, specDirName, id, specID, title, status, summary, body string) {
	t.Helper()
	dir := filepath.Join(projDir, "specd", SpecsSubdir, specDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}

	md := buildTaskMarkdown(id, specID, title, summary, status, "sync-tester", "2025-01-01T00:00:00Z", nil, nil, body)
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(md), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
}

// TestSyncInsertNewTask verifies that a task.md on disk gets inserted during sync.
func TestSyncInsertNewTask(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create parent spec first.
	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	// Write task on disk.
	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "Create schema",
		"## Overview\n\nCreate tables.\n\n## Acceptance Criteria\n\n- [ ] Must create users table\n- [ ] Should add indexes")
	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache: %v", err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	var title string
	err := db.QueryRow("SELECT title FROM tasks WHERE id = 'TASK-1'").Scan(&title)
	if err != nil {
		t.Fatalf("task not found in DB after sync: %v", err)
	}
	if title != "Setup Database" {
		t.Errorf("expected title %q, got %q", "Setup Database", title)
	}

	// Verify task criteria were synced.
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM task_criteria WHERE task_id = 'TASK-1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 task criteria, got %d", count)
	}
}

// TestSyncUpdateChangedTask verifies that editing a task.md on disk
// triggers an update in the DB.
func TestSyncUpdateChangedTask(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")
	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "V1 summary", "Version 1 body")
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db := openDB()
	var hash1 string
	_ = db.QueryRow("SELECT content_hash FROM tasks WHERE id = 'TASK-1'").Scan(&hash1)
	_ = db.Close()

	// Modify task on disk.
	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "V2 summary", "Version 2 body updated")
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db = openDB()
	defer func() { _ = db.Close() }()

	var hash2, body string
	_ = db.QueryRow("SELECT content_hash, body FROM tasks WHERE id = 'TASK-1'").Scan(&hash2, &body)
	if hash1 == hash2 {
		t.Error("content_hash should have changed after editing task on disk")
	}
	if !strings.Contains(body, "Version 2") {
		t.Errorf("expected updated body, got %q", body)
	}
}

// TestSyncDeleteRemovedTask verifies that deleting a task.md from disk
// removes it from the DB.
func TestSyncDeleteRemovedTask(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")
	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "Summary", "Body content")
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	// Delete task from disk.
	taskDir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1", "TASK-1.md")
	if err := os.RemoveAll(taskDir); err != nil {
		t.Fatal(err)
	}
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-1'").Scan(&count)
	if count != 0 {
		t.Error("task should have been deleted from DB after removing from disk")
	}
}

// TestSyncTaskLinksCreatesBidirectional verifies that linked_tasks in frontmatter
// creates bidirectional task_links rows during sync.
func TestSyncTaskLinksCreatesBidirectional(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")

	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "Summary 1", "Body 1")

	// Task 2 links to task 1.
	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	md := "---\nid: TASK-2\nspec_id: SPEC-1\nstatus: backlog\nsummary: Create tables\nlinked_tasks:\n  - TASK-1\ncreated_by: sync-tester\ncreated_at: 2025-01-01T00:00:00Z\nupdated_at: 2025-01-01T00:00:00Z\n---\n\n# Create Tables\n\nBody"
	if err := os.WriteFile(filepath.Join(dir, "TASK-2.md"), []byte(md), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM task_links WHERE from_task = 'TASK-2' AND to_task = 'TASK-1'").Scan(&count)
	if count != 1 {
		t.Error("expected link TASK-2 → TASK-1")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM task_links WHERE from_task = 'TASK-1' AND to_task = 'TASK-2'").Scan(&count)
	if count != 1 {
		t.Error("expected reverse link TASK-1 → TASK-2")
	}
}

// TestSyncTaskDependencies verifies that depends_on creates directed
// task_dependencies rows.
func TestSyncTaskDependencies(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")
	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "Summary", "Body")

	// Task 2 depends on (is blocked by) task 1.
	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	md := "---\nid: TASK-2\nspec_id: SPEC-1\nstatus: backlog\nsummary: Run migrations\ndepends_on:\n  - TASK-1\ncreated_by: sync-tester\ncreated_at: 2025-01-01T00:00:00Z\nupdated_at: 2025-01-01T00:00:00Z\n---\n\n# Run Migrations\n\nBody"
	if err := os.WriteFile(filepath.Join(dir, "TASK-2.md"), []byte(md), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM task_dependencies WHERE blocker_task = 'TASK-1' AND blocked_task = 'TASK-2'").Scan(&count)
	if count != 1 {
		t.Error("expected dependency TASK-1 blocks TASK-2")
	}
}

// TestSyncSkipsInvalidTask verifies that a task.md with missing required
// fields is silently skipped.
func TestSyncSkipsInvalidTask(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	writeSpecFile(t, projDir, "spec-1", "SPEC-1", "User Auth", "business",
		"OAuth2", "Body")
	writeTaskFile(t, projDir, "spec-1", "TASK-1", "SPEC-1",
		"Setup Database", "backlog", "Valid", "Body")

	// Invalid task — missing summary.
	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	invalidMD := "---\nid: TASK-BAD\nspec_id: SPEC-1\nstatus: backlog\n---\n\n# Bad Task\n\nBody"
	if err := os.WriteFile(filepath.Join(dir, "TASK-BAD.md"), []byte(invalidMD), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	if err := SyncCache(); err != nil {
		t.Fatalf("SyncCache should not fail on invalid task: %v", err)
	}

	db := openDB()
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-1'").Scan(&count)
	if count != 1 {
		t.Error("valid task TASK-1 should be in DB")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'TASK-BAD'").Scan(&count)
	if count != 0 {
		t.Error("invalid task TASK-BAD should have been skipped")
	}
}

// TestParseTaskMarkdown verifies the task frontmatter parser.
func TestParseTaskMarkdown(t *testing.T) {
	content := `---
id: TASK-42
spec_id: SPEC-1
status: backlog
summary: Create the database schema
created_by: alice
created_at: 2025-01-01T00:00:00Z
updated_at: 2025-06-15T12:00:00Z
---

# Setup Database

## Overview

Create tables.

## Acceptance Criteria

- [ ] Must create users table
- [x] Should add indexes
`
	dt, err := parseTaskMarkdown(content, "specd/specs/spec-1/TASK-42.md")
	if err != nil {
		t.Fatalf("parseTaskMarkdown: %v", err)
	}

	if dt.ID != "TASK-42" {
		t.Errorf("ID: got %q", dt.ID)
	}
	if dt.SpecID != "SPEC-1" {
		t.Errorf("SpecID: got %q", dt.SpecID)
	}
	if dt.Title != "Setup Database" {
		t.Errorf("Title: got %q", dt.Title)
	}
	if dt.Status != "backlog" {
		t.Errorf("Status: got %q", dt.Status)
	}
	if len(dt.Criteria) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(dt.Criteria))
	}
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	if dt.ContentHash != expectedHash {
		t.Errorf("ContentHash mismatch")
	}
}

// TestParseTaskMarkdownWithLists verifies linked_tasks and depends_on parsing.
func TestParseTaskMarkdownWithLists(t *testing.T) {
	content := "---\nid: TASK-5\nspec_id: SPEC-1\nstatus: todo\nsummary: A task\nlinked_tasks:\n  - TASK-1\n  - TASK-3\ndepends_on:\n  - TASK-2\n---\n\n# Feature X\n\nBody"

	dt, err := parseTaskMarkdown(content, "test.md")
	if err != nil {
		t.Fatalf("parseTaskMarkdown: %v", err)
	}
	if len(dt.LinkedTasks) != 2 {
		t.Fatalf("expected 2 linked tasks, got %d", len(dt.LinkedTasks))
	}
	if dt.LinkedTasks[0] != "TASK-1" || dt.LinkedTasks[1] != "TASK-3" {
		t.Errorf("linked tasks: got %v", dt.LinkedTasks)
	}
	if len(dt.DependsOn) != 1 || dt.DependsOn[0] != "TASK-2" {
		t.Errorf("depends_on: got %v", dt.DependsOn)
	}
}

// TestParseTaskMarkdownMissingFields verifies validation rejects tasks
// with missing required fields.
func TestParseTaskMarkdownMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"missing title (no H1)", "---\nid: TASK-1\nspec_id: SPEC-1\nstatus: backlog\nsummary: S\n---\n\nBody without heading"},
		{"missing spec_id", "---\nid: TASK-1\nstatus: backlog\nsummary: S\n---\n\n# T\n\nBody"},
		{"missing status", "---\nid: TASK-1\nspec_id: SPEC-1\nsummary: S\n---\n\n# T\n\nBody"},
		{"missing summary", "---\nid: TASK-1\nspec_id: SPEC-1\nstatus: backlog\n---\n\n# T\n\nBody"},
	}
	for _, tt := range tests {
		_, err := parseTaskMarkdown(tt.content, "test.md")
		if err == nil {
			t.Errorf("%s: expected validation error, got nil", tt.name)
		}
	}
}

// TestParseSpecMarkdownCRLF verifies that spec parsing works with Windows
// line endings (CRLF).
func TestParseSpecMarkdownCRLF(t *testing.T) {
	// Build content with \r\n line endings (Windows style).
	content := "---\r\nid: SPEC-1\r\ntype: business\r\nsummary: OAuth2 login\r\n---\r\n\r\n# User Auth\r\n\r\n## Acceptance Criteria\r\n\r\n- The system must support OAuth2\r\n"

	ds, err := parseSpecMarkdown(content, "test.md")
	if err != nil {
		t.Fatalf("parseSpecMarkdown with CRLF: %v", err)
	}
	if ds.ID != "SPEC-1" {
		t.Errorf("ID: got %q, want SPEC-1", ds.ID)
	}
	if ds.Title != "User Auth" {
		t.Errorf("Title: got %q, want User Auth", ds.Title)
	}
	if len(ds.Claims) != 1 {
		t.Errorf("expected 1 claim, got %d", len(ds.Claims))
	}
	// Hash should be computed from the raw CRLF content.
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	if ds.ContentHash != expectedHash {
		t.Error("content hash should be computed from raw content, not normalized")
	}
}

// TestParseTaskMarkdownCRLF verifies that task parsing works with Windows
// line endings (CRLF).
func TestParseTaskMarkdownCRLF(t *testing.T) {
	content := "---\r\nid: TASK-1\r\nspec_id: SPEC-1\r\nstatus: backlog\r\nsummary: Create schema\r\n---\r\n\r\n# Setup Database\r\n\r\n## Acceptance Criteria\r\n\r\n- [ ] Must create users table\r\n"

	dt, err := parseTaskMarkdown(content, "test.md")
	if err != nil {
		t.Fatalf("parseTaskMarkdown with CRLF: %v", err)
	}
	if dt.ID != "TASK-1" {
		t.Errorf("ID: got %q, want TASK-1", dt.ID)
	}
	if dt.Title != "Setup Database" {
		t.Errorf("Title: got %q, want Setup Database", dt.Title)
	}
	if len(dt.Criteria) != 1 {
		t.Errorf("expected 1 criterion, got %d", len(dt.Criteria))
	}
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	if dt.ContentHash != expectedHash {
		t.Error("content hash should be computed from raw content, not normalized")
	}
}

// TestSyncClaimsUpdateOnBodyChange verifies that editing the acceptance
// criteria on disk replaces the old claims in the DB.
func TestSyncClaimsUpdateOnBodyChange(t *testing.T) {
	projDir, openDB := setupSyncProject(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	dir := filepath.Join(projDir, "specd", SpecsSubdir, "spec-1")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	// Initial version with 2 claims.
	md1 := "---\nid: SPEC-1\ntype: business\nsummary: Login\n---\n\n# Auth\n\n## Acceptance Criteria\n\n- The system must do A\n- The system must do B\n"
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(md1), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db := openDB()
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_claims WHERE spec_id = 'SPEC-1'").Scan(&count)
	_ = db.Close()
	if count != 2 {
		t.Fatalf("expected 2 initial claims, got %d", count)
	}

	// Updated version with 1 different claim.
	md2 := "---\nid: SPEC-1\ntype: business\nsummary: Login\n---\n\n# Auth\n\n## Acceptance Criteria\n\n- The system must do C\n"
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(md2), 0o644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
	if err := SyncCache(); err != nil {
		t.Fatal(err)
	}

	db = openDB()
	defer func() { _ = db.Close() }()

	_ = db.QueryRow("SELECT COUNT(*) FROM spec_claims WHERE spec_id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 claim after update, got %d", count)
	}

	var text string
	_ = db.QueryRow("SELECT text FROM spec_claims WHERE spec_id = 'SPEC-1' AND position = 1").Scan(&text)
	if text != "The system must do C" {
		t.Errorf("expected updated claim text, got %q", text)
	}
}
