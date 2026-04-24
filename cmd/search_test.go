package cmd

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// testWeights returns the default BM25 weights for use in tests.
var testWeights = defaultSearchWeights()

// Helper: creates a DB with schema and inserts test specs + a KB doc/chunk.
func setupSearchDB(t *testing.T) *sql.DB {
	t.Helper()
	tmp := t.TempDir()
	specTypes := []string{"business", "functional"}
	taskStages := []string{"backlog", "todo", "in_progress", "done"}
	if err := InitDB(tmp, specTypes, taskStages); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}

	// Auth-related specs.
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'User Authentication', 'business', 'OAuth2 login flow', 'Implement OAuth2 authentication with session tokens', 'p1', 'h1', '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-2', 'Session Management', 'business', 'Authentication session tokens', 'Handle authentication sessions and token refresh', 'p2', 'h2', '2025-01-01', '2025-01-01')`)

	// Unrelated spec.
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-3', 'Invoice Generation', 'business', 'PDF invoices from billing', 'Generate PDF invoices with tax calculations', 'p3', 'h3', '2025-01-01', '2025-01-01')`)

	// Task linked to SPEC-1.
	mustExec(t, db, `INSERT INTO tasks (id, spec_id, title, status, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('TASK-1', 'SPEC-1', 'Implement OAuth2 Provider', 'backlog', 'Add OAuth2 authentication provider', 'Implement the OAuth2 login flow', 'tp1', 'th1', '2025-01-01', '2025-01-01')`)

	// Unrelated task.
	mustExec(t, db, `INSERT INTO tasks (id, spec_id, title, status, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('TASK-2', 'SPEC-3', 'Fix CSS Layout', 'todo', 'Fix broken grid layout', 'The grid layout breaks on mobile', 'tp2', 'th2', '2025-01-01', '2025-01-01')`)

	// KB doc + chunk about authentication.
	mustExec(t, db, `INSERT INTO kb_docs (id, title, source_type, path, content_hash, added_at)
		VALUES ('KB-1', 'OAuth2 Implementation Guide', 'md', 'docs/oauth.md', 'kh1', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO kb_chunks (id, doc_id, position, text, char_start, char_end)
		VALUES (1, 'KB-1', 0, 'OAuth2 authentication requires redirect URIs and token exchange', 0, 100)`)

	return db
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("mustExec: %v\nquery: %s", err, query)
	}
}

// --- sanitizeBM25 ---

