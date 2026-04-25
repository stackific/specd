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

// normalizeLineEndings converts Windows CRLF line endings to Unix LF.
// This ensures markdown parsing works regardless of the OS that wrote the file.
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

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

	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()

	// Sync specs.
	specsDir := filepath.Join(proj.Dir, SpecsSubdir)
	diskSpecs, err := readSpecsFromDisk(specsDir)
	if err != nil {
		return fmt.Errorf("reading specs from disk: %w", err)
	}
	dbSpecs, err := loadSpecHashesFromDB(db)
	if err != nil {
		return fmt.Errorf("loading spec hashes from db: %w", err)
	}
	if err := reconcileSpecs(db, diskSpecs, dbSpecs, username, now); err != nil {
		return err
	}

	// Sync tasks (task files live inside spec directories as TASK-*.md).
	diskTasks, err := readTasksFromDisk(specsDir)
	if err != nil {
		return fmt.Errorf("reading tasks from disk: %w", err)
	}
	dbTasks, err := loadTaskHashesFromDB(db)
	if err != nil {
		return fmt.Errorf("loading task hashes from db: %w", err)
	}
	if err := reconcileTasks(db, diskTasks, dbTasks, username, now); err != nil {
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
	// Hash the raw file content before normalizing line endings,
	// so the hash matches what's actually on disk regardless of OS.
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	// Normalize CRLF → LF so parsing works on Windows.
	content = normalizeLineEndings(content)

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
		Path:        path,
		Body:        body,
		ContentHash: contentHash,
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
		INSERT OR REPLACE INTO specs (id, title, type, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ds.ID, ds.Title, ds.Type, ds.Summary, ds.Body, ds.Path, ds.Position, createdBy, ds.UpdatedBy, ds.ContentHash, createdAt, now)
	return err
}

// updateSpecFromDisk updates an existing spec in the database from disk content.
func updateSpecFromDisk(db *sql.DB, ds *diskSpec, username, now string) error {
	updatedBy := ds.UpdatedBy
	if updatedBy == "" {
		updatedBy = username
	}

	_, err := db.Exec(`
		UPDATE specs SET title = ?, type = ?, summary = ?, body = ?, path = ?,
		position = ?, updated_by = ?, content_hash = ?, updated_at = ?
		WHERE id = ?
	`, ds.Title, ds.Type, ds.Summary, ds.Body, ds.Path, ds.Position, updatedBy, ds.ContentHash, now, ds.ID)
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

// ── Task sync ──────────────────────────────────────────────────────────

// diskTask holds a task parsed from a markdown file on disk.
type diskTask struct {
	ID          string
	SpecID      string
	Title       string // extracted from H1 heading, not frontmatter
	Status      string
	Summary     string
	Position    int
	LinkedTasks []string // task IDs from the linked_tasks frontmatter field
	DependsOn   []string // task IDs from the depends_on frontmatter field
	Criteria    []string // acceptance criteria checkbox items
	Body        string
	Path        string
	ContentHash string
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   string
	UpdatedAt   string
}

// readTasksFromDisk walks all spec directories looking for TASK-*.md files.
// Returns a map keyed by task ID.
func readTasksFromDisk(specsDir string) (map[string]diskTask, error) {
	result := make(map[string]diskTask)

	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return result, nil
	}

	specDirs, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, fmt.Errorf("reading specs directory for tasks: %w", err)
	}

	for _, specEntry := range specDirs {
		if !specEntry.IsDir() {
			continue
		}

		specDir := filepath.Join(specsDir, specEntry.Name())
		files, err := os.ReadDir(specDir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasPrefix(f.Name(), IDPrefixTask) || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}

			taskFile := filepath.Join(specDir, f.Name())
			data, err := os.ReadFile(taskFile) //nolint:gosec // walking a controlled directory
			if err != nil {
				continue
			}

			dt, err := parseTaskMarkdown(string(data), taskFile)
			if err != nil {
				continue // skip unparseable files
			}

			result[dt.ID] = dt
		}
	}

	return result, nil
}

// parseTaskMarkdown extracts frontmatter fields and body from a task.md file.
func parseTaskMarkdown(content, path string) (diskTask, error) {
	// Hash raw content before normalizing, so the hash matches the file on disk.
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	// Normalize CRLF → LF so parsing works on Windows.
	content = normalizeLineEndings(content)

	if !strings.HasPrefix(content, "---\n") {
		return diskTask{}, fmt.Errorf("missing frontmatter delimiter")
	}

	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) != 2 {
		return diskTask{}, fmt.Errorf("malformed frontmatter")
	}

	frontmatter := parts[0]
	body := strings.TrimSpace(parts[1])

	dt := diskTask{
		Path:        path,
		Body:        body,
		ContentHash: contentHash,
	}

	parseTaskFrontmatter(&dt, frontmatter)
	dt.Title = extractH1Title(body)
	dt.Criteria = extractTaskCriteria(body)

	if err := validateTaskFields(&dt, path); err != nil {
		return diskTask{}, err
	}

	return dt, nil
}

