// search.go implements hybrid BM25 + trigram search across specs, tasks,
// and KB chunks via FTS5 with trigram fallback.
//
// Search strategy:
//
//  1. BM25 (primary) — FTS5 full-text search with porter stemming. Each word
//     token in the query is individually quoted to avoid FTS5 syntax errors
//     from special characters (slashes, hyphens, etc.). Results are ranked
//     by BM25 relevance (higher score = better match).
//
//  2. Trigram (fallback) — FTS5 trigram tokenizer for fuzzy/substring matching.
//     Activated when BM25 returns fewer than 3 results for a kind, or when
//     the query contains special characters that BM25 would strip. Trigram
//     results have score=0 and are appended after BM25 results.
//
//  3. Deduplication — a seen map prevents the same ID from appearing in both
//     BM25 and trigram results. The excludeID parameter filters out the
//     document being searched for (e.g. the newly created spec).
package cmd

import (
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"unicode"
)

// SearchResult represents a single search hit across specs, tasks, or KB.
type SearchResult struct {
	Kind      string  `json:"kind"`              // "spec", "task", or "kb"
	ID        string  `json:"id"`                // e.g. "SPEC-1", "TASK-3", "KB-7"
	Title     string  `json:"title"`             // display title
	Summary   string  `json:"summary,omitempty"` // one-line summary or chunk preview
	Score     float64 `json:"score"`             // relevance score (higher = better match)
	MatchType string  `json:"match_type"`        // "bm25" or "trigram"
}

// SearchResults holds grouped search results by kind.
type SearchResults struct {
	Specs []SearchResult `json:"specs"`
	Tasks []SearchResult `json:"tasks"`
	KB    []SearchResult `json:"kb"`
}

// wordTokenRe extracts unicode letter/digit sequences from a query string.
// Used by sanitizeBM25 to strip FTS5-unsafe characters while preserving
// meaningful words like "OAuth2" or "UTF8".
var wordTokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// trigramMinLen is the minimum query length for trigram matching.
// The FTS5 trigram tokenizer splits text into 3-character windows,
// so queries shorter than 3 characters cannot produce any trigrams.
const trigramMinLen = 3

// bm25FallbackThreshold — if BM25 returns fewer than this many hits for a
// kind, trigram results are appended as a supplement.
const bm25FallbackThreshold = 3

// buildBM25Queries returns per-kind BM25 SQL with the given weights baked in.
// Each query returns (id, title, summary, score) and accepts (match_query, exclude_id, limit).
//
// FTS column order:
//   - specs_fts / tasks_fts: title, summary, body
//   - kb_chunks_fts: summary, text
func buildBM25Queries(w SearchWeights) map[string]string {
	// Specs and tasks have 3 columns: title, summary, body.
	tsb := fmt.Sprintf("%.1f, %.1f, %.1f", w.Title, w.Summary, w.Body)
	// KB chunks have 2 columns: summary, text (no title in FTS — title comes from docs JOIN).
	st := fmt.Sprintf("%.1f, %.1f", w.Summary, w.Body)

	return map[string]string{
		KindSpec: fmt.Sprintf(`SELECT s.id, s.title, s.summary, bm25(specs_fts, %s) AS score
			FROM specs_fts
			JOIN specs s ON s.rowid = specs_fts.rowid
			WHERE specs_fts MATCH ?
			AND s.id != ?
			ORDER BY score
			LIMIT ?`, tsb),
		KindTask: fmt.Sprintf(`SELECT t.id, t.title, t.summary, bm25(tasks_fts, %s) AS score
			FROM tasks_fts
			JOIN tasks t ON t.rowid = tasks_fts.rowid
			WHERE tasks_fts MATCH ?
			AND t.id != ?
			ORDER BY score
			LIMIT ?`, tsb),
		KindKB: fmt.Sprintf(`SELECT d.id, d.title,
			CASE WHEN k.summary != '' THEN k.summary ELSE substr(k.text, 1, %d) END,
			bm25(kb_chunks_fts, %s) AS score
			FROM kb_chunks_fts
			JOIN kb_chunks k ON k.id = kb_chunks_fts.rowid
			JOIN kb_docs d ON d.id = k.doc_id
			WHERE kb_chunks_fts MATCH ?
			AND d.id != ?
			ORDER BY score
			LIMIT ?`, ChunkPreviewLength, st),
	}
}

// defaultSearchWeights returns the fallback weights from constants,
// used when no project config is available (e.g. in tests).
func defaultSearchWeights() SearchWeights {
	return SearchWeights{
		Title:   BM25WeightTitle,
		Summary: BM25WeightSummary,
		Body:    BM25WeightBody,
	}
}

