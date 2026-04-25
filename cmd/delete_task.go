package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// deleteTaskCmd implements `specd delete-task`.
// It removes the task from the database (cascading to criteria, links, deps)
// and deletes the task markdown file from disk.
var deleteTaskCmd = &cobra.Command{
	Use:   "delete-task",
	Short: "Delete a task",
	RunE:  runDeleteTask,
}

func init() {
	deleteTaskCmd.Flags().String("id", "", "task ID to delete (required)")
	_ = deleteTaskCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(deleteTaskCmd)
}

// DeleteTaskResponse is the JSON output of the delete-task command.
type DeleteTaskResponse struct {
	ID      string `json:"id"`
	SpecID  string `json:"spec_id"`
	Deleted bool   `json:"deleted"`
	Path    string `json:"path"`
}

func runDeleteTask(c *cobra.Command, _ []string) error {
	taskID, _ := c.Flags().GetString("id")

	slog.Info("delete-task", "id", taskID)

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Read the task's path and spec_id before deleting.
	var path, specID string
	err = db.QueryRow("SELECT path, spec_id FROM tasks WHERE id = ?", taskID).Scan(&path, &specID)
	if err != nil {
		return fmt.Errorf("task %s not found: %w", taskID, err)
	}

	// Delete from the database. ON DELETE CASCADE handles task_criteria,
	// task_links, task_dependencies, and citations.
	if _, err := db.Exec("DELETE FROM tasks WHERE id = ?", taskID); err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}

	// Remove the task file from disk.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing task file: %w", err)
	}

	resp := DeleteTaskResponse{
		ID:      taskID,
		SpecID:  specID,
		Deleted: true,
		Path:    path,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}
