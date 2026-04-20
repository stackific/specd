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

// schemaSQL is the embedded SQL schema template. The placeholders
// {{SPEC_TYPES_CHECK}} and {{TASK_STAGES_CHECK}} are replaced at runtime
// with the user's selected spec types and task stages.
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
// specd project folder. CHECK constraints on specs.type and tasks.status
// are built from the user's selected spec types and task stages.
func InitDB(specdPath string, specTypes, taskStages []string) error {
	dbPath := filepath.Join(specdPath, CacheDBFile)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }() // errcheck: Close() returns an error; discard it since we can't act on it during teardown

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("setting WAL mode: %w", err)
	}

	// Enable foreign key enforcement.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("enabling foreign keys: %w", err)
	}

	// Build the schema with dynamic CHECK constraints from user selections.
	schema := buildSchema(specTypes, taskStages)

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("executing schema: %w", err)
	}

	// Record schema version in meta table.
	_, err = db.Exec(
		`INSERT OR REPLACE INTO meta (key, value) VALUES ('schema_version', ?)`,
		SchemaVersion,
	)
	if err != nil {
		return fmt.Errorf("writing schema version: %w", err)
	}

	return nil
}

// buildSchema replaces the placeholder tokens in the embedded SQL with
// the actual CHECK constraint values derived from the user's selections.
func buildSchema(specTypes, taskStages []string) string {
	schema := schemaSQL
	schema = strings.ReplaceAll(schema, "{{SPEC_TYPES_CHECK}}", quoteList(specTypes))
	schema = strings.ReplaceAll(schema, "{{TASK_STAGES_CHECK}}", quoteList(taskStages))
	return schema
}

// quoteList turns ["backlog", "todo"] into "'backlog','todo'" for use
// inside a SQL IN (...) clause.
func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		// Escape single quotes within values to prevent SQL injection.
		escaped := strings.ReplaceAll(v, "'", "''")
		quoted[i] = "'" + escaped + "'"
	}
	return strings.Join(quoted, ",")
}
