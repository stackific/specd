// new_spec.go implements `specd new-spec`. Creates a spec markdown file in
// <specd-folder>/specs/spec-<N>/spec.md, inserts the database record and
// acceptance criteria claims, then returns JSON with related specs and KB
// chunks for the AI skill to use when selecting type and links.
package cmd

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// newSpecCmd implements `specd new-spec`.
// It creates a new spec with an auto-assigned ID, writes the markdown file,
// inserts into the database, and returns JSON with the spec details and
// related content for the AI to decide on type and links.
var newSpecCmd = &cobra.Command{
	Use:   "new-spec",
	Short: "Create a new spec",
	RunE:  runNewSpec,
}

func init() {
	newSpecCmd.Flags().String("title", "", "spec title (required)")
	newSpecCmd.Flags().String("summary", "", "one-line summary (required)")
	newSpecCmd.Flags().String("body", "", "markdown body (required)")
	_ = newSpecCmd.MarkFlagRequired("title")
	_ = newSpecCmd.MarkFlagRequired("summary")
	_ = newSpecCmd.MarkFlagRequired("body")
	rootCmd.AddCommand(newSpecCmd)
}

// NewSpecResponse is the JSON output of the new-spec command.
// The AI skill parses this to decide on spec type and links.
type NewSpecResponse struct {
	ID             string         `json:"id"`
	Path           string         `json:"path"`
	DefaultType    string         `json:"default_type"`
	AvailableTypes []string       `json:"available_types"`
	RelatedSpecs   []SearchResult `json:"related_specs"`
	RelatedKB      []SearchResult `json:"related_kb_chunks"`
}

func runNewSpec(c *cobra.Command, _ []string) error {
	title, _ := c.Flags().GetString("title")
	summary, _ := c.Flags().GetString("summary")
	body, _ := c.Flags().GetString("body")

	slog.Info("new-spec", "title", title)

	// Open the project database.
	db, specdFolder, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Load project config for spec types.
	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		return fmt.Errorf("cannot read project config")
	}

	// Get the next spec number atomically.
	num, err := NextID(db, MetaNextSpecID)
	if err != nil {
		return err
	}

	specID := fmt.Sprintf("%s%d", IDPrefixSpec, num)
	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()
	// Build the relative path: <specd-folder>/specs/spec-<N>/spec.md
	specDir := filepath.Join(specdFolder, SpecsSubdir, fmt.Sprintf("spec-%d", num))
	specFile := filepath.Join(specDir, "spec.md")

	// Create the spec directory.
	if err := os.MkdirAll(specDir, 0o755); err != nil { //nolint:gosec // spec dir is part of VCS repo
		return fmt.Errorf("creating spec directory: %w", err)
	}

	// Write the spec markdown file with frontmatter.
	// No linked_specs yet — those are added in step 2 via update-spec.
	md := buildSpecMarkdown(specID, title, summary, proj.SpecTypes[0], username, now, nil, body)
	// Hash the full file content so the sync detects any change.
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(md)))
	if err := os.WriteFile(specFile, []byte(md), 0o644); err != nil { //nolint:gosec // spec file is committed to VCS
		return fmt.Errorf("writing spec file: %w", err)
	}

	// Insert into the database. Uses the default spec type (first in the list).
	_, err = db.Exec(`
		INSERT INTO specs (id, title, type, summary, body, path, created_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, specID, title, proj.SpecTypes[0], summary, body, specFile, username, contentHash, now, now)
	if err != nil {
		return fmt.Errorf("inserting spec: %w", err)
	}

	// Parse and insert acceptance criteria (claims) from the body.
	claims := extractClaims(body)
	for i, text := range claims {
		if _, err := db.Exec(
			"INSERT INTO spec_claims (spec_id, position, text) VALUES (?, ?, ?)",
			specID, i+1, text,
		); err != nil {
			return fmt.Errorf("inserting spec claim %d: %w", i+1, err)
		}
	}

	relatedSpecs, relatedKB, err := findRelatedContent(db, proj, title+" "+summary, specID)
	if err != nil {
		return err
	}

	resp := NewSpecResponse{
		ID:             specID,
		Path:           specFile,
		DefaultType:    proj.SpecTypes[0],
		AvailableTypes: proj.SpecTypes,
		RelatedSpecs:   relatedSpecs,
		RelatedKB:      relatedKB,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// findRelatedContent searches for specs and KB chunks related to the given
// text, using project-configured limits and weights with defaults as fallback.
func findRelatedContent(db *sql.DB, proj *ProjectConfig, searchText, excludeID string) (specs, kb []SearchResult, err error) {
	limit := proj.TopSearchResults
	if limit <= 0 {
		limit = TopSearchResults
	}
	weights := proj.SearchWeights
	if weights.Title == 0 && weights.Summary == 0 && weights.Body == 0 {
		weights = defaultSearchWeights()
	}

	results, err := Search(db, searchText, KindAll, limit, excludeID, weights)
	if err != nil {
		return nil, nil, fmt.Errorf("searching related content: %w", err)
	}

	specs = results.Specs
	if specs == nil {
		specs = []SearchResult{}
	}
	kb = results.KB
	if kb == nil {
		kb = []SearchResult{}
	}

	return specs, kb, nil
}

// buildSpecMarkdown generates spec.md with YAML frontmatter and H1 title.
// The title is NOT in frontmatter — the H1 heading IS the title.
// linkedSpecs may be nil for new specs that don't have links yet.
func buildSpecMarkdown(id, title, summary, specType, createdBy, timestamp string, linkedSpecs []string, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "id: %s\n", id)
	fmt.Fprintf(&b, "type: %s\n", specType)
	fmt.Fprintf(&b, "summary: %s\n", summary)
	fmt.Fprintf(&b, "position: 0\n")
	if len(linkedSpecs) > 0 {
		fmt.Fprintf(&b, "linked_specs:\n")
		for _, ls := range linkedSpecs {
			fmt.Fprintf(&b, "  - %s\n", ls)
		}
	}
	fmt.Fprintf(&b, "created_by: %s\n", createdBy)
	fmt.Fprintf(&b, "created_at: %s\n", timestamp)
	fmt.Fprintf(&b, "updated_at: %s\n", timestamp)
	fmt.Fprintf(&b, "---\n\n# %s\n\n%s\n", title, body)
	return b.String()
}
