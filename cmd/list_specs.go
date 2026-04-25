// list_specs.go implements `specd list-specs`. Returns a paginated JSON list
// of all specs in the project, ordered by position then ID. Supports --page
// and --page-size flags for cursor-free pagination.
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// listSpecsCmd implements `specd list-specs`.
// Returns a paginated list of specs ordered by position, then ID.
var listSpecsCmd = &cobra.Command{
	Use:   "list-specs",
	Short: "List specs with pagination",
	RunE:  runListSpecs,
}

func init() {
	listSpecsCmd.Flags().Int("page", 1, "page number (1-based)")
	listSpecsCmd.Flags().Int("page-size", DefaultPageSize, "results per page")
	rootCmd.AddCommand(listSpecsCmd)
}

// ListSpecItem is a single spec in the list response.
type ListSpecItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListSpecsResponse is the JSON output of the list-specs command.
type ListSpecsResponse struct {
	Specs      []ListSpecItem `json:"specs"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
}

func runListSpecs(c *cobra.Command, _ []string) error {
	page, _ := c.Flags().GetInt("page")
	pageSize, _ := c.Flags().GetInt("page-size")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Count total specs.
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM specs").Scan(&total); err != nil {
		return fmt.Errorf("counting specs: %w", err)
	}

	totalPages := (total + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	rows, err := db.Query(`
		SELECT id, title, type, summary, position, created_at, updated_at
		FROM specs
		ORDER BY position, id
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return fmt.Errorf("listing specs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	specs := []ListSpecItem{}
	for rows.Next() {
		var s ListSpecItem
		if err := rows.Scan(&s.ID, &s.Title, &s.Type, &s.Summary, &s.Position, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return fmt.Errorf("scanning spec: %w", err)
		}
		specs = append(specs, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating specs: %w", err)
	}

	resp := ListSpecsResponse{
		Specs:      specs,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
		TotalPages: totalPages,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}
