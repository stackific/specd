package cmd

import (
	"database/sql"
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

	// Open the created database and verify.
	dbPath := filepath.Join(tmp, CacheDBFile)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer func() { _ = db.Close() }() // errcheck: Close() returns an error; discard it since we can't act on it during teardown

	// Verify schema version in meta table.
	var version string
	err = db.QueryRow("SELECT value FROM meta WHERE key = 'schema_version'").Scan(&version)
	if err != nil {
		t.Fatalf("reading schema version: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("expected schema version %q, got %q", SchemaVersion, version)
	}

	// Verify CHECK constraint on specs.type accepts valid values.
	_, err = db.Exec(`INSERT INTO specs
		(id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'test', 'Test', 'business', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err != nil {
		t.Fatalf("inserting valid spec type: %v", err)
	}

	// Verify CHECK constraint on specs.type rejects invalid values.
	_, err = db.Exec(`INSERT INTO specs
		(id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-2', 'bad', 'Bad', 'invalid_type', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err == nil {
		t.Fatal("expected CHECK constraint to reject invalid spec type")
	}

	// Verify CHECK constraint on tasks.status accepts valid values.
	_, err = db.Exec(`INSERT INTO tasks
		(id, slug, spec_id, title, status, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('TASK-1', 'test', 'SPEC-1', 'Test', 'backlog', 's', 'b', 'p', 'h', '2025-01-01', '2025-01-01')`)
	if err != nil {
		t.Fatalf("inserting valid task status: %v", err)
	}

	// Verify CHECK constraint on tasks.status rejects invalid values.
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
}
