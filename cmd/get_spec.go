package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// getSpecCmd implements `specd get-spec --id SPEC-1`.
// Returns a single spec by ID as JSON, including linked specs.
var getSpecCmd = &cobra.Command{
	Use:   "get-spec",
	Short: "Get a spec by ID",
	RunE:  runGetSpec,
}

func init() {
	getSpecCmd.Flags().String("id", "", "spec ID (required)")
	_ = getSpecCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(getSpecCmd)
}

// GetSpecResponse is the JSON output of the get-spec command.
type GetSpecResponse struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Summary     string   `json:"summary"`
	Body        string   `json:"body"`
	Path        string   `json:"path"`
	Position    int      `json:"position"`
	LinkedSpecs []string `json:"linked_specs"`
	CreatedBy   string   `json:"created_by,omitempty"`
	UpdatedBy   string   `json:"updated_by,omitempty"`
	ContentHash string   `json:"content_hash"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func runGetSpec(c *cobra.Command, _ []string) error {
	specID, _ := c.Flags().GetString("id")

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Read the spec row.
	var resp GetSpecResponse
	var updatedBy *string
	err = db.QueryRow(`
		SELECT id, slug, title, type, summary, body, path, position,
		       created_by, updated_by, content_hash, created_at, updated_at
		FROM specs WHERE id = ?`, specID).Scan(
		&resp.ID, &resp.Slug, &resp.Title, &resp.Type, &resp.Summary,
		&resp.Body, &resp.Path, &resp.Position,
		&resp.CreatedBy, &updatedBy, &resp.ContentHash,
		&resp.CreatedAt, &resp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("spec %s not found: %w", specID, err)
	}
	if updatedBy != nil {
		resp.UpdatedBy = *updatedBy
	}

	// Read linked spec IDs.
	rows, err := db.Query("SELECT to_spec FROM spec_links WHERE from_spec = ? ORDER BY to_spec", specID)
	if err != nil {
		return fmt.Errorf("reading spec links: %w", err)
	}
	resp.LinkedSpecs = []string{}
	for rows.Next() {
		var toSpec string
		if err := rows.Scan(&toSpec); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning spec link: %w", err)
		}
		resp.LinkedSpecs = append(resp.LinkedSpecs, toSpec)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating spec links: %w", err)
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}
