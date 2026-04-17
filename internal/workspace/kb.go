// Package workspace — kb.go implements knowledge base operations: adding,
// listing, reading, searching, and removing KB documents. Documents are
// copied into specd/kb/ with generated filenames, chunked, and indexed
// in SQLite with FTS5 and trigram indexes.
package workspace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stackific/specd/internal/hash"
)

// KBDoc is the domain representation of a knowledge base document.
type KBDoc struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	SourceType  string  `json:"source_type"`
	Path        string  `json:"path"`
	CleanPath   *string `json:"clean_path,omitempty"`
	Note        *string `json:"note,omitempty"`
	PageCount   *int    `json:"page_count,omitempty"`
	ContentHash string  `json:"content_hash"`
	AddedAt     string  `json:"added_at"`
	AddedBy     string  `json:"added_by,omitempty"`
}

// KBChunk is the domain representation of a single chunk.
type KBChunk struct {
	ID        int    `json:"id"`
	DocID     string `json:"doc_id"`
	Position  int    `json:"position"`
	Text      string `json:"text"`
	CharStart int    `json:"char_start"`
	CharEnd   int    `json:"char_end"`
	Page      *int   `json:"page,omitempty"`
}

// KBAddInput holds the parameters for adding a KB document.
type KBAddInput struct {
	Source string // file path or URL
	Title  string // optional override title
	Note   string // optional note
}

// KBAddResult is the JSON response from kb add.
type KBAddResult struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	SourceType string `json:"source_type"`
	ChunkCount int    `json:"chunk_count"`
	PageCount  *int   `json:"page_count,omitempty"`
}

