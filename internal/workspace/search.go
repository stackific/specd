package workspace

import (
	"fmt"
	"strings"
)

// SearchResult represents a single search hit.
type SearchResult struct {
	Kind      string  `json:"kind"`       // "spec", "task", or "kb"
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Summary   string  `json:"summary,omitempty"`
	Score     float64 `json:"score"`
	MatchType string  `json:"match_type"` // "bm25" or "trigram"
}

// SearchResults holds grouped search results.
type SearchResults struct {
	Specs []SearchResult `json:"specs,omitempty"`
	Tasks []SearchResult `json:"tasks,omitempty"`
	KB    []SearchResult `json:"kb,omitempty"`
}

// Search performs hybrid BM25 + trigram search across the selected kinds.
// If BM25 returns fewer than 3 hits for a kind, trigram results are appended.
func (w *Workspace) Search(query string, kind string, limit int) (*SearchResults, error) {
	if limit <= 0 {
		limit = 20
	}
	if kind == "" {
		kind = "all"
	}

	// Sanitize query for FTS5: escape double quotes.
	ftsQuery := sanitizeFTS(query)
	if ftsQuery == "" {
		return &SearchResults{}, nil
	}

	results := &SearchResults{}

	if kind == "all" || kind == "spec" {
		specs, err := w.searchSpecs(ftsQuery, limit)
		if err != nil {
			return nil, fmt.Errorf("search specs: %w", err)
		}
		results.Specs = specs
	}

	if kind == "all" || kind == "task" {
		tasks, err := w.searchTasks(ftsQuery, limit)
		if err != nil {
			return nil, fmt.Errorf("search tasks: %w", err)
		}
		results.Tasks = tasks
	}

	if kind == "all" || kind == "kb" {
		kb, err := w.searchKB(ftsQuery, limit)
		if err != nil {
			return nil, fmt.Errorf("search kb: %w", err)
		}
		results.KB = kb
	}

	return results, nil
}

// searchSpecs searches specs using BM25 with trigram fallback.
func (w *Workspace) searchSpecs(ftsQuery string, limit int) ([]SearchResult, error) {
	// BM25 primary search.
	rows, err := w.DB.Query(`
		SELECT s.id, s.title, s.summary, bm25(specs_fts) AS score
		FROM specs_fts
		JOIN specs s ON s.rowid = specs_fts.rowid
		WHERE specs_fts MATCH ?
		ORDER BY score
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	seen := map[string]bool{}
	for rows.Next() {
		var r SearchResult
		rows.Scan(&r.ID, &r.Title, &r.Summary, &r.Score)
		r.Kind = "spec"
		r.MatchType = "bm25"
		r.Score = -r.Score // BM25 returns negative scores; invert for ranking
		results = append(results, r)
		seen[r.ID] = true
	}

	// Trigram fallback if BM25 returned fewer than 3 hits.
	if len(results) < 3 {
		trigram, err := w.trigramSearch("spec", ftsQuery, limit, seen)
		if err != nil {
			return results, nil // don't fail on trigram error
		}
		results = append(results, trigram...)
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// searchTasks searches tasks using BM25 with trigram fallback.
func (w *Workspace) searchTasks(ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := w.DB.Query(`
		SELECT t.id, t.title, t.summary, bm25(tasks_fts) AS score
		FROM tasks_fts
		JOIN tasks t ON t.rowid = tasks_fts.rowid
		WHERE tasks_fts MATCH ?
		ORDER BY score
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	seen := map[string]bool{}
	for rows.Next() {
		var r SearchResult
		rows.Scan(&r.ID, &r.Title, &r.Summary, &r.Score)
		r.Kind = "task"
		r.MatchType = "bm25"
		r.Score = -r.Score
		results = append(results, r)
		seen[r.ID] = true
	}

	if len(results) < 3 {
		trigram, err := w.trigramSearch("task", ftsQuery, limit, seen)
		if err != nil {
			return results, nil
		}
		results = append(results, trigram...)
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// searchKB searches KB chunks using BM25 with trigram fallback.
func (w *Workspace) searchKB(ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := w.DB.Query(`
		SELECT d.id, d.title, k.text, bm25(kb_chunks_fts) AS score
		FROM kb_chunks_fts
		JOIN kb_chunks k ON k.id = kb_chunks_fts.rowid
		JOIN kb_docs d ON d.id = k.doc_id
		WHERE kb_chunks_fts MATCH ?
		ORDER BY score
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	seen := map[string]bool{}
	for rows.Next() {
		var r SearchResult
		var chunkText string
		rows.Scan(&r.ID, &r.Title, &chunkText, &r.Score)
		r.Kind = "kb"
		r.MatchType = "bm25"
		r.Score = -r.Score
		// Truncate chunk text for summary.
		if len(chunkText) > 200 {
			r.Summary = chunkText[:200] + "..."
		} else {
			r.Summary = chunkText
		}
		results = append(results, r)
		seen[r.ID] = true
	}

	if len(results) < 3 {
		trigram, err := w.trigramSearch("kb", ftsQuery, limit, seen)
		if err != nil {
			return results, nil
		}
		results = append(results, trigram...)
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// trigramSearch runs a trigram substring match as a fallback.
// Returns results not already in the seen set.
func (w *Workspace) trigramSearch(kind, query string, limit int, seen map[string]bool) ([]SearchResult, error) {
	rows, err := w.DB.Query(`
		SELECT ref_id, text
		FROM search_trigram
		WHERE kind = ? AND search_trigram MATCH ?
		LIMIT ?`, kind, query, limit)
	if err != nil {
		return nil, err
	}

	// Collect all hits first to release the rows cursor before making
	// additional queries (single-connection SQLite would deadlock otherwise).
	type hit struct {
		id   string
		text string
	}
	var hits []hit
	for rows.Next() {
		var h hit
		rows.Scan(&h.id, &h.text)
		hits = append(hits, h)
	}
	rows.Close()

	var results []SearchResult
	for _, h := range hits {
		if seen[h.id] {
			continue
		}
		seen[h.id] = true

		r := SearchResult{
			Kind:      kind,
			ID:        h.id,
			MatchType: "trigram",
			Score:     0,
		}

		// Fetch title from the appropriate table.
		switch kind {
		case "spec":
			w.DB.QueryRow("SELECT title, summary FROM specs WHERE id = ?", h.id).Scan(&r.Title, &r.Summary)
		case "task":
			w.DB.QueryRow("SELECT title, summary FROM tasks WHERE id = ?", h.id).Scan(&r.Title, &r.Summary)
		case "kb":
			w.DB.QueryRow("SELECT title FROM kb_docs WHERE id = ?", h.id).Scan(&r.Title)
			if len(h.text) > 200 {
				r.Summary = h.text[:200] + "..."
			} else {
				r.Summary = h.text
			}
		}

		results = append(results, r)
	}
	return results, nil
}

// sanitizeFTS prepares a query string for FTS5 MATCH.
// Wraps individual terms with implicit OR by quoting them.
func sanitizeFTS(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// If the query contains FTS5 operators, pass through as-is.
	for _, op := range []string{"AND", "OR", "NOT", "NEAR"} {
		if strings.Contains(query, " "+op+" ") {
			return query
		}
	}

	// If already quoted, pass through.
	if strings.HasPrefix(query, "\"") {
		return query
	}

	// Split into words and join — FTS5 implicitly ANDs unquoted terms.
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}
	return strings.Join(words, " ")
}
