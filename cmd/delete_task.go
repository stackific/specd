package cmd

import (
	"database/sql"
	"encoding/json"
	"errors"
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

// DeleteTaskResponse is the JSON output of the delete-task command and the
// /api/tasks/{id} DELETE endpoint.
type DeleteTaskResponse struct {
	ID      string `json:"id"`
	SpecID  string `json:"spec_id"`
	Deleted bool   `json:"deleted"`
	Path    string `json:"path"`
}

// ErrTaskNotFound is returned by DeleteTask when no task matches the given ID.
// Callers (CLI vs. API handler) use this to translate into the right exit
// behaviour (CLI: error string; API: 404).
var ErrTaskNotFound = errors.New("task not found")

// DeleteTask removes a task from the database and its markdown file from disk.
// ON DELETE CASCADE handles task_criteria, task_links, task_dependencies, and
// citations. Returns ErrTaskNotFound when the ID is unknown so callers can
// distinguish it from internal failures.
func DeleteTask(db *sql.DB, taskID string) (*DeleteTaskResponse, error) {
	var path, specID string
	err := db.QueryRow("SELECT path, spec_id FROM tasks WHERE id = ?", taskID).Scan(&path, &specID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("looking up task %s: %w", taskID, err)
	}

	if _, err := db.Exec("DELETE FROM tasks WHERE id = ?", taskID); err != nil {
		return nil, fmt.Errorf("deleting task: %w", err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("removing task file: %w", err)
	}

	return &DeleteTaskResponse{
		ID:      taskID,
		SpecID:  specID,
		Deleted: true,
		Path:    path,
	}, nil
}

func runDeleteTask(c *cobra.Command, _ []string) error {
	taskID, _ := c.Flags().GetString("id")

	slog.Info("delete-task", "id", taskID)

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	resp, err := DeleteTask(db, taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return fmt.Errorf("task %s not found", taskID)
		}
		return err
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}
