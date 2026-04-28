// get_task.go implements `specd get-task --id TASK-N`. Returns a single task
// as JSON including its acceptance criteria (with checked state), linked tasks,
// and dependency (depends_on) relationships.
package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// getTaskCmd implements `specd get-task --id TASK-1`.
// Returns a single task by ID as JSON, including criteria and links.
var getTaskCmd = &cobra.Command{
	Use:   "get-task",
	Short: "Get a task by ID",
	RunE:  runGetTask,
}

func init() {
	getTaskCmd.Flags().String("id", "", "task ID (required)")
	_ = getTaskCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(getTaskCmd)
}

// GetTaskCriterion is a single acceptance criterion on a task.
type GetTaskCriterion struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
	Checked  int    `json:"checked"`
}

// GetTaskResponse is the JSON output of the get-task command.
type GetTaskResponse struct {
	ID          string             `json:"id"`
	SpecID      string             `json:"spec_id"`
	Title       string             `json:"title"`
	Status      string             `json:"status"`
	Summary     string             `json:"summary"`
	Body        string             `json:"body"`
	Path        string             `json:"path"`
	Position    int                `json:"position"`
	LinkedTasks []string           `json:"linked_tasks"`
	DependsOn   []string           `json:"depends_on"`
	Criteria    []GetTaskCriterion `json:"criteria"`
	CreatedBy   string             `json:"created_by,omitempty"`
	UpdatedBy   string             `json:"updated_by,omitempty"`
	ContentHash string             `json:"content_hash"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

func runGetTask(c *cobra.Command, _ []string) error {
	taskID, _ := c.Flags().GetString("id")

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	resp, err := LoadTaskDetail(db, taskID)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// loadLinkedTasks reads task link IDs for the given task.
func loadLinkedTasks(db *sql.DB, taskID string) ([]string, error) {
	rows, err := db.Query("SELECT to_task FROM task_links WHERE from_task = ? ORDER BY to_task", taskID)
	if err != nil {
		return nil, fmt.Errorf("reading task links: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []string{}
	for rows.Next() {
		var toTask string
		if err := rows.Scan(&toTask); err != nil {
			return nil, fmt.Errorf("scanning task link: %w", err)
		}
		result = append(result, toTask)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task links: %w", err)
	}
	return result, nil
}

// loadTaskDependsOn reads blocker task IDs for the given task.
func loadTaskDependsOn(db *sql.DB, taskID string) ([]string, error) {
	rows, err := db.Query(
		"SELECT blocker_task FROM task_dependencies WHERE blocked_task = ? ORDER BY blocker_task", taskID)
	if err != nil {
		return nil, fmt.Errorf("reading task dependencies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []string{}
	for rows.Next() {
		var blocker string
		if err := rows.Scan(&blocker); err != nil {
			return nil, fmt.Errorf("scanning task dependency: %w", err)
		}
		result = append(result, blocker)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task dependencies: %w", err)
	}
	return result, nil
}

// loadGetTaskCriteria reads acceptance criteria for a single task.
func loadGetTaskCriteria(db *sql.DB, taskID string) ([]GetTaskCriterion, error) {
	rows, err := db.Query(`
		SELECT position, text, checked
		FROM task_criteria WHERE task_id = ? ORDER BY position`, taskID)
	if err != nil {
		return nil, fmt.Errorf("reading criteria for %s: %w", taskID, err)
	}
	defer func() { _ = rows.Close() }()

	result := []GetTaskCriterion{}
	for rows.Next() {
		var cr GetTaskCriterion
		if err := rows.Scan(&cr.Position, &cr.Text, &cr.Checked); err != nil {
			return nil, fmt.Errorf("scanning criterion: %w", err)
		}
		result = append(result, cr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating criteria: %w", err)
	}
	return result, nil
}
