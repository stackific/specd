package cmd

import (
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"
	"regexp"
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

// ftsSpecialChars matches FTS5 query operators (* " ( ) : ^ { } -) that must
// be stripped from user input before passing to MATCH to prevent syntax errors.
var ftsSpecialChars = regexp.MustCompile(`[*"():^{}\-]`)

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

// SearchRelatedSpecs finds specs similar to the given text using FTS5 BM25
// ranking, excluding the spec with excludeID. Returns up to limit results.
func SearchRelatedSpecs(db *sql.DB, searchText, excludeID string, limit int) ([]SearchResult, error) {
	query := sanitizeFTSQuery(searchText)
	if query == "" {
		return nil, nil
	}

	// bm25() returns a negative relevance score (lower = more relevant),
	// so default ascending ORDER BY gives best matches first.
	rows, err := db.Query(`
		SELECT id, title, summary
		FROM specs_fts
		WHERE specs_fts MATCH ?
		AND id != ?
		ORDER BY bm25(specs_fts)
		LIMIT ?
	`, query, excludeID, limit)
	if err != nil {
		// No results is not an error — FTS may have no matches.
		return nil, nil //nolint:nilerr // empty result set is expected
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Summary); err != nil {
			return nil, fmt.Errorf("scanning spec result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchRelatedKBChunks finds KB chunks similar to the given text using
// FTS5 BM25 ranking. Returns up to limit results.
func SearchRelatedKBChunks(db *sql.DB, searchText string, limit int) ([]KBChunkResult, error) {
	query := sanitizeFTSQuery(searchText)
	if query == "" {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT c.id, c.doc_id, substr(c.text, 1, 200) AS preview
		FROM kb_chunks c
		JOIN kb_chunks_fts f ON c.id = f.rowid
		WHERE kb_chunks_fts MATCH ?
		ORDER BY bm25(kb_chunks_fts)
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, nil //nolint:nilerr // empty result set is expected
	}
	defer func() { _ = rows.Close() }()

	var results []KBChunkResult
	for rows.Next() {
		var r KBChunkResult
		if err := rows.Scan(&r.ChunkID, &r.DocID, &r.Preview); err != nil {
			return nil, fmt.Errorf("scanning kb chunk result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchResult holds a spec found by FTS search.
type SearchResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// KBChunkResult holds a KB chunk found by FTS search.
type KBChunkResult struct {
	ChunkID int    `json:"chunk_id"`
	DocID   string `json:"doc_id"`
	Preview string `json:"preview"`
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

// sanitizeFTSQuery strips FTS5 special characters and joins words with OR
// for broad matching (e.g. "user authentication" → "user OR authentication").
// Returns "" if no searchable words remain.
func sanitizeFTSQuery(s string) string {
	s = ftsSpecialChars.ReplaceAllString(s, " ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	return strings.Join(words, " OR ")
}
