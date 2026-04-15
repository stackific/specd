// Package workspace — trash.go implements soft-delete recovery operations:
// list trashed items, restore from trash, purge old trash, and purge all.
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
)

// TrashItem represents a soft-deleted item in the trash table.
type TrashItem struct {
	ID           int    `json:"id"`
	Kind         string `json:"kind"` // "spec", "task", or "kb"
	OriginalID   string `json:"original_id"`
	OriginalPath string `json:"original_path"`
	Metadata     string `json:"metadata"`
	DeletedAt    string `json:"deleted_at"`
	DeletedBy    string `json:"deleted_by"`
}

// TrashListFilter holds filter options for listing trash.
type TrashListFilter struct {
	Kind     string // "spec", "task", "kb", or "" for all
	OlderThan string // duration string like "30d"; empty for no age filter
}

// ListTrash returns soft-deleted items from the trash table.
func (w *Workspace) ListTrash(filter TrashListFilter) ([]TrashItem, error) {
	query := `SELECT id, kind, original_id, original_path, metadata, deleted_at, deleted_by
		FROM trash WHERE 1=1`
	var args []any

	if filter.Kind != "" {
		query += " AND kind = ?"
		args = append(args, filter.Kind)
	}

	if filter.OlderThan != "" {
		dur, err := parseDuration(filter.OlderThan)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", filter.OlderThan, err)
		}
		cutoff := time.Now().UTC().Add(-dur).Format(time.RFC3339)
		query += " AND deleted_at < ?"
		args = append(args, cutoff)
	}

	query += " ORDER BY deleted_at DESC"

	rows, err := w.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list trash: %w", err)
	}
	defer rows.Close()

	var items []TrashItem
	for rows.Next() {
		var item TrashItem
		if err := rows.Scan(&item.ID, &item.Kind, &item.OriginalID, &item.OriginalPath,
			&item.Metadata, &item.DeletedAt, &item.DeletedBy); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// RestoreTrashResult holds the result of a trash restore operation.
type RestoreTrashResult struct {
	Kind       string `json:"kind"`
	OriginalID string `json:"original_id"`
	RestoredID string `json:"restored_id"`
	Path       string `json:"path"`
	Warning    string `json:"warning,omitempty"`
}

// RestoreTrash recovers a soft-deleted item from trash.
func (w *Workspace) RestoreTrash(trashID int) (*RestoreTrashResult, error) {
	var result *RestoreTrashResult

	err := w.WithLock(func() error {
		// Read the trash entry.
		var kind, originalID, originalPath, metadataJSON, deletedAt string
		var content []byte
		err := w.DB.QueryRow(`SELECT kind, original_id, original_path, content, metadata, deleted_at
			FROM trash WHERE id = ?`, trashID).Scan(
			&kind, &originalID, &originalPath, &content, &metadataJSON, &deletedAt)
		if err != nil {
			return fmt.Errorf("trash item %d not found", trashID)
		}

		result = &RestoreTrashResult{
			Kind:       kind,
			OriginalID: originalID,
			RestoredID: originalID,
			Path:       originalPath,
		}

		switch kind {
		case "spec":
			return w.restoreSpec(trashID, originalID, originalPath, content, metadataJSON, result)
		case "task":
			return w.restoreTask(trashID, originalID, originalPath, content, metadataJSON, result)
		case "kb":
			return w.restoreKB(trashID, originalID, originalPath, content, metadataJSON, result)
		default:
			return fmt.Errorf("unknown trash kind: %s", kind)
		}
	})

	return result, err
}

// restoreSpec recreates a spec from trash data.
func (w *Workspace) restoreSpec(trashID int, originalID, originalPath string, content []byte, metadataJSON string, result *RestoreTrashResult) error {
	// Check if the original ID is already in use.
	var exists int
	w.DB.QueryRow("SELECT COUNT(*) FROM specs WHERE id = ?", originalID).Scan(&exists)

	restoredID := originalID
	restoredPath := originalPath

	if exists > 0 {
		// Allocate a new ID.
		newNum, err := w.DB.NextID("spec")
		if err != nil {
			return fmt.Errorf("allocate new spec id: %w", err)
		}
		restoredID = fmt.Sprintf("SPEC-%d", newNum)
		result.RestoredID = restoredID
		result.Warning = fmt.Sprintf("original ID %s is in use; restored as %s", originalID, restoredID)

		// Rewrite the path to use the new ID.
		oldDirBase := filepath.Base(filepath.Dir(originalPath))
		newDirBase := restoredID + oldDirBase[len(originalID):]
		restoredPath = filepath.Join("specd", "specs", newDirBase, "spec.md")
		result.Path = restoredPath
	}

	// Parse metadata.
	var meta struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	json.Unmarshal([]byte(metadataJSON), &meta)

	// Recreate directory and file.
	absDir := filepath.Join(w.Root, filepath.Dir(restoredPath))
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return fmt.Errorf("create spec dir: %w", err)
	}

	absPath := filepath.Join(w.Root, restoredPath)
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}

	// Parse the content to get the full spec data.
	doc, err := parseMDContent(string(content))
	if err != nil {
		return fmt.Errorf("parse restored spec: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userName, _ := w.DB.GetMeta("user_name")
	contentHash := hashContent(string(content))

	// Get max position.
	var maxPos int
	w.DB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM specs").Scan(&maxPos)

	_, err = w.DB.Exec(`INSERT INTO specs (id, slug, title, type, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		restoredID, Slugify(doc.title), doc.title, doc.specType, doc.summary, doc.body,
		restoredPath, maxPos+1, userName, userName, contentHash, now, now)
	if err != nil {
		return fmt.Errorf("insert restored spec: %w", err)
	}

	// Remove from trash.
	w.DB.Exec("DELETE FROM trash WHERE id = ?", trashID)

	return nil
}

// restoreTask recreates a task from trash data.
func (w *Workspace) restoreTask(trashID int, originalID, originalPath string, content []byte, metadataJSON string, result *RestoreTrashResult) error {
	// Parse metadata for spec_id.
	var meta struct {
		SpecID string `json:"spec_id"`
		Status string `json:"status"`
	}
	json.Unmarshal([]byte(metadataJSON), &meta)

	// Verify parent spec still exists.
	if _, err := w.ReadSpec(meta.SpecID); err != nil {
		return fmt.Errorf("parent spec %s no longer exists; cannot restore task", meta.SpecID)
	}

	// Check if the original ID is already in use.
	var exists int
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", originalID).Scan(&exists)

	restoredID := originalID
	restoredPath := originalPath

	if exists > 0 {
		newNum, err := w.DB.NextID("task")
		if err != nil {
			return fmt.Errorf("allocate new task id: %w", err)
		}
		restoredID = fmt.Sprintf("TASK-%d", newNum)
		result.RestoredID = restoredID
		result.Warning = fmt.Sprintf("original ID %s is in use; restored as %s", originalID, restoredID)

		// Rewrite the filename.
		dir := filepath.Dir(originalPath)
		oldBase := filepath.Base(originalPath)
		newBase := restoredID + oldBase[len(originalID):]
		restoredPath = filepath.Join(dir, newBase)
		result.Path = restoredPath
	}

	// Write file.
	absPath := filepath.Join(w.Root, restoredPath)
	absDir := filepath.Dir(absPath)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return fmt.Errorf("create task dir: %w", err)
	}
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		return fmt.Errorf("write task file: %w", err)
	}

	doc, err := parseMDContent(string(content))
	if err != nil {
		return fmt.Errorf("parse restored task: %w", err)
	}

	status := meta.Status
	if status == "" {
		status = doc.status
	}
	if status == "" {
		status = "backlog"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userName, _ := w.DB.GetMeta("user_name")
	contentHash := hashContent(string(content))

	// Get max position for this status.
	var maxPos int
	w.DB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM tasks WHERE status = ?", status).Scan(&maxPos)

	_, err = w.DB.Exec(`INSERT INTO tasks (id, slug, spec_id, title, status, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		restoredID, Slugify(doc.title), meta.SpecID, doc.title, status, doc.summary, doc.body,
		restoredPath, maxPos+1, userName, userName, contentHash, now, now)
	if err != nil {
		return fmt.Errorf("insert restored task: %w", err)
	}

	// Remove from trash.
	w.DB.Exec("DELETE FROM trash WHERE id = ?", trashID)

	return nil
}

// restoreKB recreates a KB document from trash data.
func (w *Workspace) restoreKB(trashID int, originalID, originalPath string, content []byte, metadataJSON string, result *RestoreTrashResult) error {
	// Check if the original ID is already in use.
	var exists int
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_docs WHERE id = ?", originalID).Scan(&exists)

	restoredID := originalID
	restoredPath := originalPath

	if exists > 0 {
		newNum, err := w.DB.NextID("kb")
		if err != nil {
			return fmt.Errorf("allocate new kb id: %w", err)
		}
		restoredID = fmt.Sprintf("KB-%d", newNum)
		result.RestoredID = restoredID
		result.Warning = fmt.Sprintf("original ID %s is in use; restored as %s", originalID, restoredID)

		// Rewrite filename.
		oldBase := filepath.Base(originalPath)
		newBase := restoredID + oldBase[len(originalID):]
		restoredPath = filepath.Join("specd", "kb", newBase)
		result.Path = restoredPath
	}

	// Parse metadata.
	var meta struct {
		Title      string `json:"title"`
		SourceType string `json:"source_type"`
	}
	json.Unmarshal([]byte(metadataJSON), &meta)

	// Ensure kb directory exists.
	kbDir := filepath.Join(w.Root, "specd", "kb")
	os.MkdirAll(kbDir, 0o755)

	// Write file.
	absPath := filepath.Join(w.Root, restoredPath)
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		return fmt.Errorf("write kb file: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userName, _ := w.DB.GetMeta("user_name")

	srcType := meta.SourceType
	if srcType == "" {
		srcType = detectSourceType(restoredPath)
	}

	title := meta.Title
	if title == "" {
		title = titleFromFilename(filepath.Base(restoredPath))
	}

	// Chunk the document.
	var chunks []Chunk
	switch srcType {
	case "md":
		chunks = ChunkMarkdown(string(content))
	case "txt":
		chunks = ChunkPlainText(string(content))
	case "html":
		var chunkErr error
		chunks, chunkErr = ChunkHTML(string(content))
		if chunkErr != nil {
			// Fall back to plain text chunking.
			chunks = ChunkPlainText(string(content))
		}
	}

	tx, err := w.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO kb_docs (id, slug, title, source_type, path, content_hash, added_at, added_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		restoredID, Slugify(title), title, srcType, restoredPath,
		hashContent(string(content)), now, userName)
	if err != nil {
		return fmt.Errorf("insert restored kb_docs: %w", err)
	}

	for _, c := range chunks {
		_, err = tx.Exec(`INSERT INTO kb_chunks (doc_id, position, text, char_start, char_end, page)
			VALUES (?, ?, ?, ?, ?, ?)`,
			restoredID, c.Position, c.Text, c.CharStart, c.CharEnd, c.Page)
		if err != nil {
			return fmt.Errorf("insert chunk: %w", err)
		}
	}

	// Remove from trash.
	_, err = tx.Exec("DELETE FROM trash WHERE id = ?", trashID)
	if err != nil {
		return fmt.Errorf("remove from trash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// PurgeTrash permanently removes trash entries older than the given duration.
func (w *Workspace) PurgeTrash(olderThan string) (int, error) {
	var count int

	err := w.WithLock(func() error {
		dur, err := parseDuration(olderThan)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", olderThan, err)
		}

		cutoff := time.Now().UTC().Add(-dur).Format(time.RFC3339)
		res, err := w.DB.Exec("DELETE FROM trash WHERE deleted_at < ?", cutoff)
		if err != nil {
			return fmt.Errorf("purge trash: %w", err)
		}
		n, _ := res.RowsAffected()
		count = int(n)
		return nil
	})

	return count, err
}

// PurgeAllTrash permanently removes all trash entries.
func (w *Workspace) PurgeAllTrash() (int, error) {
	var count int

	err := w.WithLock(func() error {
		res, err := w.DB.Exec("DELETE FROM trash")
		if err != nil {
			return fmt.Errorf("purge all trash: %w", err)
		}
		n, _ := res.RowsAffected()
		count = int(n)
		return nil
	})

	return count, err
}

// parseDuration parses a simple duration string like "30d", "7d", "24h".
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short: %q", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	var num int
	if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid number in duration: %q", s)
	}

	switch unit {
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'm':
		return time.Duration(num) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %c", unit)
	}
}

// parseMDContent is a minimal helper for extracting title, type, summary,
// status, and body from a markdown file with frontmatter.
type parsedMDContent struct {
	title    string
	specType string
	summary  string
	status   string
	body     string
}

func parseMDContent(content string) (*parsedMDContent, error) {
	doc, err := parseFrontmatterContent(content)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// parseFrontmatterContent extracts frontmatter fields from markdown content.
func parseFrontmatterContent(content string) (*parsedMDContent, error) {
	fm, err := frontmatterParse(content)
	if err != nil {
		return &parsedMDContent{body: content}, nil
	}

	result := &parsedMDContent{body: fm.body}

	// Parse YAML fields manually from raw frontmatter.
	for _, line := range splitLines(fm.raw) {
		line = trimSpace(line)
		if kv := splitKV(line); kv != nil {
			switch kv.key {
			case "title":
				result.title = kv.value
			case "type":
				result.specType = kv.value
			case "summary":
				result.summary = kv.value
			case "status":
				result.status = kv.value
			}
		}
	}

	return result, nil
}

type fmParsed struct {
	raw  string
	body string
}

// frontmatterParse is a lightweight frontmatter splitter.
func frontmatterParse(content string) (*fmParsed, error) {
	parsed, err := frontmatter.Parse(content)
	if err != nil {
		return nil, err
	}
	return &fmParsed{raw: parsed.RawFrontmatter, body: parsed.Body}, nil
}

type kvPair struct {
	key   string
	value string
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range splitByNewline(s) {
		lines = append(lines, line)
	}
	return lines
}

func splitByNewline(s string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func splitKV(line string) *kvPair {
	idx := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	key := trimSpace(line[:idx])
	value := trimSpace(line[idx+1:])
	// Remove surrounding quotes if present.
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		value = value[1 : len(value)-1]
	}
	return &kvPair{key: key, value: value}
}

// hashContent computes a hash of content for the content_hash column.
func hashContent(content string) string {
	return hash.String(content)
}
