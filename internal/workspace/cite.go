// Package workspace — cite.go implements KB citation operations. Specs and
// tasks can cite specific KB chunks. Citations are stored in the citations
// table and synced to the cites frontmatter field.
package workspace

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stackific/specd/internal/frontmatter"
)

// CitationInput represents a single citation reference in the format "KB-N:position".
type CitationInput struct {
	KBID          string
	ChunkPosition int
}

// CitationDetail holds a citation with its associated chunk content.
type CitationDetail struct {
	FromKind      string `json:"from_kind"`
	FromID        string `json:"from_id"`
	KBDocID       string `json:"kb_doc_id"`
	KBDocTitle    string `json:"kb_doc_title"`
	SourceType    string `json:"source_type"`
	ChunkPosition int    `json:"chunk_position"`
	ChunkText     string `json:"chunk_text"`
	Page          *int   `json:"page,omitempty"`
}

// ParseCitationRef parses a citation reference string like "KB-4:12".
func ParseCitationRef(ref string) (*CitationInput, error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid citation format %q (expected KB-N:position)", ref)
	}
	if !strings.HasPrefix(parts[0], "KB-") {
		return nil, fmt.Errorf("invalid citation format %q (must start with KB-)", ref)
	}
	pos, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid chunk position in %q: %w", ref, err)
	}
	return &CitationInput{KBID: parts[0], ChunkPosition: pos}, nil
}

// Cite adds KB chunk citations to a spec or task. Updates the citations
// table and syncs the cites frontmatter field.
func (w *Workspace) Cite(id string, refs []CitationInput) error {
	return w.WithLock(func() error {
		kind, err := w.resolveKind(id)
		if err != nil {
			return err
		}

		// Validate all refs first.
		for _, ref := range refs {
			if _, err := w.getKBDoc(ref.KBID); err != nil {
				return err
			}
			if _, err := w.getKBChunk(ref.KBID, ref.ChunkPosition); err != nil {
				return err
			}
		}

		// Insert citations.
		now := time.Now().UTC().Format(time.RFC3339)
		for _, ref := range refs {
			_, err := w.DB.Exec(`INSERT OR IGNORE INTO citations (from_kind, from_id, kb_doc_id, chunk_position, created_at)
				VALUES (?, ?, ?, ?, ?)`,
				kind, id, ref.KBID, ref.ChunkPosition, now)
			if err != nil {
				return fmt.Errorf("insert citation: %w", err)
			}
		}

		return w.syncCites(kind, id)
	})
}

// Uncite removes KB chunk citations from a spec or task.
func (w *Workspace) Uncite(id string, refs []CitationInput) error {
	return w.WithLock(func() error {
		kind, err := w.resolveKind(id)
		if err != nil {
			return err
		}

		for _, ref := range refs {
			_, err := w.DB.Exec(`DELETE FROM citations
				WHERE from_kind = ? AND from_id = ? AND kb_doc_id = ? AND chunk_position = ?`,
				kind, id, ref.KBID, ref.ChunkPosition)
			if err != nil {
				return fmt.Errorf("delete citation: %w", err)
			}
		}

		return w.syncCites(kind, id)
	})
}

// GetCitations returns all citations for a spec or task with chunk details.
func (w *Workspace) GetCitations(id string) ([]CitationDetail, error) {
	kind, err := w.resolveKind(id)
	if err != nil {
		return nil, err
	}

	rows, err := w.DB.Query(`
		SELECT c.from_kind, c.from_id, c.kb_doc_id, d.title, d.source_type,
			c.chunk_position, k.text, k.page
		FROM citations c
		JOIN kb_docs d ON d.id = c.kb_doc_id
		JOIN kb_chunks k ON k.doc_id = c.kb_doc_id AND k.position = c.chunk_position
		WHERE c.from_kind = ? AND c.from_id = ?
		ORDER BY c.kb_doc_id, c.chunk_position`, kind, id)
	if err != nil {
		return nil, fmt.Errorf("query citations: %w", err)
	}
	defer rows.Close()

	var citations []CitationDetail
	for rows.Next() {
		var cd CitationDetail
		var pageVal sql.NullInt64

		err := rows.Scan(&cd.FromKind, &cd.FromID, &cd.KBDocID, &cd.KBDocTitle,
			&cd.SourceType, &cd.ChunkPosition, &cd.ChunkText, &pageVal)
		if err != nil {
			return nil, err
		}
		if pageVal.Valid {
			pg := int(pageVal.Int64)
			cd.Page = &pg
		}
		// Truncate text for display.
		if len(cd.ChunkText) > 200 {
			cd.ChunkText = cd.ChunkText[:200] + "..."
		}
		citations = append(citations, cd)
	}
	return citations, rows.Err()
}

// syncCites reads current citations from SQLite and updates the spec or
// task's cites frontmatter field.
func (w *Workspace) syncCites(kind, id string) error {
	rows, err := w.DB.Query(`
		SELECT kb_doc_id, chunk_position
		FROM citations
		WHERE from_kind = ? AND from_id = ?
		ORDER BY kb_doc_id, chunk_position`, kind, id)
	if err != nil {
		return fmt.Errorf("load citations: %w", err)
	}
	defer rows.Close()

	// Group by KB doc, preserving order.
	citesMap := make(map[string][]int)
	var order []string
	for rows.Next() {
		var kbID string
		var pos int
		rows.Scan(&kbID, &pos)
		if _, exists := citesMap[kbID]; !exists {
			order = append(order, kbID)
		}
		citesMap[kbID] = append(citesMap[kbID], pos)
	}

	var cites []frontmatter.CitationRef
	for _, kbID := range order {
		cites = append(cites, frontmatter.CitationRef{
			KB:     kbID,
			Chunks: citesMap[kbID],
		})
	}

	switch kind {
	case "spec":
		spec, err := w.ReadSpec(id)
		if err != nil {
			return err
		}
		return w.rewriteSpecFrontmatter(spec, func(fm *frontmatter.SpecFrontmatter) {
			fm.Cites = cites
		})
	case "task":
		task, err := w.ReadTask(id)
		if err != nil {
			return err
		}
		return w.rewriteTaskFrontmatter(task, func(fm *frontmatter.TaskFrontmatter) {
			fm.Cites = cites
		})
	}
	return nil
}

// resolveKind determines whether an ID is a spec or task.
func (w *Workspace) resolveKind(id string) (string, error) {
	if isSpec(id) {
		if _, err := w.ReadSpec(id); err != nil {
			return "", err
		}
		return "spec", nil
	}
	if isTask(id) {
		if _, err := w.ReadTask(id); err != nil {
			return "", err
		}
		return "task", nil
	}
	return "", fmt.Errorf("invalid ID format: %s (expected SPEC-N or TASK-N)", id)
}