// Search performs hybrid BM25 + trigram search across the selected kinds.
//
// Parameters:
//   - db: the project's SQLite database connection
//   - query: the raw search text (title + summary of the new spec, typically)
//   - kind: "spec", "task", "kb", or "all" to search all three
//   - limit: max results per kind (falls back to TopSearchResults if <= 0)
//   - excludeID: ID to exclude from results (e.g. the spec just created)
func Search(db *sql.DB, query, kind string, limit int, excludeID string, weights SearchWeights) (*SearchResults, error) {
	if limit <= 0 {
		limit = TopSearchResults
	}
	if kind == "" {
		kind = KindAll
	}

	// Build BM25 SQL with the project-configured weights.
	queries := buildBM25Queries(weights)

	// Prepare both query forms from the raw input.
	bm25Query := sanitizeBM25(query)
	trigramQuery := sanitizeTrigram(query)
	hasSpecial := queryHasSpecialChars(query)

	// Both sanitizers returned empty — nothing to search for.
	if bm25Query == "" && trigramQuery == "" {
		return &SearchResults{
			Specs: []SearchResult{},
			Tasks: []SearchResult{},
			KB:    []SearchResult{},
		}, nil
	}

	results := &SearchResults{
		Specs: []SearchResult{},
		Tasks: []SearchResult{},
		KB:    []SearchResult{},
	}

	slog.Debug("search", "kind", kind, "limit", limit, "bm25_query", bm25Query, "trigram_query", trigramQuery) //nolint:gosec // queries are sanitized via sanitizeBM25/sanitizeTrigram before reaching this point

	// Search each requested kind using the shared searchByKind function.
	if kind == KindAll || kind == KindSpec {
		specs, err := searchByKind(db, KindSpec, bm25Query, trigramQuery, hasSpecial, limit, excludeID, queries)
		if err != nil {
			return nil, fmt.Errorf("search specs: %w", err)
		}
		results.Specs = specs
	}

	if kind == KindAll || kind == KindTask {
		tasks, err := searchByKind(db, KindTask, bm25Query, trigramQuery, hasSpecial, limit, excludeID, queries)
		if err != nil {
			return nil, fmt.Errorf("search tasks: %w", err)
		}
		results.Tasks = tasks
	}

	if kind == KindAll || kind == KindKB {
		kb, err := searchByKind(db, KindKB, bm25Query, trigramQuery, hasSpecial, limit, excludeID, queries)
		if err != nil {
			return nil, fmt.Errorf("search kb: %w", err)
		}
		results.KB = kb
	}

	return results, nil
}

// searchByKind runs BM25 search for a given kind, then supplements with
// trigram if fewer than bm25FallbackThreshold results were found or the
// query has special characters. This is the single implementation for all
// three kinds — the per-kind SQL is looked up from bm25Queries.
func searchByKind(db *sql.DB, kind, bm25Query, trigramQuery string, forceTrigramToo bool, limit int, excludeID string, queries map[string]string) ([]SearchResult, error) {
	var results []SearchResult
	seen := map[string]bool{}

	// BM25 primary search.
	if bm25Query != "" {
		bm25Results, err := bm25Search(db, kind, bm25Query, limit, excludeID, queries)
		if err != nil {
			return nil, fmt.Errorf("bm25 %s: %w", kind, err)
		}
		for _, r := range bm25Results {
			results = append(results, r)
			seen[r.ID] = true
		}
	}

	// Trigram fallback — runs when BM25 returned too few hits or when the
	// query contained special characters that BM25 would have stripped.
	if (len(results) < bm25FallbackThreshold || forceTrigramToo) && trigramQuery != "" {
		trigram, err := trigramSearch(db, kind, trigramQuery, limit, seen, excludeID)
		if err != nil {
			return nil, fmt.Errorf("trigram %s: %w", kind, err)
		}
		results = append(results, trigram...)
	}

	// Cap at the requested limit.
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// bm25Search runs the BM25 FTS5 query for a given kind. The SQL is looked
// up from bm25Queries. All three kinds return (id, title, summary, score).
// Errors are returned, not swallowed.
func bm25Search(db *sql.DB, kind, query string, limit int, excludeID string, queries map[string]string) ([]SearchResult, error) {
	querySQL, ok := queries[kind]
	if !ok {
		return nil, fmt.Errorf("unknown search kind: %s", kind)
	}

	rows, err := db.Query(querySQL, query, excludeID, limit)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Summary, &r.Score); err != nil {
			return nil, fmt.Errorf("scanning %s row: %w", kind, err)
		}
		r.Kind = kind
		r.MatchType = "bm25"
		// bm25() returns negative values (more negative = more relevant).
		// Flip sign so higher = better for consumers.
		r.Score = -r.Score
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating %s rows: %w", kind, err)
	}
	return results, nil
}