// parseTaskFrontmatter extracts key-value pairs and YAML lists from task frontmatter.
func parseTaskFrontmatter(dt *diskTask, frontmatter string) {
	fields := map[string]*string{
		"id":         &dt.ID,
		"spec_id":    &dt.SpecID,
		"status":     &dt.Status,
		"summary":    &dt.Summary,
		"created_by": &dt.CreatedBy,
		"updated_by": &dt.UpdatedBy,
		"created_at": &dt.CreatedAt,
		"updated_at": &dt.UpdatedAt,
	}

	currentList := "" // tracks which YAML list we're collecting
	for _, line := range strings.Split(frontmatter, "\n") {
		// Collect YAML list items.
		if currentList != "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- ") {
				item := strings.TrimPrefix(trimmed, "- ")
				switch currentList {
				case "linked_tasks":
					dt.LinkedTasks = append(dt.LinkedTasks, item)
				case "depends_on":
					dt.DependsOn = append(dt.DependsOn, item)
				}
				continue
			}
			currentList = ""
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
			dt.Position, _ = strconv.Atoi(val)
		} else if key == "linked_tasks" || key == "depends_on" {
			currentList = key
		}
	}
}

// validateTaskFields checks that all required frontmatter fields are present.
func validateTaskFields(dt *diskTask, path string) error {
	for _, check := range []struct{ val, name string }{
		{dt.ID, "id"},
		{dt.SpecID, "spec_id"},
		{dt.Title, "title"},
		{dt.Status, "status"},
		{dt.Summary, "summary"},
	} {
		if check.val == "" {
			return fmt.Errorf("%s: missing required field '%s'", path, check.name)
		}
	}
	return nil
}

// loadTaskHashesFromDB returns a map of task ID → content_hash for all tasks.
func loadTaskHashesFromDB(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT id, content_hash FROM tasks")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, fmt.Errorf("scanning task hash: %w", err)
		}
		result[id] = hash
	}
	return result, rows.Err()
}

// reconcileTasks inserts new, updates changed, and deletes removed tasks.
func reconcileTasks(db *sql.DB, diskTasks map[string]diskTask, dbTasks map[string]string, username, now string) error {
	for id := range diskTasks {
		dt := diskTasks[id]
		dbHash, exists := dbTasks[id]

		switch {
		case !exists:
			slog.Info("sync: inserting new task", "id", id, "path", dt.Path)
			if err := insertTaskFromDisk(db, &dt, username, now); err != nil {
				return fmt.Errorf("inserting task %s: %w", id, err)
			}
		case dbHash != dt.ContentHash:
			slog.Info("sync: updating changed task", "id", id, "path", dt.Path)
			if err := updateTaskFromDisk(db, &dt, username, now); err != nil {
				return fmt.Errorf("updating task %s: %w", id, err)
			}
		default:
			delete(dbTasks, id)
			continue // unchanged — skip link/criteria sync
		}

		if err := syncTaskLinks(db, dt.ID, dt.LinkedTasks); err != nil {
			return fmt.Errorf("syncing task links for %s: %w", id, err)
		}
		if err := syncTaskDependencies(db, dt.ID, dt.DependsOn); err != nil {
			return fmt.Errorf("syncing task deps for %s: %w", id, err)
		}
		if err := syncTaskCriteria(db, dt.ID, dt.Criteria); err != nil {
			return fmt.Errorf("syncing task criteria for %s: %w", id, err)
		}
		delete(dbTasks, id)
	}

	// Remaining dbTasks entries were not found on disk — delete them.
	for id := range dbTasks {
		slog.Info("sync: deleting removed task", "id", id)
		if err := deleteTaskFromDB(db, id); err != nil {
			return fmt.Errorf("deleting task %s: %w", id, err)
		}
	}

	return nil
}

