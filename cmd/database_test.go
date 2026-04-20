package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestInitDB verifies that the database is created with the correct schema,
// dynamic CHECK constraints match the provided spec types and task stages,
// and the schema version is recorded in the meta table.
func TestInitDB(t *testing.T) {
	tmp := t.TempDir()

	specTypes := []string{"business", "functional"}
	taskStages := []string{"backlog", "todo", "in_progress", "done"}

	if err := InitDB(tmp, specTypes, taskStages); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	dbPath := filepath.Join(tmp, CacheDBFile)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify schema version.
	var version string
	err = db.QueryRow("SELECT value FROM meta WHERE key = 'schema_version'").Scan(&version)
	if err != nil {
		t.Fatalf("reading schema version: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, version)
	}

	// Verify CHECK constraint accepts valid spec type.
	_, err = db.Exec(`INSERT INTO specs
		(id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'test', 'Test', 'business', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err != nil {
		t.Fatalf("inserting valid spec type: %v", err)
	}

	// Verify CHECK constraint rejects invalid spec type.
	_, err = db.Exec(`INSERT INTO specs
		(id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-2', 'bad', 'Bad', 'invalid_type', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err == nil {
		t.Fatal("expected CHECK constraint to reject invalid spec type")
	}

	// Verify CHECK constraint accepts valid task status.
	_, err = db.Exec(`INSERT INTO tasks
		(id, slug, spec_id, title, status, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('TASK-1', 'test', 'SPEC-1', 'Test', 'backlog', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err != nil {
		t.Fatalf("inserting valid task status: %v", err)
	}

	// Verify CHECK constraint rejects invalid task status.
	_, err = db.Exec(`INSERT INTO tasks
		(id, slug, spec_id, title, status, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('TASK-2', 'bad', 'SPEC-1', 'Bad', 'invalid_status', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err == nil {
		t.Fatal("expected CHECK constraint to reject invalid task status")
	}
}

// TestBuildSchema verifies that the schema template placeholders are correctly
// replaced with quoted, comma-separated values.
func TestBuildSchema(t *testing.T) {
	schema := buildSchema(
		[]string{"business", "functional"},
		[]string{"backlog", "done"},
	)

	if !strings.Contains(schema, "'business','functional'") {
		t.Error("spec types not correctly inserted into schema")
	}
	if !strings.Contains(schema, "'backlog','done'") {
		t.Error("task stages not correctly inserted into schema")
	}
	if !strings.Contains(schema, "DEFAULT 'business'") {
		t.Error("default spec type not set to first type")
	}
}

// TestNextID verifies atomic read-and-increment of meta counters.
func TestNextID(t *testing.T) {
	tmp := t.TempDir()

	specTypes := []string{"business"}
	taskStages := []string{"backlog", "done"}
	if err := InitDB(tmp, specTypes, taskStages); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// First call should return 1.
	id1, err := NextID(db, "next_spec_id")
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id1 != 1 {
		t.Errorf("expected 1, got %d", id1)
	}

	// Second call should return 2.
	id2, err := NextID(db, "next_spec_id")
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id2 != 2 {
		t.Errorf("expected 2, got %d", id2)
	}
}

// TestSearchRelatedSpecsEmpty verifies search returns nil when no matches exist.
func TestSearchRelatedSpecsEmpty(t *testing.T) {
	tmp := t.TempDir()

	if err := InitDB(tmp, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	results, err := SearchRelatedSpecs(db, "nonexistent query", "SPEC-0", 5)
	if err != nil {
		t.Fatalf("SearchRelatedSpecs: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

// TestSanitizeFTSQuery verifies that special characters are stripped and
// words are joined with OR.
func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user authentication", "user OR authentication"},
		{"hello*world", "hello OR world"},
		{`test "quoted"`, "test OR quoted"},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		got := sanitizeFTSQuery(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFTSQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestResolveActiveUsername verifies that project-level username takes
// precedence over global.
func TestResolveActiveUsername(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// No config at all — should return empty.
	if got := ResolveActiveUsername(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// Set global username.
	if err := SaveGlobalConfig(&GlobalConfig{Username: "global-user"}); err != nil {
		t.Fatal(err)
	}
	if got := ResolveActiveUsername(); got != "global-user" {
		t.Errorf("expected %q, got %q", "global-user", got)
	}

	// Set project-level override.
	origDir, _ := os.Getwd()
	projDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := SaveProjectConfig(projDir, &ProjectConfig{
		Folder:   "specd",
		Username: "project-user",
	}); err != nil {
		t.Fatal(err)
	}
	if got := ResolveActiveUsername(); got != "project-user" {
		t.Errorf("expected %q, got %q", "project-user", got)
	}
}
