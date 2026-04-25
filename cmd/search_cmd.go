package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// searchCmd implements `specd search --query "..." --kind spec|task|kb|all`.
// Returns matching results ranked by relevance using hybrid BM25 + trigram.
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search specs, tasks, and KB by text",
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().String("query", "", "search terms (required)")
	searchCmd.Flags().String("kind", KindAll, "what to search: spec, task, kb, or all")
	searchCmd.Flags().Int("limit", 0, "max results per kind (default: from project config)")
	_ = searchCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(c *cobra.Command, _ []string) error {
	query, _ := c.Flags().GetString("query")
	kind, _ := c.Flags().GetString("kind")
	limit, _ := c.Flags().GetInt("limit")

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Read search config from project.
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

	weights := proj.SearchWeights
	if weights.Title == 0 && weights.Summary == 0 && weights.Body == 0 {
		weights = defaultSearchWeights()
	}

	results, err := Search(db, query, kind, limit, "", weights)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling results: %w", err)
	}
	fmt.Println(string(out))

	return nil
}
