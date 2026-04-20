package cmd

import (
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	// Pure Go SQLite driver — no CGO required.
	_ "modernc.org/sqlite"
)

// schemaSQL is the embedded SQL schema template. Placeholders are replaced
// at runtime with the user's selected values.
//
//go:embed schema.sql
var schemaSQL string

const (
	// CacheDBFile is the SQLite database filename inside the specd project folder.
	CacheDBFile = "cache.db"
	// SchemaVersion is stored in the meta table for future migrations.
	SchemaVersion = "1"
)

// InitDB creates and initializes the cache.db SQLite database inside the
// specd project folder. CHECK constraints and defaults are built from
// the user's selected spec types and task stages.
func InitDB(specdPath string, specTypes, taskStages []string) error {
	dbPath := filepath.Join(specdPath, CacheDBFile)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("setting WAL mode: %w", err)
	}

	// Enable foreign key enforcement.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("enabling foreign keys: %w", err)
	}

	// Build the schema with dynamic CHECK constraints and defaults.
	schema := buildSchema(specTypes, taskStages)

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("executing schema: %w", err)
	}

	// Seed the meta table with schema version and ID counters.
	for _, kv := range []struct{ k, v string }{
		{"schema_version", SchemaVersion},
		{"next_spec_id", "1"},
		{"next_task_id", "1"},
		{"next_kb_id", "1"},
	} {
		if _, err := db.Exec(
			`INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)`,
			kv.k, kv.v,
		); err != nil {
			return fmt.Errorf("seeding meta %s: %w", kv.k, err)
		}
	}

	return nil
}

// OpenProjectDB opens the cache.db for the current project.
// Returns the database handle and the specd folder path.
func OpenProjectDB() (*sql.DB, string, error) {
	proj, err := LoadProjectConfig(".")
	if err != nil {
		return nil, "", fmt.Errorf("reading project config: %w", err)
	}
	if proj == nil {
		return nil, "", fmt.Errorf("specd is not initialized in this directory.\nRun: specd init")
	}

	dbPath := filepath.Join(proj.Folder, CacheDBFile)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("opening database: %w", err)
	}

	// Enable foreign keys on every connection.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, "", fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, proj.Folder, nil
}

// NextID atomically reads and increments a counter in the meta table.
// key should be "next_spec_id", "next_task_id", etc.
// Returns the current value before increment (e.g. 1, 2, 3...).
func NextID(db *sql.DB, key string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// The meta table stores all values as TEXT; CAST extracts the integer counter.
	var id int
	err = tx.QueryRow("SELECT CAST(value AS INTEGER) FROM meta WHERE key = ?", key).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", key, err)
	}

	_, err = tx.Exec("UPDATE meta SET value = CAST(? AS TEXT) WHERE key = ?", id+1, key)
	if err != nil {
		return 0, fmt.Errorf("incrementing %s: %w", key, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing %s: %w", key, err)
	}

	return id, nil
}

// ResolveActiveUsername returns the effective username for the current project.
// Project-level override takes precedence over the global config.
func ResolveActiveUsername() string {
	proj, err := LoadProjectConfig(".")
	if err == nil && proj != nil && proj.Username != "" {
		return proj.Username
	}
	cfg, err := LoadGlobalConfig()
	if err == nil {
		return cfg.Username
	}
	return ""
}

// buildSchema replaces the placeholder tokens in the embedded SQL with
// the actual values derived from the user's selections.
func buildSchema(specTypes, taskStages []string) string {
	schema := schemaSQL
	schema = strings.ReplaceAll(schema, "{{SPEC_TYPES_CHECK}}", quoteList(specTypes))
	schema = strings.ReplaceAll(schema, "{{TASK_STAGES_CHECK}}", quoteList(taskStages))

	// First spec type is the default for new specs.
	defaultType := ""
	if len(specTypes) > 0 {
		defaultType = specTypes[0]
	}
	schema = strings.ReplaceAll(schema, "{{DEFAULT_SPEC_TYPE}}", defaultType)

	return schema
}

// quoteList turns ["backlog", "todo"] into "'backlog','todo'" for use
// inside a SQL IN (...) clause.
func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		escaped := strings.ReplaceAll(v, "'", "''")
		quoted[i] = "'" + escaped + "'"
	}
	return strings.Join(quoted, ",")
}
