package db

import (
	"path/filepath"
	"testing"
)

func TestSpecFTSTriggers(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Insert a spec.
	_, err = d.Exec(`INSERT INTO specs (id, slug, title, type, summary, body, path, position, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'oauth-github', 'OAuth with GitHub', 'technical', 'OAuth flow using GitHub', 'Full body text about authentication.', 'specd/specs/SPEC-1-oauth-github/spec.md', 0, 'abc123', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert spec: %v", err)
	}

	// BM25 search should find it.
	var count int
	err = d.QueryRow("SELECT count(*) FROM specs_fts WHERE specs_fts MATCH 'OAuth'").Scan(&count)
	if err != nil {
		t.Fatalf("FTS query: %v", err)
	}
	if count != 1 {
		t.Errorf("FTS count = %d, want 1", count)
	}

	// Trigram search should also find it.
	err = d.QueryRow("SELECT count(*) FROM search_trigram WHERE kind = 'spec' AND text MATCH 'auth'").Scan(&count)
	if err != nil {
		t.Fatalf("trigram query: %v", err)
	}
	if count != 1 {
		t.Errorf("trigram count = %d, want 1", count)
	}

	// Update the spec.
	_, err = d.Exec(`UPDATE specs SET title = 'OAuth with Google', summary = 'OAuth flow using Google' WHERE id = 'SPEC-1'`)
	if err != nil {
		t.Fatalf("update spec: %v", err)
	}

	// FTS should reflect update.
	err = d.QueryRow("SELECT count(*) FROM specs_fts WHERE specs_fts MATCH 'Google'").Scan(&count)
	if err != nil {
		t.Fatalf("FTS after update: %v", err)
	}
	if count != 1 {
		t.Errorf("FTS after update count = %d, want 1", count)
	}

	// Old term should be gone from FTS.
	err = d.QueryRow("SELECT count(*) FROM specs_fts WHERE specs_fts MATCH 'GitHub'").Scan(&count)
	if err != nil {
		t.Fatalf("FTS old term: %v", err)
	}
	if count != 0 {
		t.Errorf("FTS old term count = %d, want 0", count)
	}

	// Delete the spec.
	_, err = d.Exec("DELETE FROM specs WHERE id = 'SPEC-1'")
	if err != nil {
		t.Fatalf("delete spec: %v", err)
	}

	err = d.QueryRow("SELECT count(*) FROM specs_fts WHERE specs_fts MATCH 'Google'").Scan(&count)
	if err != nil {
		t.Fatalf("FTS after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("FTS after delete count = %d, want 0", count)
	}
}

func TestTaskFTSTriggers(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Need a parent spec first.
	_, err = d.Exec(`INSERT INTO specs (id, slug, title, type, summary, body, path, position, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'auth', 'Auth', 'technical', 'Auth spec', 'Body', 'specd/specs/SPEC-1-auth/spec.md', 0, 'abc', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert spec: %v", err)
	}

	_, err = d.Exec(`INSERT INTO tasks (id, slug, spec_id, title, status, summary, body, path, position, content_hash, created_at, updated_at)
		VALUES ('TASK-1', 'jwt-impl', 'SPEC-1', 'Implement JWT', 'todo', 'Add JWT token generation', 'Full body about JWT implementation.', 'specd/specs/SPEC-1-auth/TASK-1-jwt-impl.md', 0, 'def', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	var count int
	err = d.QueryRow("SELECT count(*) FROM tasks_fts WHERE tasks_fts MATCH 'JWT'").Scan(&count)
	if err != nil {
		t.Fatalf("FTS query: %v", err)
	}
	if count != 1 {
		t.Errorf("task FTS count = %d, want 1", count)
	}
}

func TestTrashInsert(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	_, err = d.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
		VALUES ('spec', 'SPEC-1', 'specd/specs/SPEC-1-test/spec.md', X'48656C6C6F', '{"id":"SPEC-1"}', '2025-01-01T00:00:00Z', 'cli')`)
	if err != nil {
		t.Fatalf("insert trash: %v", err)
	}

	var count int
	d.QueryRow("SELECT count(*) FROM trash").Scan(&count)
	if count != 1 {
		t.Errorf("trash count = %d, want 1", count)
	}
}
