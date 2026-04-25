package cmd

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// updateSpecCmd implements `specd update-spec`.
// Used by the AI after reviewing a spec. Updates type, adds/removes links
// to related specs and KB chunks.
var updateSpecCmd = &cobra.Command{
	Use:   "update-spec",
	Short: "Update a spec's type and links",
	RunE:  runUpdateSpec,
}

func init() {
	updateSpecCmd.Flags().String("id", "", "spec ID to update (required)")
	updateSpecCmd.Flags().String("type", "", "spec type to set")
	updateSpecCmd.Flags().String("link-specs", "", "comma-separated spec IDs to link")
	updateSpecCmd.Flags().String("unlink-specs", "", "comma-separated spec IDs to unlink")
	updateSpecCmd.Flags().String("link-kb-chunks", "", "comma-separated KB chunk IDs to link")
	updateSpecCmd.Flags().String("unlink-kb-chunks", "", "comma-separated KB chunk IDs to unlink")
	_ = updateSpecCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(updateSpecCmd)
}

// LinkedSpecSummary holds an ID and summary for a linked spec,
// so the AI can see what's currently linked without a separate get-spec call.
type LinkedSpecSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// LinkedKBChunkSummary holds an ID and preview for a linked KB chunk.
type LinkedKBChunkSummary struct {
	ChunkID int    `json:"chunk_id"`
	DocID   string `json:"doc_id"`
	Preview string `json:"preview"`
}

// UpdateSpecResponse is the JSON output of the update-spec command.
type UpdateSpecResponse struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	LinkedSpecs    []LinkedSpecSummary    `json:"linked_specs"`
	LinkedKBChunks []LinkedKBChunkSummary `json:"linked_kb_chunks"`
}

func runUpdateSpec(c *cobra.Command, _ []string) error {
	specID, _ := c.Flags().GetString("id")
	specType, _ := c.Flags().GetString("type")
	linkSpecs, _ := c.Flags().GetString("link-specs")
	unlinkSpecs, _ := c.Flags().GetString("unlink-specs")
	linkKBChunks, _ := c.Flags().GetString("link-kb-chunks")
	unlinkKBChunks, _ := c.Flags().GetString("unlink-kb-chunks")

	slog.Info("update-spec", "id", specID, "type", specType,
		"link_specs", linkSpecs, "unlink_specs", unlinkSpecs,
		"link_kb", linkKBChunks, "unlink_kb", unlinkKBChunks)

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()

	if specType != "" {
		if err := applySpecType(db, specID, specType, username, now); err != nil {
			return err
		}
	}

	if err := linkRelatedSpecs(db, specID, linkSpecs); err != nil {
		return err
	}
	if err := unlinkRelatedSpecs(db, specID, unlinkSpecs); err != nil {
		return err
	}

	if err := linkKBChunkCitations(db, specID, linkKBChunks, now); err != nil {
		return err
	}
	if err := unlinkKBChunkCitations(db, specID, unlinkKBChunks); err != nil {
		return err
	}

	// Read the final type from DB if not provided.
	if specType == "" {
		if err := db.QueryRow(`SELECT type FROM specs WHERE id = ?`, specID).Scan(&specType); err != nil {
			return fmt.Errorf("reading spec type: %w", err)
		}
	}

	// Rewrite spec.md from DB state so it stays the ground truth.
	if err := rewriteSpecFile(db, specID); err != nil {
		return fmt.Errorf("rewriting spec file: %w", err)
	}

	// Build response with current linked specs and KB chunks (with summaries).
	linkedSpecSummaries, err := getLinkedSpecSummaries(db, specID)
	if err != nil {
		return err
	}
	linkedChunkSummaries, err := getLinkedKBChunkSummaries(db, specID)
	if err != nil {
		return err
	}

	return printUpdateResponse(specID, specType, linkedSpecSummaries, linkedChunkSummaries)
}

