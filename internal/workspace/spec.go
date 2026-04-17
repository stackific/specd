// Package workspace — spec.go implements spec lifecycle operations: create,
// read, list, update, rename, and soft-delete. Each mutation acquires the
// workspace lock, updates both the markdown file and SQLite in a single
// transaction, and syncs system-managed frontmatter fields.
package workspace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
)

// Spec is the domain representation of a spec.
type Spec struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	Position  int    `json:"position"`
	CreatedBy string `json:"created_by,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// NewSpecInput holds the parameters for creating a new spec.
type NewSpecInput struct {
	Title   string
	Type    string // business, functional, non-functional
	Summary string
	Body    string
}

// NewSpecResult is the JSON response from new-spec.
type NewSpecResult struct {
	ID         string            `json:"id"`
	Path       string            `json:"path"`
	Candidates *CandidatesResult `json:"candidates,omitempty"`
}

// NewSpec creates a new spec in the workspace.
func (w *Workspace) NewSpec(input NewSpecInput) (*NewSpecResult, error) {
	var result *NewSpecResult

	err := w.WithLock(func() error {
		id, err := w.DB.NextID("spec")
		if err != nil {
			return fmt.Errorf("allocate spec id: %w", err)
		}

		specID := fmt.Sprintf("SPEC-%d", id)
		slug := Slugify(input.Title)
		dirName := fmt.Sprintf("%s-%s", specID, slug)
		relDir := filepath.Join("specd", "specs", dirName)
		relPath := filepath.Join(relDir, "spec.md")
		absDir := filepath.Join(w.Root, relDir)

		if err := os.MkdirAll(absDir, 0o755); err != nil {
			return fmt.Errorf("create spec dir: %w", err)
		}

		// Render markdown file.
		fm := &frontmatter.SpecFrontmatter{
			Title:   input.Title,
			Type:    input.Type,
			Summary: input.Summary,
		}
		content, err := frontmatter.RenderSpec(fm, input.Body)
		if err != nil {
			return fmt.Errorf("render spec: %w", err)
		}

		absPath := filepath.Join(w.Root, relPath)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write spec file: %w", err)
		}

		contentHash := hash.String(content)
		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		// Get max position.
		var maxPos int
		err = w.DB.QueryRow("SELECT COALESCE(MAX(position), -1) FROM specs").Scan(&maxPos)
		if err != nil {
			return err
		}

		_, err = w.DB.Exec(`INSERT INTO specs (id, slug, title, type, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			specID, slug, input.Title, input.Type, input.Summary, input.Body,
			relPath, maxPos+1, userName, userName, contentHash, now, now)
		if err != nil {
			return fmt.Errorf("insert spec: %w", err)
		}

		// Update index.md and log.md.
		w.appendIndex(specID, input.Title)
		w.appendLog(specID, input.Title, now)

		result = &NewSpecResult{ID: specID, Path: relPath}
		return nil
	})

	// Compute candidates outside the lock (read-only).
	if err == nil && result != nil {
		candidates, _ := w.Candidates(result.ID, 20)
		result.Candidates = candidates
	}

	return result, err
}

// ReadSpec reads a spec by ID.
func (w *Workspace) ReadSpec(specID string) (*Spec, error) {
	s := &Spec{}
	err := w.DB.QueryRow(`SELECT id, slug, title, type, summary, body, path, position,
		COALESCE(created_by, ''), COALESCE(updated_by, ''), created_at, updated_at
		FROM specs WHERE id = ?`, specID).Scan(
		&s.ID, &s.Slug, &s.Title, &s.Type, &s.Summary, &s.Body, &s.Path, &s.Position,
		&s.CreatedBy, &s.UpdatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("spec %s not found", specID)
	}
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	return s, nil
}

// ListSpecsFilter holds filter options for listing specs.
type ListSpecsFilter struct {
	Type     string
	LinkedTo string
	Empty    bool // only specs with zero tasks
	Limit    int
}

// ListSpecs returns specs matching the filter.
func (w *Workspace) ListSpecs(filter ListSpecsFilter) ([]Spec, error) {
	query := `SELECT id, slug, title, type, summary, body, path, position,
		COALESCE(created_by, ''), COALESCE(updated_by, ''), created_at, updated_at
		FROM specs WHERE 1=1`
	var args []any

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.LinkedTo != "" {
		query += ` AND id IN (
			SELECT to_spec FROM spec_links WHERE from_spec = ?
			UNION SELECT from_spec FROM spec_links WHERE to_spec = ?)`
		args = append(args, filter.LinkedTo, filter.LinkedTo)
	}
	if filter.Empty {
		query += ` AND id NOT IN (SELECT DISTINCT spec_id FROM tasks)`
	}

	query += " ORDER BY position ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := w.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list specs: %w", err)
	}
	defer rows.Close()

	var specs []Spec
	for rows.Next() {
		var s Spec
		if err := rows.Scan(&s.ID, &s.Slug, &s.Title, &s.Type, &s.Summary, &s.Body,
			&s.Path, &s.Position, &s.CreatedBy, &s.UpdatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		specs = append(specs, s)
	}
	return specs, rows.Err()
}

// UpdateSpecInput holds optional fields for updating a spec.
type UpdateSpecInput struct {
	Title   *string
	Type    *string
	Summary *string
	Body    *string
}

