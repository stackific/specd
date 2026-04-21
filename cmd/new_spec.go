package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Slug           string         `json:"slug"`
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
	num, err := NextID(db, "next_spec_id")
	if err != nil {
		return err
	}

	specID := fmt.Sprintf("SPEC-%d", num)
	slug := ToDashSlug(title)
	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))

	// Build the relative path: <specd-folder>/specs/spec-<N>/spec.md
	specDir := filepath.Join(specdFolder, SpecsSubdir, fmt.Sprintf("spec-%d", num))
	specFile := filepath.Join(specDir, "spec.md")

	// Create the spec directory.
	if err := os.MkdirAll(specDir, 0o755); err != nil { //nolint:gosec // spec dir is part of VCS repo
		return fmt.Errorf("creating spec directory: %w", err)
	}

	// Write the spec markdown file with frontmatter.
	md := buildSpecMarkdown(specID, slug, title, summary, proj.SpecTypes[0], username, now, body)
	if err := os.WriteFile(specFile, []byte(md), 0o644); err != nil { //nolint:gosec // spec file is committed to VCS
		return fmt.Errorf("writing spec file: %w", err)
	}

	// Insert into the database. Uses the default spec type (first in the list).
	_, err = db.Exec(`
		INSERT INTO specs (id, slug, title, type, summary, body, path, created_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, specID, slug, title, proj.SpecTypes[0], summary, body, specFile, username, contentHash, now, now)
	if err != nil {
		return fmt.Errorf("inserting spec: %w", err)
	}

	// Hybrid BM25 + trigram search for related content.
	searchText := title + " " + summary
	limit := proj.TopSearchResults
	if limit <= 0 {
		limit = TopSearchResults
	}
	searchResults, _ := Search(db, searchText, "all", limit, specID)

	// Build response for the AI skill. Ensure empty arrays, never null.
	relatedSpecs := searchResults.Specs
	if relatedSpecs == nil {
		relatedSpecs = []SearchResult{}
	}
	relatedKB := searchResults.KB
	if relatedKB == nil {
		relatedKB = []SearchResult{}
	}

	resp := NewSpecResponse{
		ID:             specID,
		Slug:           slug,
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

// buildSpecMarkdown generates the spec.md content with YAML frontmatter.
func buildSpecMarkdown(id, slug, title, summary, specType, createdBy, timestamp, body string) string {
	return fmt.Sprintf(`---
id: %s
slug: %s
title: %s
type: %s
summary: %s
created_by: %s
created_at: %s
updated_at: %s
---

%s
`, id, slug, title, specType, summary, createdBy, timestamp, timestamp, body)
}