// applySpecType updates the spec type in the database.
// The file is rewritten at the end of runUpdateSpec via rewriteSpecFile.
func applySpecType(db *sql.DB, specID, specType, username, now string) error {
	_, err := db.Exec(`UPDATE specs SET type = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		specType, username, now, specID)
	if err != nil {
		return fmt.Errorf("updating spec type: %w", err)
	}
	return nil
}

// linkRelatedSpecs creates bidirectional spec_links entries for each
// comma-separated spec ID. Self-links are silently skipped.
func linkRelatedSpecs(db *sql.DB, specID, linkSpecs string) error {
	if linkSpecs == "" {
		return nil
	}

	for _, toSpec := range strings.Split(linkSpecs, ",") {
		toSpec = strings.TrimSpace(toSpec)
		if toSpec == "" || toSpec == specID {
			continue
		}
		if _, err := db.Exec(`INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)`, specID, toSpec); err != nil {
			return fmt.Errorf("linking specs: %w", err)
		}
		if _, err := db.Exec(`INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)`, toSpec, specID); err != nil {
			return fmt.Errorf("linking specs (reverse): %w", err)
		}
	}

	return nil
}

// unlinkRelatedSpecs removes bidirectional spec_links for each
// comma-separated spec ID.
func unlinkRelatedSpecs(db *sql.DB, specID, unlinkSpecs string) error {
	if unlinkSpecs == "" {
		return nil
	}

	for _, toSpec := range strings.Split(unlinkSpecs, ",") {
		toSpec = strings.TrimSpace(toSpec)
		if toSpec == "" {
			continue
		}
		if _, err := db.Exec(`DELETE FROM spec_links WHERE from_spec = ? AND to_spec = ?`, specID, toSpec); err != nil {
			return fmt.Errorf("unlinking spec %s: %w", toSpec, err)
		}
		if _, err := db.Exec(`DELETE FROM spec_links WHERE from_spec = ? AND to_spec = ?`, toSpec, specID); err != nil {
			return fmt.Errorf("unlinking spec %s (reverse): %w", toSpec, err)
		}
	}

	return nil
}

// linkKBChunkCitations creates citation entries linking the spec to KB chunks.
// Each chunk ID is resolved to its doc_id and position before insertion.
func linkKBChunkCitations(db *sql.DB, specID, linkKBChunks, now string) error {
	if linkKBChunks == "" {
		return nil
	}

	for _, chunkStr := range strings.Split(linkKBChunks, ",") {
		chunkStr = strings.TrimSpace(chunkStr)
		if chunkStr == "" {
			continue
		}
		chunkID, err := strconv.Atoi(chunkStr)
		if err != nil {
			return fmt.Errorf("invalid chunk ID %q: %w", chunkStr, err)
		}

		var docID string
		var pos int
		err = db.QueryRow(`SELECT doc_id, position FROM kb_chunks WHERE id = ?`, chunkID).Scan(&docID, &pos)
		if err != nil {
			return fmt.Errorf("chunk %d not found: %w", chunkID, err)
		}

		_, err = db.Exec(`INSERT OR IGNORE INTO citations (from_kind, from_id, kb_doc_id, chunk_position, created_at)
			VALUES ('spec', ?, ?, ?, ?)`, specID, docID, pos, now)
		if err != nil {
			return fmt.Errorf("creating citation: %w", err)
		}
	}

	return nil
}

// unlinkKBChunkCitations removes citation entries for the given chunk IDs.
func unlinkKBChunkCitations(db *sql.DB, specID, unlinkKBChunks string) error {
	if unlinkKBChunks == "" {
		return nil
	}

	for _, chunkStr := range strings.Split(unlinkKBChunks, ",") {
		chunkStr = strings.TrimSpace(chunkStr)
		if chunkStr == "" {
			continue
		}
		chunkID, err := strconv.Atoi(chunkStr)
		if err != nil {
			return fmt.Errorf("invalid chunk ID %q: %w", chunkStr, err)
		}

		// Resolve chunk to doc_id + position for the citation primary key.
		var docID string
		var pos int
		err = db.QueryRow(`SELECT doc_id, position FROM kb_chunks WHERE id = ?`, chunkID).Scan(&docID, &pos)
		if err != nil {
			return fmt.Errorf("chunk %d not found: %w", chunkID, err)
		}

		if _, err := db.Exec(`DELETE FROM citations WHERE from_kind = 'spec' AND from_id = ? AND kb_doc_id = ? AND chunk_position = ?`,
			specID, docID, pos); err != nil {
			return fmt.Errorf("unlinking chunk %d: %w", chunkID, err)
		}
	}

	return nil
}

// getLinkedSpecSummaries returns summaries for all specs currently linked to specID.
func getLinkedSpecSummaries(db *sql.DB, specID string) ([]LinkedSpecSummary, error) {
	rows, err := db.Query(`
		SELECT s.id, s.title, s.summary
		FROM spec_links l
		JOIN specs s ON s.id = l.to_spec
		WHERE l.from_spec = ?
		ORDER BY s.id`, specID)
	if err != nil {
		return nil, fmt.Errorf("reading linked specs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []LinkedSpecSummary{}
	for rows.Next() {
		var s LinkedSpecSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Summary); err != nil {
			return nil, fmt.Errorf("scanning linked spec: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// getLinkedKBChunkSummaries returns summaries for all KB chunks currently
// cited by specID.
func getLinkedKBChunkSummaries(db *sql.DB, specID string) ([]LinkedKBChunkSummary, error) {
	rows, err := db.Query(`
		SELECT k.id, k.doc_id, substr(k.text, 1, ?) AS preview
		FROM citations c
		JOIN kb_chunks k ON k.doc_id = c.kb_doc_id AND k.position = c.chunk_position
		WHERE c.from_kind = 'spec' AND c.from_id = ?
		ORDER BY k.id`, ChunkPreviewLength, specID)
	if err != nil {
		return nil, fmt.Errorf("reading linked KB chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []LinkedKBChunkSummary{}
	for rows.Next() {
		var c LinkedKBChunkSummary
		if err := rows.Scan(&c.ChunkID, &c.DocID, &c.Preview); err != nil {
			return nil, fmt.Errorf("scanning linked KB chunk: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// printUpdateResponse marshals and prints the JSON confirmation.
func printUpdateResponse(specID, specType string, linkedSpecs []LinkedSpecSummary, linkedChunks []LinkedKBChunkSummary) error {
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

// rewriteSpecFile rebuilds a spec.md file from the current DB state.
// This ensures the file stays the ground truth after DB-side changes
// (type updates, link additions/removals, etc.).
func rewriteSpecFile(db *sql.DB, specID string) error {
	var id, title, specType, summary, body, path, createdBy, createdAt, updatedAt string
	var updatedBy sql.NullString
	err := db.QueryRow(`
		SELECT id, title, type, summary, body, path, created_by, updated_by, created_at, updated_at
		FROM specs WHERE id = ?`, specID).Scan(
		&id, &title, &specType, &summary, &body, &path,
		&createdBy, &updatedBy, &createdAt, &updatedAt,
	)
	if err != nil {
		return fmt.Errorf("reading spec from db: %w", err)
	}

	// Read linked spec IDs for the frontmatter.
	rows, err := db.Query("SELECT to_spec FROM spec_links WHERE from_spec = ? ORDER BY to_spec", specID)
	if err != nil {
		return fmt.Errorf("reading spec links: %w", err)
	}
	var linkedSpecs []string
	for rows.Next() {
		var toSpec string
		if err := rows.Scan(&toSpec); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning spec link: %w", err)
		}
		linkedSpecs = append(linkedSpecs, toSpec)
	}
	_ = rows.Close()

	md := buildSpecMarkdown(id, title, summary, specType, createdBy, updatedAt, linkedSpecs, body)

	if err := os.WriteFile(path, []byte(md), 0o644); err != nil { //nolint:gosec // spec file is committed to VCS
		return fmt.Errorf("writing spec file: %w", err)
	}

	// Recompute content_hash from the file we just wrote.
	newHash := fmt.Sprintf("%x", sha256.Sum256([]byte(md)))
	if _, err := db.Exec(`UPDATE specs SET content_hash = ? WHERE id = ?`, newHash, specID); err != nil {
		return fmt.Errorf("updating content_hash: %w", err)
	}

	return nil
}
