package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// updateSpecCmd implements `specd update-spec`.
// Used by the AI in step 2 after reviewing the new-spec response.
// Updates the spec type and creates links to related specs and KB chunks.
var updateSpecCmd = &cobra.Command{
	Use:   "update-spec",
	Short: "Update a spec's type and links",
	RunE:  runUpdateSpec,
}

func init() {
	updateSpecCmd.Flags().String("id", "", "spec ID to update (required)")
	updateSpecCmd.Flags().String("type", "", "spec type to set")
	updateSpecCmd.Flags().String("link-specs", "", "comma-separated spec IDs to link")
	updateSpecCmd.Flags().String("link-kb-chunks", "", "comma-separated KB chunk IDs to link")
	_ = updateSpecCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(updateSpecCmd)
}

// UpdateSpecResponse is the JSON output of the update-spec command.
type UpdateSpecResponse struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	LinkedSpecs    []string `json:"linked_specs"`
	LinkedKBChunks []int    `json:"linked_kb_chunks"`
}

func runUpdateSpec(c *cobra.Command, _ []string) error {
	specID, _ := c.Flags().GetString("id")
	specType, _ := c.Flags().GetString("type")
	linkSpecs, _ := c.Flags().GetString("link-specs")
	linkKBChunks, _ := c.Flags().GetString("link-kb-chunks")

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()

	// Update the spec type if provided.
	if specType != "" {
		if err := applySpecType(db, specID, specType, username, now); err != nil {
			return err
		}
	}

	linkedSpecIDs, err := linkRelatedSpecs(db, specID, linkSpecs)
	if err != nil {
		return err
	}

	linkedChunkIDs, err := linkKBChunkCitations(db, specID, linkKBChunks, now)
	if err != nil {
		return err
	}

	// If no type was provided, read the current one from DB for the response.
	if specType == "" {
		_ = db.QueryRow(`SELECT type FROM specs WHERE id = ?`, specID).Scan(&specType)
	}

	return printUpdateResponse(specID, specType, linkedSpecIDs, linkedChunkIDs)
}

// applySpecType updates the spec type in both the database and the spec.md frontmatter.
func applySpecType(db *sql.DB, specID, specType, username, now string) error {
	_, err := db.Exec(`UPDATE specs SET type = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		specType, username, now, specID)
	if err != nil {
		return fmt.Errorf("updating spec type: %w", err)
	}

	if err := updateSpecFileType(db, specID, specType, now); err != nil {
		return fmt.Errorf("updating spec file: %w", err)
	}

	return nil
}

// linkRelatedSpecs creates bidirectional spec_links entries for each
// comma-separated spec ID. Self-links are silently skipped.
func linkRelatedSpecs(db *sql.DB, specID, linkSpecs string) ([]string, error) {
	if linkSpecs == "" {
		return []string{}, nil
	}

	var linked []string
	for _, toSpec := range strings.Split(linkSpecs, ",") {
		toSpec = strings.TrimSpace(toSpec)
		if toSpec == "" || toSpec == specID {
			continue
		}
		// Insert both directions for the undirected link.
		if _, err := db.Exec(`INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)`, specID, toSpec); err != nil {
			return nil, fmt.Errorf("linking specs: %w", err)
		}
		if _, err := db.Exec(`INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)`, toSpec, specID); err != nil {
			return nil, fmt.Errorf("linking specs (reverse): %w", err)
		}
		linked = append(linked, toSpec)
	}

	return linked, nil
}

// linkKBChunkCitations creates citation entries linking the spec to KB chunks.
// Each chunk ID is resolved to its doc_id and position before insertion.
func linkKBChunkCitations(db *sql.DB, specID, linkKBChunks, now string) ([]int, error) {
	if linkKBChunks == "" {
		return []int{}, nil
	}

	var linked []int
	for _, chunkStr := range strings.Split(linkKBChunks, ",") {
		chunkStr = strings.TrimSpace(chunkStr)
		if chunkStr == "" {
			continue
		}
		chunkID, err := strconv.Atoi(chunkStr)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk ID %q: %w", chunkStr, err)
		}

		var docID string
		var pos int
		err = db.QueryRow(`SELECT doc_id, position FROM kb_chunks WHERE id = ?`, chunkID).Scan(&docID, &pos)
		if err != nil {
			return nil, fmt.Errorf("chunk %d not found: %w", chunkID, err)
		}

		_, err = db.Exec(`INSERT OR IGNORE INTO citations (from_kind, from_id, kb_doc_id, chunk_position, created_at)
			VALUES ('spec', ?, ?, ?, ?)`, specID, docID, pos, now)
		if err != nil {
			return nil, fmt.Errorf("creating citation: %w", err)
		}
		linked = append(linked, chunkID)
	}

	return linked, nil
}

// printUpdateResponse marshals and prints the JSON confirmation.
func printUpdateResponse(specID, specType string, linkedSpecs []string, linkedChunks []int) error {
	// Ensure nil slices serialize as [] not null in JSON.
	if linkedSpecs == nil {
		linkedSpecs = []string{}
	}
	if linkedChunks == nil {
		linkedChunks = []int{}
	}

	resp := UpdateSpecResponse{
		ID:             specID,
		Type:           specType,
		LinkedSpecs:    linkedSpecs,
		LinkedKBChunks: linkedChunks,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// updateSpecFileType reads the spec.md file, updates the type in frontmatter,
// and writes it back.
func updateSpecFileType(db *sql.DB, specID, newType, now string) error {
	var path string
	err := db.QueryRow(`SELECT path FROM specs WHERE id = ?`, specID).Scan(&path)
	if err != nil {
		return fmt.Errorf("reading spec path: %w", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path from DB
	if err != nil {
		return fmt.Errorf("reading spec file: %w", err)
	}

	content := string(data)

	// Replace specific lines in the YAML frontmatter by prefix matching.
	// We avoid a full YAML parse/serialize round-trip to preserve the
	// original formatting, field order, and any comments in the file.
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "type: ") {
			lines[i] = "type: " + newType
		}
		if strings.HasPrefix(line, "updated_at: ") {
			lines[i] = "updated_at: " + now
		}
	}

	updated := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil { //nolint:gosec // spec file is committed to VCS
		return fmt.Errorf("writing spec file: %w", err)
	}

	return nil
}
