// update_task.go implements `specd update-task`, which modifies a task's
// status and/or toggles acceptance criteria checked state. After any change,
// the task's markdown file is rewritten from DB state via rewriteTaskFile()
// so the file remains the ground truth. The criteria toggle operates on
// 1-based positions matching the spec_claims table layout.
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

// updateTaskCmd implements `specd update-task`.
// Supports changing status, toggling acceptance criteria checked state.
var updateTaskCmd = &cobra.Command{
	Use:   "update-task",
	Short: "Update a task's status or toggle acceptance criteria",
	RunE:  runUpdateTask,
}

func init() {
	updateTaskCmd.Flags().String("id", "", "task ID to update (required)")
	updateTaskCmd.Flags().String("status", "", "new task status/stage")
	updateTaskCmd.Flags().String("check", "", "comma-separated criterion positions to mark checked")
	updateTaskCmd.Flags().String("uncheck", "", "comma-separated criterion positions to mark unchecked")
	_ = updateTaskCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(updateTaskCmd)
}

// UpdateTaskCriterion is a criterion in the update response.
type UpdateTaskCriterion struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
	Checked  int    `json:"checked"`
}

// UpdateTaskResponse is the JSON output of the update-task command.
type UpdateTaskResponse struct {
	ID       string                `json:"id"`
	SpecID   string                `json:"spec_id"`
	Status   string                `json:"status"`
	Criteria []UpdateTaskCriterion `json:"criteria"`
}