// insertTaskFromDisk inserts a new task parsed from disk into the database.
func insertTaskFromDisk(db *sql.DB, dt *diskTask, username, now string) error {
	createdBy := dt.CreatedBy
	if createdBy == "" {
		createdBy = username
	}
	createdAt := dt.CreatedAt
	if createdAt == "" {
		createdAt = now
	}

	_, err := db.Exec(`
		INSERT OR REPLACE INTO tasks (id, spec_id, title, status, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, dt.ID, dt.SpecID, dt.Title, dt.Status, dt.Summary, dt.Body, dt.Path, dt.Position, createdBy, dt.UpdatedBy, dt.ContentHash, createdAt, now)
	return err
}

// updateTaskFromDisk updates an existing task in the database from disk content.
func updateTaskFromDisk(db *sql.DB, dt *diskTask, username, now string) error {
	updatedBy := dt.UpdatedBy
	if updatedBy == "" {
		updatedBy = username
	}

	_, err := db.Exec(`
		UPDATE tasks SET spec_id = ?, title = ?, status = ?, summary = ?, body = ?, path = ?,
		position = ?, updated_by = ?, content_hash = ?, updated_at = ?
		WHERE id = ?
	`, dt.SpecID, dt.Title, dt.Status, dt.Summary, dt.Body, dt.Path, dt.Position, updatedBy, dt.ContentHash, now, dt.ID)
	return err
}

// syncTaskLinks reconciles the task_links table for a given task ID.
func syncTaskLinks(db *sql.DB, taskID string, linkedTasks []string) error {
	if _, err := db.Exec("DELETE FROM task_links WHERE from_task = ?", taskID); err != nil {
		return fmt.Errorf("clearing task links: %w", err)
	}

	for _, toTask := range linkedTasks {
		toTask = strings.TrimSpace(toTask)
		if toTask == "" || toTask == taskID {
			continue
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO task_links (from_task, to_task) VALUES (?, ?)", taskID, toTask); err != nil {
			return fmt.Errorf("inserting task link %s→%s: %w", taskID, toTask, err)
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO task_links (from_task, to_task) VALUES (?, ?)", toTask, taskID); err != nil {
			return fmt.Errorf("inserting task link %s→%s: %w", toTask, taskID, err)
		}
	}

	return nil
}

// syncTaskDependencies reconciles the task_dependencies table for a given task ID.
// depends_on lists tasks that block this task.
func syncTaskDependencies(db *sql.DB, taskID string, dependsOn []string) error {
	if _, err := db.Exec("DELETE FROM task_dependencies WHERE blocked_task = ?", taskID); err != nil {
		return fmt.Errorf("clearing task deps: %w", err)
	}

	for _, blocker := range dependsOn {
		blocker = strings.TrimSpace(blocker)
		if blocker == "" || blocker == taskID {
			continue
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO task_dependencies (blocker_task, blocked_task) VALUES (?, ?)", blocker, taskID); err != nil {
			return fmt.Errorf("inserting dep %s blocks %s: %w", blocker, taskID, err)
		}
	}

	return nil
}

// syncTaskCriteria reconciles the task_criteria table for a given task ID.
// Preserves checked state for criteria that haven't changed text.
func syncTaskCriteria(db *sql.DB, taskID string, criteria []string) error {
	// Load existing criteria to preserve checked state.
	existing := make(map[string]struct {
		checked   int
		checkedBy string
	})
	rows, err := db.Query("SELECT text, checked, COALESCE(checked_by, '') FROM task_criteria WHERE task_id = ?", taskID)
	if err != nil {
		return fmt.Errorf("reading existing criteria: %w", err)
	}
	for rows.Next() {
		var text, checkedBy string
		var checked int
		if err := rows.Scan(&text, &checked, &checkedBy); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning criteria: %w", err)
		}
		existing[text] = struct {
			checked   int
			checkedBy string
		}{checked, checkedBy}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating criteria: %w", err)
	}

	// Delete and re-insert.
	if _, err := db.Exec("DELETE FROM task_criteria WHERE task_id = ?", taskID); err != nil {
		return fmt.Errorf("clearing task criteria: %w", err)
	}

	for i, text := range criteria {
		checked := 0
		checkedBy := ""
		if prev, ok := existing[text]; ok {
			checked = prev.checked
			checkedBy = prev.checkedBy
		}

		var cbPtr *string
		if checkedBy != "" {
			cbPtr = &checkedBy
		}

		if _, err := db.Exec(
			"INSERT INTO task_criteria (task_id, position, text, checked, checked_by) VALUES (?, ?, ?, ?, ?)",
			taskID, i+1, text, checked, cbPtr,
		); err != nil {
			return fmt.Errorf("inserting task criteria %d: %w", i+1, err)
		}
	}

	return nil
}

// deleteTaskFromDB removes a task from the database. ON DELETE CASCADE
// handles task_links, task_dependencies, and task_criteria cleanup.
func deleteTaskFromDB(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}