func TestSanitizeBM25(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Normal words are individually quoted.
		{"user authentication", `"user" "authentication"`},
		// Special chars are stripped, tokens extracted.
		{"hello*world", `"hello" "world"`},
		{"path/to/file", `"path" "to" "file"`},
		// Empty input.
		{"", ""},
		{"   ", ""},
		// Already-quoted input passes through.
		{`"exact phrase"`, `"exact phrase"`},
		// FTS5 operators in user input are NOT passed through (injection fix).
		// "Supply AND Demand" is treated as three words, not an operator.
		{"Supply AND Demand", `"Supply" "AND" "Demand"`},
	}
	for _, tt := range tests {
		got := sanitizeBM25(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeBM25(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- sanitizeTrigram ---

func TestSanitizeTrigram(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"auth", `"auth"`},
		{"ab", ""},                         // too short for trigram
		{"", ""},                           // empty
		{`say "hello"`, `"say ""hello"""`}, // internal quotes escaped
		{"path/to/file", `"path/to/file"`}, // special chars preserved
	}
	for _, tt := range tests {
		got := sanitizeTrigram(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeTrigram(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- queryHasSpecialChars ---

func TestQueryHasSpecialChars(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", false},
		{"hello-world", true},
		{"path/to/file", true},
		{"clean query 123", false},
	}
	for _, tt := range tests {
		got := queryHasSpecialChars(tt.input)
		if got != tt.want {
			t.Errorf("queryHasSpecialChars(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Search: empty DB ---

func TestSearchEmpty(t *testing.T) {
	tmp := t.TempDir()
	if err := InitDB(tmp, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	results, err := Search(db, "nonexistent query", "all", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Specs) != 0 || len(results.Tasks) != 0 || len(results.KB) != 0 {
		t.Errorf("expected empty results, got specs=%d tasks=%d kb=%d",
			len(results.Specs), len(results.Tasks), len(results.KB))
	}
}

// --- Search: BM25 finds related specs, excludes self, scores populated ---

func TestSearchBM25FindsRelatedSpecs(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Search for auth content, excluding SPEC-1.
	results, err := Search(db, "authentication session tokens", "spec", 5, "SPEC-1", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// SPEC-2 (sessions) should appear — it contains "authentication" and "session".
	found := false
	for _, r := range results.Specs {
		if r.ID == "SPEC-1" {
			t.Error("excluded spec SPEC-1 should not appear in results")
		}
		if r.ID == "SPEC-2" {
			found = true
			if r.Score <= 0 {
				t.Error("expected positive BM25 score")
			}
			if r.MatchType != "bm25" {
				t.Errorf("expected match_type=bm25, got %q", r.MatchType)
			}
			if r.Kind != "spec" {
				t.Errorf("expected kind=spec, got %q", r.Kind)
			}
		}
	}
	if !found {
		t.Error("expected SPEC-2 (Session Management) in BM25 results")
	}
}

// --- Search: negative case, unrelated spec does NOT match ---

func TestSearchNegativeCaseUnrelatedSpec(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Search for invoice/billing content — should NOT find auth specs.
	results, err := Search(db, "invoice PDF billing tax", "spec", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	for _, r := range results.Specs {
		if r.ID == "SPEC-1" || r.ID == "SPEC-2" {
			t.Errorf("auth spec %s should not match invoice query", r.ID)
		}
	}
}

// --- Search: trigram fallback for special char queries ---

func TestSearchTrigramFallbackSpecialChars(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// "OAuth2" as a query — BM25 will find it, but also test that the
	// pipeline doesn't error with special chars in the broader query.
	results, err := Search(db, "OAuth2/authentication", "spec", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// Should find at least one auth spec via BM25 or trigram.
	if len(results.Specs) == 0 {
		t.Error("expected trigram or BM25 to find auth specs for 'OAuth2/authentication'")
	}
}

// --- Search: task search ---

func TestSearchTasks(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	results, err := Search(db, "OAuth2 authentication provider", "task", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// TASK-1 (Implement OAuth2 Provider) should match.
	found := false
	for _, r := range results.Tasks {
		if r.ID == "TASK-1" {
			found = true
			if r.Kind != "task" {
				t.Errorf("expected kind=task, got %q", r.Kind)
			}
		}
		if r.ID == "TASK-2" {
			t.Error("TASK-2 (Fix CSS Layout) should not match auth query")
		}
	}
	if !found {
		t.Error("expected TASK-1 in task search results")
	}
}

// --- Search: KB search ---

func TestSearchKB(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	results, err := Search(db, "OAuth2 authentication redirect", "kb", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// KB-1 chunk about OAuth2 should match.
	found := false
	for _, r := range results.KB {
		if r.ID == "KB-1" {
			found = true
			if r.Kind != "kb" {
				t.Errorf("expected kind=kb, got %q", r.Kind)
			}
		}
	}
	if !found {
		t.Error("expected KB-1 in KB search results")
	}
}

// --- Search: kind filter ---

func TestSearchKindFilter(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Search only specs — tasks and KB should be empty.
	results, err := Search(db, "authentication", "spec", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Tasks) != 0 {
		t.Errorf("expected no task results when kind=spec, got %d", len(results.Tasks))
	}
	if len(results.KB) != 0 {
		t.Errorf("expected no KB results when kind=spec, got %d", len(results.KB))
	}

	// Search all — should have results in multiple kinds.
	allResults, err := Search(db, "authentication OAuth2", "all", 5, "", testWeights)
	if err != nil {
		t.Fatalf("Search all: %v", err)
	}
	if len(allResults.Specs) == 0 {
		t.Error("expected spec results when kind=all")
	}
}

// --- Search: limit parameter ---

func TestSearchLimit(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// With limit=1, should return at most 1 spec.
	results, err := Search(db, "authentication session", "spec", 1, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Specs) > 1 {
		t.Errorf("expected at most 1 result with limit=1, got %d", len(results.Specs))
	}
}

// --- Search: deduplication between BM25 and trigram ---

func TestSearchDeduplication(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// "OAuth2" is present in the spec body and the trigram index.
	// Both BM25 and trigram should find SPEC-1, but it should appear only once.
	results, err := Search(db, "OAuth2/authentication", "spec", 10, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	idCount := map[string]int{}
	for _, r := range results.Specs {
		idCount[r.ID]++
	}
	for id, count := range idCount {
		if count > 1 {
			t.Errorf("spec %s appears %d times — should be deduplicated to 1", id, count)
		}
	}
}

// --- Search: excludeID works in trigram path ---

func TestSearchExcludeIDInTrigram(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Use a special-char query to force trigram, excluding SPEC-1.
	results, err := Search(db, "OAuth2/login", "spec", 10, "SPEC-1", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	for _, r := range results.Specs {
		if r.ID == "SPEC-1" {
			t.Error("SPEC-1 should be excluded from trigram results")
		}
	}
}

// --- Search: trigram-only query (no usable BM25 tokens) ---

func TestSearchTrigramOnly(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Insert a spec with a distinctive substring containing only special chars + digits.
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-4', 'Error Code Handling', 'functional', 'Handle error code E-4072', 'Handle the specific error code E-4072 from the upstream API', 'p4', 'h4', '2025-01-01', '2025-01-01')`)

	// Search for "E-4072" — BM25 tokenizes this into "E" and "4072" which
	// would match many things. Trigram matches the exact substring "E-4072".
	// With special chars present, forceTrigramToo=true guarantees trigram runs.
	results, err := Search(db, "E-4072", "spec", 10, "", testWeights)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	found := false
	for _, r := range results.Specs {
		if r.ID == "SPEC-4" {
			found = true
		}
	}
	if !found {
		t.Error("expected SPEC-4 to be found for 'E-4072' query (via BM25 or trigram)")
	}

	// Now test a true trigram-only case: search for a partial word substring
	// that exists inside a token but isn't a token itself. BM25 quoted token
	// "thenticat" won't match the FTS5 token "authentication", but trigram
	// will find the substring.
	results3, err := Search(db, "thenticat", "spec", 10, "", testWeights)
	if err != nil {
		t.Fatalf("Search trigram-only: %v", err)
	}

	foundTrigram := false
	for _, r := range results3.Specs {
		if r.MatchType == "trigram" {
			foundTrigram = true
		}
	}
	if !foundTrigram {
		t.Error("expected at least one trigram-only match for partial word 'thenticat'")
	}
}

// --- Search: custom weights affect ranking ---

func TestSearchCustomWeightsAffectRanking(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Insert two specs with a unique term "zephyrion" that doesn't appear
	// anywhere else in the test data. One has it in the title, the other in the body.
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-10', 'Zephyrion Module', 'business', 'A new module', 'Implementation details for the system', 'p10', 'h10', '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO specs (id, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-11', 'Backend Refactor', 'business', 'Refactoring plan', 'This refactor introduces the zephyrion module in the codebase', 'p11', 'h11', '2025-01-01', '2025-01-01')`)

	// With high title weight, SPEC-10 (title match) should rank first.
	highTitle := SearchWeights{Title: 100.0, Summary: 1.0, Body: 1.0}
	results, err := Search(db, "zephyrion", KindSpec, 2, "", highTitle)
	if err != nil {
		t.Fatalf("Search high title: %v", err)
	}
	if len(results.Specs) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results.Specs))
	}
	if results.Specs[0].ID != "SPEC-10" {
		t.Errorf("with title weight=100, expected SPEC-10 first, got %s", results.Specs[0].ID)
	}

	// With high body weight, SPEC-11 (body match) should rank first.
	highBody := SearchWeights{Title: 1.0, Summary: 1.0, Body: 100.0}
	results2, err := Search(db, "zephyrion", KindSpec, 2, "", highBody)
	if err != nil {
		t.Fatalf("Search high body: %v", err)
	}
	if len(results2.Specs) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results2.Specs))
	}
	if results2.Specs[0].ID != "SPEC-11" {
		t.Errorf("with body weight=100, expected SPEC-11 first, got %s", results2.Specs[0].ID)
	}
}

// --- Search: zero weights fall back to defaults ---

func TestSearchZeroWeightsFallback(t *testing.T) {
	db := setupSearchDB(t)
	defer func() { _ = db.Close() }()

	// Zero weights should still produce results (defaultSearchWeights is used
	// by callers, but Search itself should not crash with zero weights).
	zeroWeights := SearchWeights{Title: 0, Summary: 0, Body: 0}
	results, err := Search(db, "authentication", KindSpec, 5, "", zeroWeights)
	if err != nil {
		t.Fatalf("Search with zero weights: %v", err)
	}
	// bm25() with all-zero weights returns 0 for everything, but rows still match.
	// This verifies the code path doesn't crash.
	_ = results
}

// --- Config: SearchWeights round-trip ---

func TestSearchWeightsRoundTrip(t *testing.T) {
	tmp := t.TempDir()

	cfg := &ProjectConfig{
		Folder:           "specd",
		SpecTypes:        []string{"business"},
		TaskStages:       []string{"backlog"},
		TopSearchResults: 5,
		SearchWeights: SearchWeights{
			Title:   15.0,
			Summary: 7.5,
			Body:    2.0,
		},
	}

	if err := SaveProjectConfig(tmp, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	loaded, err := LoadProjectConfig(tmp)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil config")
	}
	if loaded.SearchWeights.Title != 15.0 {
		t.Errorf("expected title weight 15.0, got %f", loaded.SearchWeights.Title)
	}
	if loaded.SearchWeights.Summary != 7.5 {
		t.Errorf("expected summary weight 7.5, got %f", loaded.SearchWeights.Summary)
	}
	if loaded.SearchWeights.Body != 2.0 {
		t.Errorf("expected body weight 2.0, got %f", loaded.SearchWeights.Body)
	}
}
