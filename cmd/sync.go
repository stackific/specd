// sync.go implements the cache refresher that keeps the .specd.cache database
// in sync with the spec markdown files on disk. The markdown files in the
// specd folder are always the ground truth — the database is a derived cache.
//
// Before every non-exempt command, SyncCache walks the specs directory, parses
// each spec.md frontmatter, computes a content hash of the full file, and
// compares it against the database. New specs are inserted, changed specs are
// updated, and specs deleted from disk are removed from the database. FTS
// indexes and trigram entries are kept in sync automatically via triggers.
package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SyncCache reads spec markdown files from disk and reconciles the database.
// It inserts new specs, updates changed specs (by content hash), deletes
// specs that no longer exist on disk, and reconciles spec_links.
func SyncCache() error {
	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		return nil // not initialized — nothing to sync
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		return fmt.Errorf("opening database for sync: %w", err)
	}
	defer func() { _ = db.Close() }()

	specsDir := filepath.Join(proj.Folder, SpecsSubdir)

	// Collect all spec.md files from disk.
	diskSpecs, err := readSpecsFromDisk(specsDir)
	if err != nil {
		return fmt.Errorf("reading specs from disk: %w", err)
	}

	// Load all spec IDs and hashes from the database.
	dbSpecs, err := loadSpecHashesFromDB(db)
	if err != nil {
		return fmt.Errorf("loading spec hashes from db: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()

	if err := reconcileSpecs(db, diskSpecs, dbSpecs, username, now); err != nil {
		return err
	}

	return nil
}

// reconcileSpecs inserts new, updates changed, and deletes removed specs.
func reconcileSpecs(db *sql.DB, diskSpecs map[string]diskSpec, dbSpecs map[string]string, username, now string) error {
	for id := range diskSpecs {
		ds := diskSpecs[id]
		dbHash, exists := dbSpecs[id]

		switch {
		case !exists:
			slog.Info("sync: inserting new spec", "id", id, "path", ds.Path)
			if err := insertSpecFromDisk(db, &ds, username, now); err != nil {
				return fmt.Errorf("inserting spec %s: %w", id, err)
			}
		case dbHash != ds.ContentHash:
			slog.Info("sync: updating changed spec", "id", id, "path", ds.Path)
			if err := updateSpecFromDisk(db, &ds, username, now); err != nil {
				return fmt.Errorf("updating spec %s: %w", id, err)
			}
		default:
			delete(dbSpecs, id)
			continue // unchanged — skip link sync
		}

		if err := syncSpecLinks(db, ds.ID, ds.LinkedSpecs); err != nil {
			return fmt.Errorf("syncing links for %s: %w", id, err)
		}
		if err := syncSpecClaims(db, ds.ID, ds.Claims); err != nil {
			return fmt.Errorf("syncing claims for %s: %w", id, err)
		}
		delete(dbSpecs, id)
	}

	// Remaining dbSpecs entries were not found on disk — delete them.
	for id := range dbSpecs {
		slog.Info("sync: deleting removed spec", "id", id)
		if err := deleteSpecFromDB(db, id); err != nil {
			return fmt.Errorf("deleting spec %s: %w", id, err)
		}
	}

	return nil
}

// diskSpec holds a spec parsed from a markdown file on disk.
type diskSpec struct {
	ID          string
	Slug        string
	Title       string // extracted from H1 heading, not frontmatter
	Type        string
	Summary     string
	Position    int
	LinkedSpecs []string // spec IDs from the linked_specs frontmatter field
	Claims      []string // acceptance criteria bullet items from ## Acceptance Criteria
	Body        string
	Path        string // relative to project root
	ContentHash string // SHA-256 of the full file (frontmatter + body)
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   string
	UpdatedAt   string
}

// readSpecsFromDisk walks the specs directory and parses each spec.md file.
// Returns a map keyed by spec ID.
func readSpecsFromDisk(specsDir string) (map[string]diskSpec, error) {
	result := make(map[string]diskSpec)

	// specsDir may not exist yet (no specs created).
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return result, nil
	}

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, fmt.Errorf("reading specs directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		specFile := filepath.Join(specsDir, entry.Name(), "spec.md")
		data, err := os.ReadFile(specFile) //nolint:gosec // walking a controlled directory
		if err != nil {
			continue // skip directories without spec.md
		}

		ds, err := parseSpecMarkdown(string(data), specFile)
		if err != nil {
			continue // skip unparseable files
		}

		result[ds.ID] = ds
	}

	return result, nil
}

