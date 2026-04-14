// Package db manages the specd SQLite cache database. It handles schema
// creation, migrations, meta key-value storage, and atomic ID counters.
// The schema includes FTS5 full-text indexes and trigram indexes with
// triggers that keep them in sync with base tables automatically.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

const SchemaVersion = "1"

// DB wraps a SQLite connection for specd.
type DB struct {
	*sql.DB
	path string
}

// Open opens (or creates) the specd SQLite database at the given path.
// It enables WAL mode, foreign keys, and applies the schema if needed.
func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Single connection to avoid locking issues with SQLite.
	sqlDB.SetMaxOpenConns(1)

	db := &DB{DB: sqlDB, path: dbPath}

	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate applies the schema and seeds meta rows if needed.
func (db *DB) migrate() error {
	// Check if meta table exists (proxy for "schema applied").
	var exists int
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='meta'").Scan(&exists)
	if err != nil {
		return err
	}

	if exists == 0 {
		if _, err := db.Exec(schemaSQL); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
		if err := db.seed(); err != nil {
			return fmt.Errorf("seed meta: %w", err)
		}
	}

	return nil
}

// seed inserts initial meta rows.
func (db *DB) seed() error {
	now := time.Now().UTC().Format(time.RFC3339)
	seeds := map[string]string{
		"schema_version": SchemaVersion,
		"next_spec_id":   "1",
		"next_task_id":   "1",
		"next_kb_id":     "1",
		"last_tidy_at":   now,
		"user_name":      "",
	}
	for k, v := range seeds {
		if _, err := db.Exec("INSERT INTO meta (key, value) VALUES (?, ?)", k, v); err != nil {
			return fmt.Errorf("seed %s: %w", k, err)
		}
	}
	return nil
}

// GetMeta reads a value from the meta table.
func (db *DB) GetMeta(key string) (string, error) {
	var val string
	err := db.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", fmt.Errorf("get meta %s: %w", key, err)
	}
	return val, nil
}

// SetMeta sets a value in the meta table.
func (db *DB) SetMeta(key, value string) error {
	_, err := db.Exec("UPDATE meta SET value = ? WHERE key = ?", value, key)
	if err != nil {
		return fmt.Errorf("set meta %s: %w", key, err)
	}
	return nil
}

// NextID atomically reads and increments the counter for the given kind.
// kind must be one of "spec", "task", "kb".
func (db *DB) NextID(kind string) (int, error) {
	key := "next_" + kind + "_id"

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var val string
	if err := tx.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&val); err != nil {
		return 0, fmt.Errorf("read counter %s: %w", key, err)
	}

	var id int
	if _, err := fmt.Sscanf(val, "%d", &id); err != nil {
		return 0, fmt.Errorf("parse counter %s: %w", key, err)
	}

	next := id + 1
	if _, err := tx.Exec("UPDATE meta SET value = ? WHERE key = ?", fmt.Sprintf("%d", next), key); err != nil {
		return 0, fmt.Errorf("increment counter %s: %w", key, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}
