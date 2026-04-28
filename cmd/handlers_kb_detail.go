// handlers_kb_page.go implements GET /kb/{id} — the per-KB-doc detail page.
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

// KBDocDetail describes a knowledge-base document for the detail page.
type KBDocDetail struct {
	ID         string
	Title      string
	Summary    string
	SourceType string
	Path       string
	AddedAt    string
	AddedBy    string
}

// KBChunkDetail is a single chunk within a KB document.
type KBChunkDetail struct {
	Position int
	Summary  string
	Text     string
}

// KBDetailPageData is the view model for the KB detail template.
type KBDetailPageData struct {
	Doc    KBDocDetail
	Chunks []KBChunkDetail
}

// makeKBDetailHandler returns an http.HandlerFunc that renders /kb/{id}.
func makeKBDetailHandler(freshPages func() map[string]*template.Template, devMode bool, cssHash, jsHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kbID := strings.ToUpper(r.PathValue("id"))
		if !strings.HasPrefix(kbID, "KB-") {
			http.Error(w, "invalid kb id", http.StatusBadRequest)
			return
		}

		data, err := loadKBDetailPage(kbID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			slog.Error("kb detail: load", "error", err)
			http.Error(w, "failed to load kb doc", http.StatusInternalServerError)
			return
		}

		renderPage(w, r, freshPages(), "kb_detail", &PageData{
			Title:   data.Doc.ID + " — " + data.Doc.Title,
			Active:  "kb",
			DevMode: devMode,
			CSSHash: cssHash,
			JSHash:  jsHash,
			Data:    data,
		})
	}
}

// loadKBDetailPage fetches a KB doc and its ordered chunks.
func loadKBDetailPage(kbID string) (*KBDetailPageData, error) {
	db, _, err := OpenProjectDB()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	var doc KBDocDetail
	var addedBy *string
	err = db.QueryRow(`
		SELECT id, title, summary, source_type, path, added_at, added_by
		FROM kb_docs WHERE id = ?`, kbID).Scan(
		&doc.ID, &doc.Title, &doc.Summary, &doc.SourceType, &doc.Path,
		&doc.AddedAt, &addedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("loading kb doc %s: %w", kbID, err)
	}
	if addedBy != nil {
		doc.AddedBy = *addedBy
	}

	chunks, err := loadKBChunks(db, kbID)
	if err != nil {
		return nil, err
	}

	return &KBDetailPageData{Doc: doc, Chunks: chunks}, nil
}

// loadKBChunks returns chunks for a KB doc ordered by position.
func loadKBChunks(db *sql.DB, kbID string) ([]KBChunkDetail, error) {
	rows, err := db.Query(`
		SELECT position, summary, text
		FROM kb_chunks WHERE doc_id = ?
		ORDER BY position`, kbID)
	if err != nil {
		return nil, fmt.Errorf("loading kb chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []KBChunkDetail
	for rows.Next() {
		var c KBChunkDetail
		if err := rows.Scan(&c.Position, &c.Summary, &c.Text); err != nil {
			return nil, fmt.Errorf("scanning kb chunk: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating kb chunks: %w", err)
	}
	return out, nil
}
