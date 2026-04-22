package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// searchClaimsCmd implements `specd search-claims --query "..." --exclude SPEC-1`.
// Searches the spec_claims_fts index for matching acceptance criteria claims
// across all specs. Used to find potential contradictions.
var searchClaimsCmd = &cobra.Command{
	Use:   "search-claims",
	Short: "Search acceptance criteria claims across all specs",
	RunE:  runSearchClaims,
}

func init() {
	searchClaimsCmd.Flags().String("query", "", "search terms (required)")
	searchClaimsCmd.Flags().String("exclude", "", "spec ID to exclude from results")
	searchClaimsCmd.Flags().Int("limit", 0, "max results (default: from project config)")
	_ = searchClaimsCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(searchClaimsCmd)
}

// ClaimSearchResult holds a single claim match with its spec context.
type ClaimSearchResult struct {
	SpecID    string `json:"spec_id"`
	Title     string `json:"spec_title"`
	Claim     string `json:"claim"`
	MatchType string `json:"match_type"`
}

func runSearchClaims(c *cobra.Command, _ []string) error {
	query, _ := c.Flags().GetString("query")
	excludeID, _ := c.Flags().GetString("exclude")
	limit, _ := c.Flags().GetInt("limit")

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		return fmt.Errorf("cannot read project config")
	}

	if limit <= 0 {
		limit = proj.TopSearchResults
		if limit <= 0 {
			limit = TopSearchResults
		}
	}

	results, err := searchClaimsFTS(db, query, excludeID, limit)
	if err != nil {
		return fmt.Errorf("searching claims: %w", err)
	}

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling results: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// searchClaimsFTS searches the spec_claims_fts index for matching claims,
// excluding claims belonging to excludeID. Returns claim text with spec context.
func searchClaimsFTS(db *sql.DB, queryText, excludeID string, limit int) ([]ClaimSearchResult, error) {
	bm25Query := sanitizeBM25(queryText)
	if bm25Query == "" {
		return []ClaimSearchResult{}, nil
	}

	rows, err := db.Query(`
		SELECT c.spec_id, s.title, c.text
		FROM spec_claims_fts f
		JOIN spec_claims c ON c.rowid = f.rowid
		JOIN specs s ON s.id = c.spec_id
		WHERE spec_claims_fts MATCH ?
		AND c.spec_id != ?
		ORDER BY bm25(spec_claims_fts)
		LIMIT ?
	`, bm25Query, excludeID, limit)
	if err != nil {
		return []ClaimSearchResult{}, nil //nolint:nilerr // no matches is fine
	}
	defer func() { _ = rows.Close() }()

	var results []ClaimSearchResult
	for rows.Next() {
		var r ClaimSearchResult
		if err := rows.Scan(&r.SpecID, &r.Title, &r.Claim); err != nil {
			return nil, fmt.Errorf("scanning claim: %w", err)
		}
		r.MatchType = "bm25"
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating claims: %w", err)
	}

	if results == nil {
		results = []ClaimSearchResult{}
	}
	return results, nil
}