// trigramSearch runs a trigram substring match against the search_trigram
// virtual table as a fallback when BM25 returns insufficient results.
//
// Important: this function collects all rows into a slice before making
// additional queries. Single-connection SQLite would deadlock if we tried
// to query while a rows cursor is still open.
func trigramSearch(db *sql.DB, kind, query string, limit int, seen map[string]bool, excludeID string) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT ref_id, text
		FROM search_trigram
		WHERE kind = ? AND search_trigram MATCH ?
		LIMIT ?`, kind, query, limit)
	if err != nil {
		return nil, fmt.Errorf("trigram query: %w", err)
	}

	// Collect all hits first to release the rows cursor before making
	// follow-up queries to fetch title/summary from the base tables.
	type hit struct {
		id   string
		text string
	}
	var hits []hit
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.id, &h.text); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scanning trigram row: %w", err)
		}
		hits = append(hits, h)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating trigram rows: %w", err)
	}

	var results []SearchResult
	for _, h := range hits {
		if seen[h.id] || h.id == excludeID {
			continue
		}
		seen[h.id] = true

		r := SearchResult{
			Kind:      kind,
			ID:        h.id,
			MatchType: "trigram",
			Score:     0, // trigram tokenizer doesn't produce relevance scores
		}

		if err := fetchTrigramMeta(db, kind, h.id, h.text, &r); err != nil {
			return nil, err
		}

		results = append(results, r)
	}
	return results, nil
}

// sanitizeBM25 prepares a query string for FTS5 BM25 MATCH.
//
// Extracts only alphanumeric word tokens and quotes each one individually:
// "user" "authentication" "OAuth2". This prevents FTS5 syntax errors from
// special characters and eliminates injection via crafted input containing
// FTS5 operators (AND, OR, NOT, NEAR).
//
// Already-quoted queries (starting with ") are passed through for callers
// that construct FTS5 queries programmatically.
func sanitizeBM25(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Pass through already-quoted queries (programmatic callers only).
	if strings.HasPrefix(query, "\"") {
		return query
	}

	// Extract only alphanumeric tokens, stripping everything else.
	tokens := wordTokenRe.FindAllString(query, -1)
	if len(tokens) == 0 {
		return ""
	}

	// Quote each token individually to prevent FTS5 syntax errors.
	quoted := make([]string, len(tokens))
	for i, t := range tokens {
		quoted[i] = `"` + t + `"`
	}
	return strings.Join(quoted, " ")
}

// sanitizeTrigram prepares a query string for FTS5 trigram MATCH.
//
// Wraps the entire query in double quotes for exact phrase/substring
// matching. Returns "" if the query is shorter than 3 characters, since
// the trigram tokenizer cannot produce any trigrams from shorter strings.
func sanitizeTrigram(query string) string {
	query = strings.TrimSpace(query)
	if query == "" || len(query) < trigramMinLen {
		return ""
	}
	// Escape internal double quotes (FTS5 uses "" to represent a literal ").
	escaped := strings.ReplaceAll(query, `"`, `""`)
	return `"` + escaped + `"`
}

// queryHasSpecialChars returns true if the query contains non-alphanumeric,
// non-space characters (hyphens, slashes, dots, etc.). When true, trigram
// search is always included alongside BM25 to catch exact substring matches
// that BM25 would miss after stripping these characters.
func queryHasSpecialChars(query string) bool {
	for _, r := range query {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

// fetchTrigramMeta populates the Title and Summary fields of a trigram
// SearchResult by querying the appropriate base table.
func fetchTrigramMeta(db *sql.DB, kind, id, text string, r *SearchResult) error {
	switch kind {
	case KindSpec:
		return db.QueryRow("SELECT title, summary FROM specs WHERE id = ?", id).Scan(&r.Title, &r.Summary)
	case KindTask:
		return db.QueryRow("SELECT title, summary FROM tasks WHERE id = ?", id).Scan(&r.Title, &r.Summary)
	case KindKB:
		if err := db.QueryRow("SELECT title FROM kb_docs WHERE id = ?", id).Scan(&r.Title); err != nil {
			return err
		}
		if len(text) > ChunkPreviewLength {
			r.Summary = text[:ChunkPreviewLength] + "..."
		} else {
			r.Summary = text
		}
		return nil
	default:
		return fmt.Errorf("unknown search kind: %s", kind)
	}
}
