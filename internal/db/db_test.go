package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, ".specd", "cache.db")

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}
}

func TestMetaSeedValues(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	tests := map[string]string{
		"schema_version": SchemaVersion,
		"next_spec_id":   "1",
		"next_task_id":   "1",
		"next_kb_id":     "1",
		"user_name":      "",
	}

	for key, want := range tests {
		got, err := d.GetMeta(key)
		if err != nil {
			t.Errorf("GetMeta(%s): %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("GetMeta(%s) = %q, want %q", key, got, want)
		}
	}
}

func TestSetMeta(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if err := d.SetMeta("user_name", "alice"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}

	got, err := d.GetMeta("user_name")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if got != "alice" {
		t.Errorf("got %q, want %q", got, "alice")
	}
}

func TestNextID(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	id1, err := d.NextID("spec")
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id1 != 1 {
		t.Errorf("first spec id = %d, want 1", id1)
	}

	id2, err := d.NextID("spec")
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id2 != 2 {
		t.Errorf("second spec id = %d, want 2", id2)
	}
}

func TestIdempotentOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cache.db")

	// Open, write, close.
	d1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := d1.SetMeta("user_name", "bob"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	d1.Close()

	// Reopen — schema should not be reapplied, data preserved.
	d2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer d2.Close()

	got, err := d2.GetMeta("user_name")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if got != "bob" {
		t.Errorf("got %q, want %q", got, "bob")
	}
}

func TestTablesExist(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	tables := []string{
		"meta", "specs", "tasks", "task_criteria",
		"spec_links", "task_links", "task_dependencies",
		"kb_docs", "kb_chunks", "citations", "chunk_connections",
		"trash", "rejected_files",
	}

	for _, table := range tables {
		var count int
		err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("check table %s: %v", table, err)
			continue
		}
		if count != 1 {
			t.Errorf("table %s not found", table)
		}
	}
}

func TestFTSTablesExist(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// FTS5 tables show as type='table' in sqlite_master.
	fts := []string{"specs_fts", "tasks_fts", "kb_chunks_fts", "search_trigram"}

	for _, table := range fts {
		var count int
		err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("check FTS %s: %v", table, err)
			continue
		}
		if count != 1 {
			t.Errorf("FTS table %s not found", table)
		}
	}
}
