// list_tasks.go implements `specd list-tasks`. Returns a paginated JSON list
// of tasks, optionally filtered by parent spec ID or status. Supports --page,
// --page-size, --spec-id, and --status flags.
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// listTasksCmd implements `specd list-tasks`.
// Returns a paginated list of tasks, optionally filtered by spec ID or status.
var listTasksCmd = &cobra.Command{
	Use:   "list-tasks",
	Short: "List tasks with pagination",
	RunE:  runListTasks,
}

func init() {
	listTasksCmd.Flags().Int("page", 1, "page number (1-based)")
	listTasksCmd.Flags().Int("page-size", DefaultPageSize, "results per page")
	listTasksCmd.Flags().String("spec-id", "", "filter by parent spec ID")
	listTasksCmd.Flags().String("status", "", "filter by task status")
	rootCmd.AddCommand(listTasksCmd)
}

// ListTaskItem is a single task in the list response.
type ListTaskItem struct {
	ID        string `json:"id"`
	SpecID    string `json:"spec_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListTasksResponse is the JSON output of the list-tasks command.
type ListTasksResponse struct {
	Tasks      []ListTaskItem `json:"tasks"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
}

func runListTasks(c *cobra.Command, _ []string) error {
	page, _ := c.Flags().GetInt("page")
	pageSize, _ := c.Flags().GetInt("page-size")
	specID, _ := c.Flags().GetString("spec-id")
	status, _ := c.Flags().GetString("status")

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

	// Build WHERE clause from filters.
	where, args := buildTaskFilters(specID, status)

	// Count total matching tasks.
	var total int
	countSQL := "SELECT COUNT(*) FROM tasks" + where
	if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return fmt.Errorf("counting tasks: %w", err)
	}

	totalPages := (total + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	querySQL := "SELECT id, spec_id, title, status, summary, position, created_at, updated_at FROM tasks" + where + " ORDER BY position, id LIMIT ? OFFSET ?" //nolint:gosec // where is built from hardcoded column names with parameterized values
	queryArgs := append(args, pageSize, offset)                                                                                                               //nolint:gocritic // append to copy is intentional

	rows, err := db.Query(querySQL, queryArgs...)
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tasks := []ListTaskItem{}
	for rows.Next() {
		var t ListTaskItem
		if err := rows.Scan(&t.ID, &t.SpecID, &t.Title, &t.Status, &t.Summary, &t.Position, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating tasks: %w", err)
	}

	resp := ListTasksResponse{
		Tasks:      tasks,
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

// buildTaskFilters constructs a SQL WHERE clause and args for optional
// spec_id and status filters.
func buildTaskFilters(specID, status string) (where string, args []any) {
	var conditions []string

	if specID != "" {
		conditions = append(conditions, "spec_id = ?")
		args = append(args, specID)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	if len(conditions) == 0 {
		return "", nil
	}

	where = " WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		where += " AND " + c
	}
	return where, args
}