func runUpdateTask(c *cobra.Command, _ []string) error {
	taskID, _ := c.Flags().GetString("id")
	newStatus, _ := c.Flags().GetString("status")
	checkStr, _ := c.Flags().GetString("check")
	uncheckStr, _ := c.Flags().GetString("uncheck")

	slog.Info("update-task", "id", taskID, "status", newStatus,
		"check", checkStr, "uncheck", uncheckStr)

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()

	// Verify the task exists.
	var specID, currentStatus string
	err = db.QueryRow("SELECT spec_id, status FROM tasks WHERE id = ?", taskID).Scan(&specID, &currentStatus)
	if err != nil {
		return fmt.Errorf("task %s not found: %w", taskID, err)
	}

	// Update status if provided.
	if newStatus != "" {
		if err := applyTaskStatus(db, taskID, newStatus, username, now); err != nil {
			return err
		}
	}

	// Toggle criteria checked state.
	if err := toggleCriteria(db, taskID, checkStr, 1, username); err != nil {
		return err
	}
	if err := toggleCriteria(db, taskID, uncheckStr, 0, ""); err != nil {
		return err
	}

	// Rewrite task file from DB state.
	if err := rewriteTaskFile(db, taskID); err != nil {
		return fmt.Errorf("rewriting task file: %w", err)
	}

	// Build response with current state.
	finalStatus := currentStatus
	if newStatus != "" {
		finalStatus = newStatus
	}

	criteria, err := loadUpdateTaskCriteria(db, taskID)
	if err != nil {
		return err
	}

	resp := UpdateTaskResponse{
		ID:       taskID,
		SpecID:   specID,
		Status:   finalStatus,
		Criteria: criteria,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// applyTaskStatus updates the task status in the database.
func applyTaskStatus(db *sql.DB, taskID, status, username, now string) error {
	_, err := db.Exec(`UPDATE tasks SET status = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		status, username, now, taskID)
	if err != nil {
		return fmt.Errorf("updating task status: %w", err)
	}
	return nil
}

// toggleCriteria sets the checked state for criteria at the given positions.
// positions is a comma-separated string of 1-based position numbers.
func toggleCriteria(db *sql.DB, taskID, positionsStr string, checked int, checkedBy string) error {
	if positionsStr == "" {
		return nil
	}

	for _, posStr := range strings.Split(positionsStr, ",") {
		posStr = strings.TrimSpace(posStr)
		if posStr == "" {
			continue
		}
		pos, err := strconv.Atoi(posStr)
		if err != nil {
			return fmt.Errorf("invalid criterion position %q: %w", posStr, err)
		}

		var cbPtr *string
		if checkedBy != "" {
			cbPtr = &checkedBy
		}

		res, err := db.Exec(
			`UPDATE task_criteria SET checked = ?, checked_by = ? WHERE task_id = ? AND position = ?`,
			checked, cbPtr, taskID, pos)
		if err != nil {
			return fmt.Errorf("toggling criterion %d: %w", pos, err)
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("criterion at position %d not found for task %s", pos, taskID)
		}
	}

	return nil
}

// loadUpdateTaskCriteria reads the current criteria for the response.
func loadUpdateTaskCriteria(db *sql.DB, taskID string) ([]UpdateTaskCriterion, error) {
	rows, err := db.Query(`
		SELECT position, text, checked
		FROM task_criteria WHERE task_id = ? ORDER BY position`, taskID)
	if err != nil {
		return nil, fmt.Errorf("reading criteria: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []UpdateTaskCriterion{}
	for rows.Next() {
		var cr UpdateTaskCriterion
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

// rewriteTaskFile rebuilds a TASK-N.md file from the current DB state.
// This ensures the file stays the ground truth after DB-side changes.
func rewriteTaskFile(db *sql.DB, taskID string) error {
	var id, specID, title, status, summary, body, path, createdBy, createdAt, updatedAt string
	err := db.QueryRow(`
		SELECT id, spec_id, title, status, summary, body, path,
		       created_by, created_at, updated_at
		FROM tasks WHERE id = ?`, taskID).Scan(
		&id, &specID, &title, &status, &summary, &body, &path,
		&createdBy, &createdAt, &updatedAt,
	)
	if err != nil {
		return fmt.Errorf("reading task from db: %w", err)
	}

	// Read linked task IDs.
	linkedTasks, err := queryStringList(db,
		"SELECT to_task FROM task_links WHERE from_task = ? ORDER BY to_task", taskID)
	if err != nil {
		return fmt.Errorf("reading task links: %w", err)
	}

	// Read depends_on IDs.
	dependsOn, err := queryStringList(db,
		"SELECT blocker_task FROM task_dependencies WHERE blocked_task = ? ORDER BY blocker_task", taskID)
	if err != nil {
		return fmt.Errorf("reading task dependencies: %w", err)
	}

	// Read criteria to rebuild the body with correct checked state.
	rows, err := db.Query(
		"SELECT text, checked FROM task_criteria WHERE task_id = ? ORDER BY position", taskID)
	if err != nil {
		return fmt.Errorf("reading task criteria: %w", err)
	}
	var criteria []criterionState
	for rows.Next() {
		var cs criterionState
		if err := rows.Scan(&cs.text, &cs.checked); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning criterion: %w", err)
		}
		criteria = append(criteria, cs)
	}
	_ = rows.Close()

	// Rebuild the body with updated checkbox states in Acceptance Criteria.
	rebuiltBody := rebuildTaskBody(body, criteria)

	md := buildTaskMarkdown(id, specID, title, summary, status, createdBy, updatedAt, linkedTasks, dependsOn, rebuiltBody)

	if err := os.WriteFile(path, []byte(md), 0o644); err != nil { //nolint:gosec // task file is committed to VCS
		return fmt.Errorf("writing task file: %w", err)
	}

	// Recompute content_hash from the file we just wrote.
	newHash := fmt.Sprintf("%x", sha256.Sum256([]byte(md)))
	if _, err := db.Exec(`UPDATE tasks SET content_hash = ?, body = ? WHERE id = ?`, newHash, rebuiltBody, taskID); err != nil {
		return fmt.Errorf("updating content_hash: %w", err)
	}

	return nil
}

// queryStringList executes a query returning a single string column and collects results.
func queryStringList(db *sql.DB, query, arg string) ([]string, error) {
	rows, err := db.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// rebuildTaskBody replaces the checkboxes in ## Acceptance Criteria with
// the checked state from the database.
func rebuildTaskBody(body string, criteria []criterionState) string {
	lines := strings.Split(body, "\n")
	var result []string
	inSection := false
	criteriaIdx := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			inSection = strings.EqualFold(heading, "Acceptance Criteria")
			result = append(result, line)
			continue
		}

		if inSection {
			// End of section on H1/H2.
			if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "### ") {
				inSection = false
				result = append(result, line)
				continue
			}

			trimmed := strings.TrimSpace(line)
			if parseCheckboxItem(trimmed) != "" && criteriaIdx < len(criteria) {
				// Replace this checkbox line with the DB state.
				cs := criteria[criteriaIdx]
				checkbox := "[ ]"
				if cs.checked == 1 {
					checkbox = "[x]"
				}
				result = append(result, fmt.Sprintf("- %s %s", checkbox, cs.text))
				criteriaIdx++
				continue
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// criterionState holds the checked state for a single criterion.
// Used by rebuildTaskBody.
type criterionState struct {
	text    string
	checked int
}
