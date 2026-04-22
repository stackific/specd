package cmd

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestSearchClaimsFTS verifies that claims are searchable via FTS and
// the exclude parameter filters out the querying spec's own claims.
func TestSearchClaimsFTS(t *testing.T) {
	tmp := t.TempDir()
	specTypes := []string{"business", "functional"}
	taskStages := []string{"backlog", "done"}
	if err := InitDB(tmp, specTypes, taskStages); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}

	// Insert two specs.
	mustExec(t, db, `INSERT INTO specs (id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'auth', 'Authentication', 'business', 'Login flow', 'Body', 'p1', 'h1', '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO specs (id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-2', 'sessions', 'Sessions', 'business', 'Token mgmt', 'Body', 'p2', 'h2', '2025-01-01', '2025-01-01')`)

	// Insert claims for SPEC-1.
	mustExec(t, db, `INSERT INTO spec_claims (spec_id, position, text) VALUES ('SPEC-1', 1, 'The system must authenticate users via OAuth2')`)
	mustExec(t, db, `INSERT INTO spec_claims (spec_id, position, text) VALUES ('SPEC-1', 2, 'The system must invalidate sessions on password change')`)

	// Insert a potentially conflicting claim for SPEC-2.
	mustExec(t, db, `INSERT INTO spec_claims (spec_id, position, text) VALUES ('SPEC-2', 1, 'The system should keep sessions alive across password changes')`)

	// Search for "sessions password change" excluding SPEC-2 — should find SPEC-1's claim.
	results, err := searchClaimsFTS(db, "sessions password change", "SPEC-2", 10)
	if err != nil {
		t.Fatalf("searchClaimsFTS: %v", err)
	}

	found := false
	for _, r := range results {
		if r.SpecID == "SPEC-1" {
			found = true
		}
		if r.SpecID == "SPEC-2" {
			t.Error("SPEC-2 should be excluded from results")
		}
	}
	if !found {
		t.Error("expected SPEC-1 claim in search results")
	}
}

// TestSearchClaimsFTSEmpty verifies empty results when no claims match.
func TestSearchClaimsFTSEmpty(t *testing.T) {
	tmp := t.TempDir()
	if err := InitDB(tmp, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	results, err := searchClaimsFTS(db, "nonexistent query", "", 10)
	if err != nil {
		t.Fatalf("searchClaimsFTS: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

// TestSearchClaimsNegativeCase verifies that unrelated claims don't match.
func TestSearchClaimsNegativeCase(t *testing.T) {
	tmp := t.TempDir()
	if err := InitDB(tmp, []string{"business"}, []string{"backlog", "done"}); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}

	// Auth spec with auth claims.
	mustExec(t, db, `INSERT INTO specs (id, slug, title, type, summary, body, path, content_hash, created_at, updated_at)
		VALUES ('SPEC-1', 'auth', 'Auth', 'business', 'Login', 'Body', 'p1', 'h1', '2025-01-01', '2025-01-01')`)
	mustExec(t, db, `INSERT INTO spec_claims (spec_id, position, text) VALUES ('SPEC-1', 1, 'The system must authenticate users via OAuth2')`)

	// Search for invoice-related terms — should NOT match auth claims.
	results, err := searchClaimsFTS(db, "invoice PDF billing tax calculation", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for unrelated query, got %d", len(results))
	}
}
