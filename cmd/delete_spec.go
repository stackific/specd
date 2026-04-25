package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// deleteSpecCmd implements `specd delete-spec`.
// It removes the spec from the database (cascading to tasks, links, criteria)
// and deletes the spec directory from disk.
var deleteSpecCmd = &cobra.Command{
	Use:   "delete-spec",
	Short: "Delete a spec and its tasks",
	RunE:  runDeleteSpec,
}

func init() {
	deleteSpecCmd.Flags().String("id", "", "spec ID to delete (required)")
	_ = deleteSpecCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(deleteSpecCmd)
}

// DeleteSpecResponse is the JSON output of the delete-spec command.
type DeleteSpecResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
	Path    string `json:"path"`
}

func runDeleteSpec(c *cobra.Command, _ []string) error {
	specID, _ := c.Flags().GetString("id")

	slog.Info("delete-spec", "id", specID)

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Read the spec's path before deleting so we can remove the directory.
	var path string
	err = db.QueryRow("SELECT path FROM specs WHERE id = ?", specID).Scan(&path)
	if err != nil {
		return fmt.Errorf("spec %s not found: %w", specID, err)
	}

	// Delete from the database. ON DELETE CASCADE handles tasks, spec_links,
	// task_criteria, task_links, task_dependencies, and citations.
	if _, err := db.Exec("DELETE FROM specs WHERE id = ?", specID); err != nil {
		return fmt.Errorf("deleting spec: %w", err)
	}

	// Remove the spec directory from disk (e.g. specd/specs/spec-1/).
	// The path points to spec.md — remove its parent directory.
	specDir := filepath.Dir(path)
	if err := os.RemoveAll(specDir); err != nil {
		return fmt.Errorf("removing spec directory: %w", err)
	}

	resp := DeleteSpecResponse{
		ID:      specID,
		Deleted: true,
		Path:    specDir,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}