// parseSpecMarkdown extracts frontmatter fields and body from a spec.md file.
// The content hash is computed from the full file content (frontmatter + body)
// so that any edit — metadata or body — triggers a sync update.
func parseSpecMarkdown(content, path string) (diskSpec, error) {
	// Split frontmatter from body.
	if !strings.HasPrefix(content, "---\n") {
		return diskSpec{}, fmt.Errorf("missing frontmatter delimiter")
	}

	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) != 2 {
		return diskSpec{}, fmt.Errorf("malformed frontmatter")
	}

	frontmatter := parts[0]
	body := strings.TrimSpace(parts[1])

	ds := diskSpec{
		Path: path,
		Body: body,
		// Hash the full file so frontmatter changes are also detected.
		ContentHash: fmt.Sprintf("%x", sha256.Sum256([]byte(content))),
	}

	parseFrontmatter(&ds, frontmatter)

	// Extract the title from the first H1 heading in the body.
	ds.Title = extractH1Title(body)

	// Extract acceptance criteria claims from ## Acceptance Criteria.
	ds.Claims = extractClaims(body)

	// Validate required fields.
	if err := validateSpecFields(&ds, path); err != nil {
		return diskSpec{}, err
	}

	return ds, nil
}

// extractH1Title finds the first `# Title` line in the body and returns
// the title text. Returns "" if no H1 is found.
func extractH1Title(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// extractClaims parses bullet items from the ## Acceptance Criteria section.
// Each `- text` line under that heading is a claim.
func extractClaims(body string) []string {
	var claims []string
	inSection := false

	for _, line := range strings.Split(body, "\n") {
		// Check for ## Acceptance Criteria heading (case-insensitive).
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			inSection = strings.EqualFold(heading, "Acceptance Criteria")
			continue
		}

		if !inSection {
			continue
		}

		// A new H1 or H2 ends the section. H3-H6 are allowed within.
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "### ") {
			break
		}

		// Parse bullet items.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			claim := strings.TrimPrefix(trimmed, "- ")
			if claim != "" {
				claims = append(claims, claim)
			}
		}
	}

	return claims
}

// parseFrontmatter extracts key-value pairs and YAML lists from frontmatter text.
func parseFrontmatter(ds *diskSpec, frontmatter string) {
	// Map scalar fields to their destinations for simple assignment.
	// Title is NOT in frontmatter — it comes from the H1 heading in the body.
	fields := map[string]*string{
		"id":         &ds.ID,
		"slug":       &ds.Slug,
		"type":       &ds.Type,
		"summary":    &ds.Summary,
		"created_by": &ds.CreatedBy,
		"updated_by": &ds.UpdatedBy,
		"created_at": &ds.CreatedAt,
		"updated_at": &ds.UpdatedAt,
	}

	inLinkedSpecs := false
	for _, line := range strings.Split(frontmatter, "\n") {
		// Collect YAML list items under linked_specs.
		if inLinkedSpecs {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- ") {
				ds.LinkedSpecs = append(ds.LinkedSpecs, strings.TrimPrefix(trimmed, "- "))
				continue
			}
			inLinkedSpecs = false
		}

		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		if dest, ok := fields[key]; ok {
			*dest = val
		} else if key == "position" {
			ds.Position, _ = strconv.Atoi(val)
		} else if key == "linked_specs" {
			inLinkedSpecs = true
		}
	}
}