// UpdateSpec updates mutable fields on a spec.
func (w *Workspace) UpdateSpec(specID string, input UpdateSpecInput) error {
	return w.WithLock(func() error {
		spec, err := w.ReadSpec(specID)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		title := spec.Title
		typ := spec.Type
		summary := spec.Summary
		body := spec.Body

		if input.Title != nil {
			title = *input.Title
		}
		if input.Type != nil {
			typ = *input.Type
		}
		if input.Summary != nil {
			summary = *input.Summary
		}
		if input.Body != nil {
			body = *input.Body
		}

		// Rewrite markdown file.
		fm := &frontmatter.SpecFrontmatter{
			Title:   title,
			Type:    typ,
			Summary: summary,
		}

		// Preserve existing system-managed fields by reading current frontmatter.
		absPath := filepath.Join(w.Root, spec.Path)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read spec file: %w", err)
		}
		doc, err := frontmatter.Parse(string(data))
		if err != nil {
			return fmt.Errorf("parse spec: %w", err)
		}
		existingFM, err := frontmatter.DecodeSpec(doc.RawFrontmatter)
		if err != nil {
			return err
		}
		fm.LinkedSpecs = existingFM.LinkedSpecs
		fm.Cites = existingFM.Cites

		content, err := frontmatter.RenderSpec(fm, body)
		if err != nil {
			return err
		}

		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return err
		}

		contentHash := hash.String(content)
		_, err = w.DB.Exec(`UPDATE specs SET title=?, type=?, summary=?, body=?,
			updated_by=?, content_hash=?, updated_at=? WHERE id=?`,
			title, typ, summary, body, userName, contentHash, now, specID)
		return err
	})
}

// RenameSpec changes a spec's title and updates its slug and folder name.
func (w *Workspace) RenameSpec(specID string, newTitle string) error {
	return w.WithLock(func() error {
		spec, err := w.ReadSpec(specID)
		if err != nil {
			return err
		}

		newSlug := Slugify(newTitle)
		newDirName := fmt.Sprintf("%s-%s", specID, newSlug)
		oldDir := filepath.Join(w.Root, filepath.Dir(spec.Path))
		newDir := filepath.Join(w.Root, "specd", "specs", newDirName)

		if err := os.Rename(oldDir, newDir); err != nil {
			return fmt.Errorf("rename spec dir: %w", err)
		}

		newRelPath := filepath.Join("specd", "specs", newDirName, "spec.md")
		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		// Rewrite frontmatter with new title.
		absPath := filepath.Join(w.Root, newRelPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read spec: %w", err)
		}
		doc, err := frontmatter.Parse(string(data))
		if err != nil {
			return err
		}
		fm, err := frontmatter.DecodeSpec(doc.RawFrontmatter)
		if err != nil {
			return err
		}
		fm.Title = newTitle
		content, err := frontmatter.RenderSpec(fm, doc.Body)
		if err != nil {
			return err
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return err
		}

		contentHash := hash.String(content)
		_, err = w.DB.Exec(`UPDATE specs SET title=?, slug=?, path=?,
			updated_by=?, content_hash=?, updated_at=? WHERE id=?`,
			newTitle, newSlug, newRelPath, userName, contentHash, now, specID)
		if err != nil {
			return err
		}

		// Update task paths that reference this spec dir.
		rows, err := w.DB.Query("SELECT id, path FROM tasks WHERE spec_id = ?", specID)
		if err != nil {
			return err
		}
		type taskPath struct{ id, path string }
		var tasks []taskPath
		for rows.Next() {
			var tp taskPath
			rows.Scan(&tp.id, &tp.path)
			tasks = append(tasks, tp)
		}
		rows.Close()

		for _, tp := range tasks {
			newTaskPath := filepath.Join("specd", "specs", newDirName, filepath.Base(tp.path))
			w.DB.Exec("UPDATE tasks SET path = ? WHERE id = ?", newTaskPath, tp.id)
		}

		return nil
	})
}

// DeleteSpec soft-deletes a spec and its tasks to trash.
func (w *Workspace) DeleteSpec(specID string) error {
	return w.WithLock(func() error {
		spec, err := w.ReadSpec(specID)
		if err != nil {
			return err
		}

		absDir := filepath.Join(w.Root, filepath.Dir(spec.Path))
		now := time.Now().UTC().Format(time.RFC3339)

		// Read spec file for trash.
		absPath := filepath.Join(w.Root, spec.Path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read spec for trash: %w", err)
		}

		metaBytes, _ := json.Marshal(map[string]string{
			"id": spec.ID, "title": spec.Title, "type": spec.Type, "path": spec.Path,
		})
		metadata := string(metaBytes)

		tx, err := w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Trash the spec.
		_, err = tx.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
			VALUES ('spec', ?, ?, ?, ?, ?, 'cli')`,
			spec.ID, spec.Path, content, metadata, now)
		if err != nil {
			return fmt.Errorf("insert spec trash: %w", err)
		}

		// Delete citations referencing this spec (not FK-cascaded).
		_, err = tx.Exec("DELETE FROM citations WHERE from_kind = 'spec' AND from_id = ?", specID)
		if err != nil {
			return fmt.Errorf("delete spec citations: %w", err)
		}

		// Delete from specs (cascades tasks, links, etc).
		_, err = tx.Exec("DELETE FROM specs WHERE id = ?", specID)
		if err != nil {
			return fmt.Errorf("delete spec: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		// Remove the entire spec directory from disk.
		os.RemoveAll(absDir)
		return nil
	})
}

// appendIndex adds a spec entry to specd/specs/index.md.
func (w *Workspace) appendIndex(specID, title string) {
	f, err := os.OpenFile(w.IndexPath(), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "- **%s**: %s\n", specID, title)
}

// appendLog adds a timestamped creation entry to specd/specs/log.md.
func (w *Workspace) appendLog(specID, title, timestamp string) {
	f, err := os.OpenFile(w.LogPath(), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "- %s — Created **%s**: %s\n", timestamp, specID, title)
}