// KBAdd adds a document to the knowledge base. It copies the source file
// into specd/kb/, detects the source type, chunks the content, and indexes
// everything in SQLite.
func (w *Workspace) KBAdd(input KBAddInput) (*KBAddResult, error) {
	var result *KBAddResult

	err := w.WithLock(func() error {
		// Ensure kb directory exists.
		kbDir := w.KBDir()
		if err := os.MkdirAll(kbDir, 0o755); err != nil {
			return fmt.Errorf("create kb dir: %w", err)
		}

		// Resolve source: URL or local file.
		srcPath, srcType, cleanup, err := resolveSource(input.Source)
		if err != nil {
			return fmt.Errorf("resolve source: %w", err)
		}
		if cleanup != nil {
			defer cleanup()
		}

		// Read the source content.
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read source: %w", err)
		}

		// Allocate KB ID.
		id, err := w.DB.NextID("kb")
		if err != nil {
			return fmt.Errorf("allocate kb id: %w", err)
		}
		kbID := fmt.Sprintf("KB-%d", id)

		// Derive title.
		title := input.Title
		if title == "" {
			title = titleFromFilename(filepath.Base(srcPath))
		}

		slug := Slugify(title)
		ext := extForType(srcType)
		filename := fmt.Sprintf("%s-%s%s", kbID, slug, ext)
		relPath := filepath.Join("specd", "kb", filename)
		absPath := filepath.Join(w.Root, relPath)

		// Copy file into kb directory.
		if err := os.WriteFile(absPath, data, 0o644); err != nil {
			return fmt.Errorf("write kb file: %w", err)
		}

		contentHash := hash.Bytes(data)
		now := time.Now().UTC().Format(time.RFC3339)
		userName, _ := w.DB.GetMeta("user_name")

		// Chunk the document.
		var chunks []Chunk
		var pageCount *int
		var cleanRelPath *string

		switch srcType {
		case "md":
			chunks = ChunkMarkdown(string(data))
		case "txt":
			chunks = ChunkPlainText(string(data))
		case "html":
			// Sanitize and write sidecar.
			cleanHTML := SanitizeHTML(string(data))
			cleanFilename := fmt.Sprintf("%s-%s.clean.html", kbID, slug)
			cleanRel := filepath.Join("specd", "kb", cleanFilename)
			cleanAbs := filepath.Join(w.Root, cleanRel)
			if err := os.WriteFile(cleanAbs, []byte(cleanHTML), 0o644); err != nil {
				return fmt.Errorf("write clean HTML: %w", err)
			}
			cleanRelPath = &cleanRel

			var chunkErr error
			chunks, chunkErr = ChunkHTML(string(data))
			if chunkErr != nil {
				return fmt.Errorf("chunk HTML: %w", chunkErr)
			}
		case "pdf":
			var pc int
			var chunkErr error
			chunks, pc, chunkErr = ChunkPDF(absPath)
			if chunkErr != nil {
				// Clean up the copied file on failure.
				os.Remove(absPath)
				return fmt.Errorf("chunk PDF: %w", chunkErr)
			}
			pageCount = &pc
		}

		// Begin SQLite transaction.
		tx, err := w.DB.Begin()
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		// Insert kb_docs row.
		_, err = tx.Exec(`INSERT INTO kb_docs (id, slug, title, source_type, path, clean_path, note, page_count, content_hash, added_at, added_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			kbID, slug, title, srcType, relPath,
			nilString(cleanRelPath), nilString(notePtr(input.Note)),
			pageCount, contentHash, now, userName)
		if err != nil {
			return fmt.Errorf("insert kb_docs: %w", err)
		}

		// Insert chunks.
		for _, c := range chunks {
			_, err = tx.Exec(`INSERT INTO kb_chunks (doc_id, position, text, char_start, char_end, page)
				VALUES (?, ?, ?, ?, ?, ?)`,
				kbID, c.Position, c.Text, c.CharStart, c.CharEnd, c.Page)
			if err != nil {
				return fmt.Errorf("insert chunk %d: %w", c.Position, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		// Compute TF-IDF connections for the new chunks against existing ones.
		w.computeConnectionsForDoc(kbID)

		result = &KBAddResult{
			ID:         kbID,
			Path:       relPath,
			SourceType: srcType,
			ChunkCount: len(chunks),
			PageCount:  pageCount,
		}
		return nil
	})

	return result, err
}

// KBListFilter holds filter options for listing KB documents.
type KBListFilter struct {
	SourceType string
}

// KBList returns KB documents matching the filter.
func (w *Workspace) KBList(filter KBListFilter) ([]KBDoc, error) {
	query := `SELECT id, slug, title, source_type, path,
		clean_path, note, page_count, content_hash, added_at,
		COALESCE(added_by, '')
		FROM kb_docs WHERE 1=1`
	var args []any

	if filter.SourceType != "" {
		query += " AND source_type = ?"
		args = append(args, filter.SourceType)
	}

	query += " ORDER BY id ASC"

	rows, err := w.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list kb: %w", err)
	}
	defer rows.Close()

	var docs []KBDoc
	for rows.Next() {
		var d KBDoc
		var cleanPath, note sql.NullString
		var pageCount sql.NullInt64
		if err := rows.Scan(&d.ID, &d.Slug, &d.Title, &d.SourceType,
			&d.Path, &cleanPath, &note, &pageCount,
			&d.ContentHash, &d.AddedAt, &d.AddedBy); err != nil {
			return nil, err
		}
		if cleanPath.Valid {
			d.CleanPath = &cleanPath.String
		}
		if note.Valid {
			d.Note = &note.String
		}
		if pageCount.Valid {
			pc := int(pageCount.Int64)
			d.PageCount = &pc
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// KBReadResult is the response from kb read.
type KBReadResult struct {
	Doc    KBDoc     `json:"doc"`
	Chunks []KBChunk `json:"chunks,omitempty"` // all chunks, or single chunk
}

// KBRead reads a KB document by ID, optionally returning a single chunk.
func (w *Workspace) KBRead(kbID string, chunkPos *int) (*KBReadResult, error) {
	doc, err := w.getKBDoc(kbID)
	if err != nil {
		return nil, err
	}

	result := &KBReadResult{Doc: *doc}

	if chunkPos != nil {
		// Return a single chunk.
		chunk, err := w.getKBChunk(kbID, *chunkPos)
		if err != nil {
			return nil, err
		}
		result.Chunks = []KBChunk{*chunk}
	} else {
		// Return all chunks.
		chunks, err := w.listKBChunks(kbID)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
	}

	return result, nil
}

// KBSearchResult represents a single KB search hit with chunk detail.
type KBSearchResult struct {
	DocID         string  `json:"doc_id"`
	DocTitle      string  `json:"doc_title"`
	ChunkPosition int     `json:"chunk_position"`
	Text          string  `json:"text"`
	Score         float64 `json:"score"`
	MatchType     string  `json:"match_type"`
	Page          *int    `json:"page,omitempty"`
}

// KBSearch performs a KB-specific search returning chunk-level results.
func (w *Workspace) KBSearch(query string, limit int) ([]KBSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	ftsQuery := sanitizeFTS(query)
	if ftsQuery == "" {
		return nil, nil
	}

	// BM25 primary search.
	rows, err := w.DB.Query(`
		SELECT d.id, d.title, k.position, k.text, k.page, bm25(kb_chunks_fts) AS score
		FROM kb_chunks_fts
		JOIN kb_chunks k ON k.id = kb_chunks_fts.rowid
		JOIN kb_docs d ON d.id = k.doc_id
		WHERE kb_chunks_fts MATCH ?
		ORDER BY score
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("kb search: %w", err)
	}
	defer rows.Close()

	var results []KBSearchResult
	for rows.Next() {
		var r KBSearchResult
		var page sql.NullInt64
		if err := rows.Scan(&r.DocID, &r.DocTitle, &r.ChunkPosition, &r.Text, &page, &r.Score); err != nil {
			return nil, err
		}
		r.MatchType = "bm25"
		r.Score = -r.Score // BM25 returns negative; invert
		if page.Valid {
			pg := int(page.Int64)
			r.Page = &pg
		}
		// Truncate text for display.
		if len(r.Text) > 200 {
			r.Text = r.Text[:200] + "..."
		}
		results = append(results, r)
	}

	// Trigram fallback if fewer than 3 BM25 hits.
	if len(results) < 3 {
		trigramResults, err := w.kbTrigramSearch(ftsQuery, limit, results)
		if err == nil {
			results = append(results, trigramResults...)
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// kbTrigramSearch runs a trigram fallback for KB search.
func (w *Workspace) kbTrigramSearch(query string, limit int, existing []KBSearchResult) ([]KBSearchResult, error) {
	// Build seen set from existing results.
	type chunkKey struct {
		docID string
		pos   int
	}
	seen := make(map[chunkKey]bool)
	for _, r := range existing {
		seen[chunkKey{r.DocID, r.ChunkPosition}] = true
	}

	rows, err := w.DB.Query(`
		SELECT ref_id, text
		FROM search_trigram
		WHERE kind = 'kb' AND search_trigram MATCH ?
		LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}

	type hit struct {
		docID string
		text  string
	}
	var hits []hit
	for rows.Next() {
		var h hit
		rows.Scan(&h.docID, &h.text)
		hits = append(hits, h)
	}
	rows.Close()

	var results []KBSearchResult
	for _, h := range hits {
		// Look up the chunk details.
		var title string
		w.DB.QueryRow("SELECT title FROM kb_docs WHERE id = ?", h.docID).Scan(&title)

		text := h.text
		if len(text) > 200 {
			text = text[:200] + "..."
		}

		r := KBSearchResult{
			DocID:     h.docID,
			DocTitle:  title,
			Text:      text,
			MatchType: "trigram",
			Score:     0,
		}
		results = append(results, r)
	}
	return results, nil
}

// KBRemove soft-deletes a KB document, moving it to trash.
func (w *Workspace) KBRemove(kbID string) error {
	return w.WithLock(func() error {
		doc, err := w.getKBDoc(kbID)
		if err != nil {
			return err
		}

		// Read the file content for trash.
		absPath := filepath.Join(w.Root, doc.Path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read file for trash: %w", err)
		}

		now := time.Now().UTC().Format(time.RFC3339)

		// Build metadata snapshot.
		metaBytes, _ := json.Marshal(map[string]string{
			"id": doc.ID, "title": doc.Title, "source_type": doc.SourceType, "path": doc.Path,
		})
		metadata := string(metaBytes)

		tx, err := w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Insert into trash.
		_, err = tx.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
			VALUES ('kb', ?, ?, ?, ?, ?, 'cli')`,
			doc.ID, doc.Path, content, metadata, now)
		if err != nil {
			return fmt.Errorf("insert trash: %w", err)
		}

		// Delete from kb_docs (cascades to chunks, citations, connections).
		_, err = tx.Exec("DELETE FROM kb_docs WHERE id = ?", kbID)
		if err != nil {
			return fmt.Errorf("delete kb_docs: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		// Remove the file from disk.
		os.Remove(absPath)

		// Remove clean sidecar if it exists.
		if doc.CleanPath != nil {
			os.Remove(filepath.Join(w.Root, *doc.CleanPath))
		}

		return nil
	})
}

// getKBDoc reads a single KB doc row by ID.
func (w *Workspace) getKBDoc(kbID string) (*KBDoc, error) {
	d := &KBDoc{}
	var cleanPath, note sql.NullString
	var pageCount sql.NullInt64

	err := w.DB.QueryRow(`SELECT id, slug, title, source_type, path,
		clean_path, note, page_count, content_hash, added_at,
		COALESCE(added_by, '')
		FROM kb_docs WHERE id = ?`, kbID).Scan(
		&d.ID, &d.Slug, &d.Title, &d.SourceType, &d.Path,
		&cleanPath, &note, &pageCount,
		&d.ContentHash, &d.AddedAt, &d.AddedBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("KB document %s not found", kbID)
	}
	if err != nil {
		return nil, fmt.Errorf("read kb doc: %w", err)
	}

	if cleanPath.Valid {
		d.CleanPath = &cleanPath.String
	}
	if note.Valid {
		d.Note = &note.String
	}
	if pageCount.Valid {
		pc := int(pageCount.Int64)
		d.PageCount = &pc
	}
	return d, nil
}

// getKBChunk reads a single chunk by doc ID and position.
func (w *Workspace) getKBChunk(kbID string, position int) (*KBChunk, error) {
	c := &KBChunk{}
	var page sql.NullInt64

	err := w.DB.QueryRow(`SELECT id, doc_id, position, text, char_start, char_end, page
		FROM kb_chunks WHERE doc_id = ? AND position = ?`, kbID, position).Scan(
		&c.ID, &c.DocID, &c.Position, &c.Text, &c.CharStart, &c.CharEnd, &page)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chunk %d not found in %s", position, kbID)
	}
	if err != nil {
		return nil, fmt.Errorf("read chunk: %w", err)
	}
	if page.Valid {
		pg := int(page.Int64)
		c.Page = &pg
	}
	return c, nil
}

// listKBChunks returns all chunks for a KB document ordered by position.
func (w *Workspace) listKBChunks(kbID string) ([]KBChunk, error) {
	rows, err := w.DB.Query(`SELECT id, doc_id, position, text, char_start, char_end, page
		FROM kb_chunks WHERE doc_id = ? ORDER BY position ASC`, kbID)
	if err != nil {
		return nil, fmt.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var chunks []KBChunk
	for rows.Next() {
		var c KBChunk
		var page sql.NullInt64
		if err := rows.Scan(&c.ID, &c.DocID, &c.Position, &c.Text, &c.CharStart, &c.CharEnd, &page); err != nil {
			return nil, err
		}
		if page.Valid {
			pg := int(page.Int64)
			c.Page = &pg
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

// resolveSource resolves a source path or URL. For URLs, it fetches the
// content to a temp file. Returns the local path, detected source type,
// and an optional cleanup function.
func resolveSource(source string) (path string, srcType string, cleanup func(), err error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return fetchURL(source)
	}

	// Local file.
	absPath, err := filepath.Abs(source)
	if err != nil {
		return "", "", nil, fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", "", nil, fmt.Errorf("source file not found: %s", source)
	}

	srcType = detectSourceType(absPath)
	return absPath, srcType, nil, nil
}

// fetchURL downloads a URL to a temp file and returns its path and type.
func fetchURL(url string) (path string, srcType string, cleanup func(), err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", nil, fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", nil, fmt.Errorf("fetch URL: HTTP %d", resp.StatusCode)
	}

	// Detect type from Content-Type header.
	ct := resp.Header.Get("Content-Type")
	srcType = typeFromContentType(ct)

	// Determine extension.
	ext := extForType(srcType)

	tmpFile, err := os.CreateTemp("", "specd-kb-*"+ext)
	if err != nil {
		return "", "", nil, fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", "", nil, fmt.Errorf("download: %w", err)
	}
	tmpFile.Close()

	cleanup = func() { os.Remove(tmpFile.Name()) }
	return tmpFile.Name(), srcType, cleanup, nil
}

// detectSourceType determines the KB source type from a file extension.
func detectSourceType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown":
		return "md"
	case ".html", ".htm":
		return "html"
	case ".pdf":
		return "pdf"
	default:
		return "txt"
	}
}

// typeFromContentType maps HTTP Content-Type to a KB source type.
func typeFromContentType(ct string) string {
	ct = strings.ToLower(ct)
	switch {
	case strings.Contains(ct, "text/html"):
		return "html"
	case strings.Contains(ct, "application/pdf"):
		return "pdf"
	case strings.Contains(ct, "text/markdown"):
		return "md"
	default:
		return "txt"
	}
}

// extForType returns the file extension for a KB source type.
func extForType(srcType string) string {
	switch srcType {
	case "md":
		return ".md"
	case "html":
		return ".html"
	case "pdf":
		return ".pdf"
	default:
		return ".txt"
	}
}

// titleFromFilename derives a human-readable title from a filename.
func titleFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	// Replace common separators with spaces.
	name = strings.NewReplacer("-", " ", "_", " ").Replace(name)
	return strings.TrimSpace(name)
}

// nilString returns nil if the pointer is nil, otherwise the string value.
func nilString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// notePtr returns a pointer to s if non-empty, nil otherwise.
func notePtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- Chunk connections ---

const (
	defaultConnectionThreshold = 0.3
	defaultConnectionTopK      = 10
)

// ChunkConnectionResult represents a connected chunk returned by KBConnections.
type ChunkConnectionResult struct {
	FromChunkID int     `json:"from_chunk_id"`
	ToChunkID   int     `json:"to_chunk_id"`
	ToDocID     string  `json:"to_doc_id"`
	ToDocTitle  string  `json:"to_doc_title"`
	ToPosition  int     `json:"to_position"`
	ToText      string  `json:"to_text"`
	Strength    float64 `json:"strength"`
	Method      string  `json:"method"`
	ToPage      *int    `json:"to_page,omitempty"`
}

// KBConnections returns chunk-to-chunk connections for a KB document.
// If chunkPos is non-nil, returns connections for that specific chunk only.
func (w *Workspace) KBConnections(kbID string, chunkPos *int, limit int) ([]ChunkConnectionResult, error) {
	if limit <= 0 {
		limit = 20
	}

	// Verify doc exists.
	if _, err := w.getKBDoc(kbID); err != nil {
		return nil, err
	}

	var query string
	var args []any

	if chunkPos != nil {
		// Connections for a specific chunk.
		query = `SELECT cc.from_chunk_id, cc.to_chunk_id, d.id, d.title,
				k.position, k.text, cc.strength, cc.method, k.page
			FROM chunk_connections cc
			JOIN kb_chunks fk ON fk.id = cc.from_chunk_id
			JOIN kb_chunks k ON k.id = cc.to_chunk_id
			JOIN kb_docs d ON d.id = k.doc_id
			WHERE fk.doc_id = ? AND fk.position = ?
			ORDER BY cc.strength DESC
			LIMIT ?`
		args = []any{kbID, *chunkPos, limit}
	} else {
		// All connections for all chunks in this doc.
		query = `SELECT cc.from_chunk_id, cc.to_chunk_id, d.id, d.title,
				k.position, k.text, cc.strength, cc.method, k.page
			FROM chunk_connections cc
			JOIN kb_chunks fk ON fk.id = cc.from_chunk_id
			JOIN kb_chunks k ON k.id = cc.to_chunk_id
			JOIN kb_docs d ON d.id = k.doc_id
			WHERE fk.doc_id = ?
			ORDER BY cc.strength DESC
			LIMIT ?`
		args = []any{kbID, limit}
	}

	rows, err := w.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var results []ChunkConnectionResult
	for rows.Next() {
		var r ChunkConnectionResult
		var page sql.NullInt64
		if err := rows.Scan(&r.FromChunkID, &r.ToChunkID, &r.ToDocID, &r.ToDocTitle,
			&r.ToPosition, &r.ToText, &r.Strength, &r.Method, &page); err != nil {
			return nil, err
		}
		if page.Valid {
			pg := int(page.Int64)
			r.ToPage = &pg
		}
		// Truncate text for display.
		if len(r.ToText) > 200 {
			r.ToText = r.ToText[:200] + "..."
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// KBRebuildConnections recomputes the entire chunk connection graph.
func (w *Workspace) KBRebuildConnections(threshold float64, topK int) (int, error) {
	if threshold <= 0 {
		threshold = defaultConnectionThreshold
	}
	if topK <= 0 {
		topK = defaultConnectionTopK
	}

	var count int

	err := w.WithLock(func() error {
		// Clear existing connections.
		if _, err := w.DB.Exec("DELETE FROM chunk_connections"); err != nil {
			return fmt.Errorf("clear connections: %w", err)
		}

		// Load all chunks.
		allChunks, err := w.loadAllChunkTexts()
		if err != nil {
			return err
		}

		if len(allChunks) < 2 {
			return nil
		}

		// Compute connections treating all chunks as "new" against an empty set.
		// Use buildConnections with all chunks as both new and existing.
		connections := buildConnectionsFull(allChunks, threshold, topK)

		// Insert connections.
		tx, err := w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, conn := range connections {
			_, err := tx.Exec(`INSERT OR IGNORE INTO chunk_connections (from_chunk_id, to_chunk_id, strength, method)
				VALUES (?, ?, ?, 'tfidf_cosine')`,
				conn.fromID, conn.toID, conn.strength)
			if err != nil {
				return fmt.Errorf("insert connection: %w", err)
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		count = len(connections) / 2 // bidirectional, so divide by 2 for unique pairs
		return nil
	})

	return count, err
}

// computeConnectionsForDoc computes TF-IDF connections for a newly added
// document's chunks against all existing chunks. Called after KBAdd.
func (w *Workspace) computeConnectionsForDoc(kbID string) {
	// Load new chunks.
	newChunks := make(map[int]string)
	rows, err := w.DB.Query("SELECT id, text FROM kb_chunks WHERE doc_id = ?", kbID)
	if err != nil {
		return
	}
	for rows.Next() {
		var id int
		var text string
		rows.Scan(&id, &text)
		newChunks[id] = text
	}
	rows.Close()

	if len(newChunks) == 0 {
		return
	}

	// Load existing chunks (excluding the new doc's chunks).
	existingChunks := make(map[int]string)
	rows, err = w.DB.Query("SELECT id, text FROM kb_chunks WHERE doc_id != ?", kbID)
	if err != nil {
		return
	}
	for rows.Next() {
		var id int
		var text string
		rows.Scan(&id, &text)
		existingChunks[id] = text
	}
	rows.Close()

	if len(existingChunks) == 0 {
		return
	}

	connections := buildConnections(newChunks, existingChunks,
		defaultConnectionThreshold, defaultConnectionTopK)

	if len(connections) == 0 {
		return
	}

	// Insert connections.
	tx, err := w.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	for _, conn := range connections {
		tx.Exec(`INSERT OR IGNORE INTO chunk_connections (from_chunk_id, to_chunk_id, strength, method)
			VALUES (?, ?, ?, 'tfidf_cosine')`,
			conn.fromID, conn.toID, conn.strength)
	}
	tx.Commit()
}

// loadAllChunkTexts loads all chunk IDs and texts from the database.
func (w *Workspace) loadAllChunkTexts() (map[int]string, error) {
	rows, err := w.DB.Query("SELECT id, text FROM kb_chunks")
	if err != nil {
		return nil, fmt.Errorf("load chunks: %w", err)
	}
	defer rows.Close()

	chunks := make(map[int]string)
	for rows.Next() {
		var id int
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		chunks[id] = text
	}
	return chunks, rows.Err()
}

// buildConnectionsFull computes connections among all chunks (for rebuild).
// Every chunk is compared against every other chunk.
func buildConnectionsFull(allChunks map[int]string, threshold float64, topK int) []chunkConnection {
	if len(allChunks) < 2 {
		return nil
	}

	// Tokenize all chunks and build IDF.
	allTokens := make(map[int][]string, len(allChunks))
	df := make(map[string]int)

	for id, text := range allChunks {
		tokens := tokenize(text)
		allTokens[id] = tokens
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				seen[t] = true
				df[t]++
			}
		}
	}

	idf := buildIDF(df, len(allChunks))

	// Compute TF-IDF vectors.
	vecs := make(map[int]sparseVec, len(allChunks))
	for id := range allChunks {
		vecs[id] = tfidfVector(allTokens[id], idf)
	}

	// Collect IDs for ordered iteration.
	ids := make([]int, 0, len(allChunks))
	for id := range allChunks {
		ids = append(ids, id)
	}

	// Compute pairwise similarities.
	type perChunk struct {
		candidates []connectionCandidate
	}
	results := make(map[int]*perChunk, len(ids))
	for _, id := range ids {
		results[id] = &perChunk{}
	}

	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := ids[i], ids[j]
			sim := cosineSimilarity(vecs[a], vecs[b])
			if sim >= threshold {
				results[a].candidates = append(results[a].candidates, connectionCandidate{chunkID: b, strength: sim})
				results[b].candidates = append(results[b].candidates, connectionCandidate{chunkID: a, strength: sim})
			}
		}
	}

	// Cap per-chunk at topK and build bidirectional connections.
	var connections []chunkConnection
	seen := make(map[[2]int]bool)

	for id, pc := range results {
		sortCandidates(pc.candidates)
		if len(pc.candidates) > topK {
			pc.candidates = pc.candidates[:topK]
		}
		for _, c := range pc.candidates {
			pair := [2]int{id, c.chunkID}
			if id > c.chunkID {
				pair = [2]int{c.chunkID, id}
			}
			if seen[pair] {
				continue
			}
			seen[pair] = true
			connections = append(connections,
				chunkConnection{fromID: id, toID: c.chunkID, strength: c.strength},
				chunkConnection{fromID: c.chunkID, toID: id, strength: c.strength},
			)
		}
	}

	return connections
}
