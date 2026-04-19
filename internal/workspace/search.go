// Package workspace — search.go implements hybrid BM25 + trigram search
// across specs, tasks, and KB chunks via FTS5 with trigram fallback.
package workspace

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
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

	// Sanitize query for FTS5 BM25 (extract word tokens).
	bm25Query := sanitizeBM25(query)
	// Sanitize query for trigram (quote original for substring match).
	trigramQuery := sanitizeTrigram(query)
	// If the query contains special characters, always include trigram results.
	hasSpecial := queryHasSpecialChars(query)

	if bm25Query == "" && trigramQuery == "" {
		return &SearchResults{}, nil
	}

	results := &SearchResults{}

	if kind == "all" || kind == "spec" {
		specs, err := w.searchSpecs(bm25Query, trigramQuery, hasSpecial, limit)
		if err != nil {
			return nil, fmt.Errorf("search specs: %w", err)
		}
		results.Specs = specs
	}

	if kind == "all" || kind == "task" {
		tasks, err := w.searchTasks(bm25Query, trigramQuery, hasSpecial, limit)
		if err != nil {
			return nil, fmt.Errorf("search tasks: %w", err)
		}
		results.Tasks = tasks
	}

	if kind == "all" || kind == "kb" {
		kb, err := w.searchKB(bm25Query, trigramQuery, hasSpecial, limit)
		if err != nil {
			return nil, fmt.Errorf("search kb: %w", err)
		}
		results.KB = kb
	}

	return results, nil
}

// searchSpecs searches specs using BM25 with trigram fallback.
func (w *Workspace) searchSpecs(bm25Query, trigramQuery string, forceTrigramToo bool, limit int) ([]SearchResult, error) {
	var results []SearchResult
	seen := map[string]bool{}

	// BM25 primary search (only if we extracted word tokens).
	if bm25Query != "" {
		rows, err := w.DB.Query(`
			SELECT s.id, s.title, s.summary, bm25(specs_fts) AS score
			FROM specs_fts
			JOIN specs s ON s.rowid = specs_fts.rowid
			WHERE specs_fts MATCH ?
			ORDER BY score
			LIMIT ?`, bm25Query, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r SearchResult
				rows.Scan(&r.ID, &r.Title, &r.Summary, &r.Score)
				r.Kind = "spec"
				r.MatchType = "bm25"
				r.Score = -r.Score
				results = append(results, r)
				seen[r.ID] = true
			}
		}
	}

	// Trigram fallback if BM25 returned fewer than 3 hits, or if query has special chars.
	if (len(results) < 3 || forceTrigramToo) && trigramQuery != "" {
		trigram, err := w.trigramSearch("spec", trigramQuery, limit, seen)
		if err == nil {
			results = append(results, trigram...)
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// searchTasks searches tasks using BM25 with trigram fallback.
func (w *Workspace) searchTasks(bm25Query, trigramQuery string, forceTrigramToo bool, limit int) ([]SearchResult, error) {
	var results []SearchResult
	seen := map[string]bool{}

	if bm25Query != "" {
		rows, err := w.DB.Query(`
			SELECT t.id, t.title, t.summary, bm25(tasks_fts) AS score
			FROM tasks_fts
			JOIN tasks t ON t.rowid = tasks_fts.rowid
			WHERE tasks_fts MATCH ?
			ORDER BY score
			LIMIT ?`, bm25Query, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r SearchResult
				rows.Scan(&r.ID, &r.Title, &r.Summary, &r.Score)
				r.Kind = "task"
				r.MatchType = "bm25"
				r.Score = -r.Score
				results = append(results, r)
				seen[r.ID] = true
			}
		}
	}

	if (len(results) < 3 || forceTrigramToo) && trigramQuery != "" {
		trigram, err := w.trigramSearch("task", trigramQuery, limit, seen)
		if err == nil {
			results = append(results, trigram...)
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// searchKB searches KB chunks using BM25 with trigram fallback.
func (w *Workspace) searchKB(bm25Query, trigramQuery string, forceTrigramToo bool, limit int) ([]SearchResult, error) {
	var results []SearchResult
	seen := map[string]bool{}

	if bm25Query != "" {
		rows, err := w.DB.Query(`
			SELECT d.id, d.title, k.text, bm25(kb_chunks_fts) AS score
			FROM kb_chunks_fts
			JOIN kb_chunks k ON k.id = kb_chunks_fts.rowid
			JOIN kb_docs d ON d.id = k.doc_id
			WHERE kb_chunks_fts MATCH ?
			ORDER BY score
			LIMIT ?`, bm25Query, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r SearchResult
				var chunkText string
				rows.Scan(&r.ID, &r.Title, &chunkText, &r.Score)
				r.Kind = "kb"
				r.MatchType = "bm25"
				r.Score = -r.Score
				if len(chunkText) > 200 {
					r.Summary = chunkText[:200] + "..."
				} else {
					r.Summary = chunkText
				}
				results = append(results, r)
				seen[r.ID] = true
			}
		}
	}

	if (len(results) < 3 || forceTrigramToo) && trigramQuery != "" {
		trigram, err := w.trigramSearch("kb", trigramQuery, limit, seen)
		if err == nil {
			results = append(results, trigram...)
		}
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

// wordTokenRe extracts alphanumeric word tokens from a query string.
var wordTokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// sanitizeBM25 prepares a query string for FTS5 BM25 MATCH.
// Extracts only alphanumeric word tokens, stripping special characters
// that would cause FTS5 syntax errors (slashes, hyphens, etc.).
func sanitizeBM25(query string) string {
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

	// Extract only alphanumeric tokens to avoid FTS5 syntax errors
	// from special characters like /, -, :, etc.
	tokens := wordTokenRe.FindAllString(query, -1)
	if len(tokens) == 0 {
		return ""
	}

	// Quote each token individually for safety.
	quoted := make([]string, len(tokens))
	for i, t := range tokens {
		quoted[i] = `"` + t + `"`
	}
	return strings.Join(quoted, " ")
}

// sanitizeTrigram prepares a query string for FTS5 trigram MATCH.
// Quotes the original string as a phrase for exact substring matching.
// The trigram tokenizer requires at least 3 characters.
func sanitizeTrigram(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	// Trigram needs at least 3 characters to produce any trigrams.
	if len(query) < 3 {
		return ""
	}
	// Escape internal double quotes and wrap in quotes for phrase matching.
	escaped := strings.ReplaceAll(query, `"`, `""`)
	return `"` + escaped + `"`
}

// queryHasSpecialChars returns true if the query contains non-alphanumeric,
// non-space characters that FTS5 BM25 would strip, meaning the user likely
// intends an exact substring match better served by trigram.
func queryHasSpecialChars(query string) bool {
	for _, r := range query {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