// validateSpecFields checks that all required frontmatter fields are present.
func validateSpecFields(ds *diskSpec, path string) error {
	for _, check := range []struct{ val, name string }{
		{ds.ID, "id"},
		{ds.Slug, "slug"},
		{ds.Title, "title"},
		{ds.Type, "type"},
		{ds.Summary, "summary"},
	} {
		if check.val == "" {
			return fmt.Errorf("%s: missing required field '%s'", path, check.name)
		}
	}
	return nil
}

// loadSpecHashesFromDB returns a map of spec ID → content_hash for all specs.
func loadSpecHashesFromDB(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT id, content_hash FROM specs")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, fmt.Errorf("scanning spec hash: %w", err)
		}
		result[id] = hash
	}
	return result, rows.Err()
}

// insertSpecFromDisk inserts a new spec parsed from disk into the database.
func insertSpecFromDisk(db *sql.DB, ds *diskSpec, username, now string) error {
	createdBy := ds.CreatedBy
	if createdBy == "" {
		createdBy = username
	}
	createdAt := ds.CreatedAt
	if createdAt == "" {
		createdAt = now
	}

	_, err := db.Exec(`
		INSERT OR REPLACE INTO specs (id, slug, title, type, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ds.ID, ds.Slug, ds.Title, ds.Type, ds.Summary, ds.Body, ds.Path, ds.Position, createdBy, ds.UpdatedBy, ds.ContentHash, createdAt, now)
	return err
}

// updateSpecFromDisk updates an existing spec in the database from disk content.
func updateSpecFromDisk(db *sql.DB, ds *diskSpec, username, now string) error {
	updatedBy := ds.UpdatedBy
	if updatedBy == "" {
		updatedBy = username
	}

	_, err := db.Exec(`
		UPDATE specs SET slug = ?, title = ?, type = ?, summary = ?, body = ?, path = ?,
		position = ?, updated_by = ?, content_hash = ?, updated_at = ?
		WHERE id = ?
	`, ds.Slug, ds.Title, ds.Type, ds.Summary, ds.Body, ds.Path, ds.Position, updatedBy, ds.ContentHash, now, ds.ID)
	return err
}

// syncSpecLinks reconciles the spec_links table for a given spec ID.
// Deletes existing links from this spec, then inserts bidirectional links
// for each ID in the linked list.
func syncSpecLinks(db *sql.DB, specID string, linkedSpecs []string) error {
	// Remove all existing outbound links from this spec.
	if _, err := db.Exec("DELETE FROM spec_links WHERE from_spec = ?", specID); err != nil {
		return fmt.Errorf("clearing links: %w", err)
	}

	// Insert bidirectional links.
	for _, toSpec := range linkedSpecs {
		toSpec = strings.TrimSpace(toSpec)
		if toSpec == "" || toSpec == specID {
			continue
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)", specID, toSpec); err != nil {
			return fmt.Errorf("inserting link %s→%s: %w", specID, toSpec, err)
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)", toSpec, specID); err != nil {
			return fmt.Errorf("inserting link %s→%s: %w", toSpec, specID, err)
		}
	}

	return nil
}

// syncSpecClaims reconciles the spec_claims table for a given spec ID.
// Deletes existing claims and re-inserts from the parsed list.
// FTS triggers on spec_claims keep the spec_claims_fts index in sync.
func syncSpecClaims(db *sql.DB, specID string, claims []string) error {
	// Remove all existing claims for this spec.
	if _, err := db.Exec("DELETE FROM spec_claims WHERE spec_id = ?", specID); err != nil {
		return fmt.Errorf("clearing claims: %w", err)
	}

	// Insert new claims with 1-based position.
	for i, claim := range claims {
		if _, err := db.Exec(
			"INSERT INTO spec_claims (spec_id, position, text) VALUES (?, ?, ?)",
			specID, i+1, claim,
		); err != nil {
			return fmt.Errorf("inserting claim %d: %w", i+1, err)
		}
	}

	return nil
}

// deleteSpecFromDB removes a spec from the database. ON DELETE CASCADE
// handles spec_links and spec_claims cleanup. FTS triggers clean up indexes.
func deleteSpecFromDB(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM specs WHERE id = ?", id)
	return err
}
