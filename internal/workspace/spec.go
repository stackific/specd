package workspace

import (
	"database/sql"
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
	Type    string // business, technical, non-technical
	Summary string
	Body    string
}

// NewSpecResult is the JSON response from new-spec.
type NewSpecResult struct {
	ID   string `json:"id"`
	Path string `json:"path"`
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
