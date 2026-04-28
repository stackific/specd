// new_task.go implements `specd new-task`. Creates a task markdown file as
// TASK-<N>.md inside the parent spec's directory, inserts the database record
// and acceptance criteria, then returns JSON with the task ID and path.
// Task criteria use checkbox syntax (- [ ] / - [x]) unlike spec claims
// which use plain bullets.
package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// newTaskCmd implements `specd new-task`.
// It creates a new task linked to a spec, writes the markdown file,
// inserts into the database, and returns JSON with the task details.
var newTaskCmd = &cobra.Command{
	Use:   "new-task",
	Short: "Create a new task for a spec",
	RunE:  runNewTask,
}

func init() {
	newTaskCmd.Flags().String("spec-id", "", "parent spec ID (required)")
	newTaskCmd.Flags().String("title", "", "task title (required)")
	newTaskCmd.Flags().String("summary", "", "one-line summary (required)")
	newTaskCmd.Flags().String("body", "", "markdown body (required)")
	_ = newTaskCmd.MarkFlagRequired("spec-id")
	_ = newTaskCmd.MarkFlagRequired("title")
	_ = newTaskCmd.MarkFlagRequired("summary")
	_ = newTaskCmd.MarkFlagRequired("body")
	rootCmd.AddCommand(newTaskCmd)
}

// NewTaskResponse is the JSON output of the new-task command.
type NewTaskResponse struct {
	ID     string `json:"id"`
	SpecID string `json:"spec_id"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

func runNewTask(c *cobra.Command, _ []string) error {
	specID, _ := c.Flags().GetString("spec-id")
	title, _ := c.Flags().GetString("title")
	summary, _ := c.Flags().GetString("summary")
	body, _ := c.Flags().GetString("body")

	slog.Info("new-task", "spec_id", specID, "title", title)

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Load project config for task stages.
	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		return fmt.Errorf("cannot read project config")
	}

	// Verify the parent spec exists.
	var specExists int
	err = db.QueryRow("SELECT 1 FROM specs WHERE id = ?", specID).Scan(&specExists)
	if err != nil {
		return fmt.Errorf("spec %s not found", specID)
	}

	// Get the next task number atomically.
	num, err := NextID(db, MetaNextTaskID)
	if err != nil {
		return err
	}

	taskID := fmt.Sprintf("%s%d", IDPrefixTask, num)
	now := time.Now().UTC().Format(time.RFC3339)
	username := ResolveActiveUsername()

	// Default status is the first task stage (backlog).
	defaultStatus := proj.TaskStages[0]

	// Query the parent spec's path to find its directory.
	var specPath string
	err = db.QueryRow("SELECT path FROM specs WHERE id = ?", specID).Scan(&specPath)
	if err != nil {
		return fmt.Errorf("spec %s not found", specID)
	}
	specDir := filepath.Dir(specPath)
	taskFile := filepath.Join(specDir, fmt.Sprintf("%s%d.md", IDPrefixTask, num))

	// Write the task markdown file with frontmatter.
	md := buildTaskMarkdown(taskID, specID, title, summary, defaultStatus, username, now, 0, nil, nil, body)
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(md)))
	if err := os.WriteFile(taskFile, []byte(md), 0o644); err != nil { //nolint:gosec // task file is committed to VCS
		return fmt.Errorf("writing task file: %w", err)
	}

	// Insert into the database.
	_, err = db.Exec(`
		INSERT INTO tasks (id, spec_id, title, status, summary, body, path, created_by, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, taskID, specID, title, defaultStatus, summary, body, taskFile, username, contentHash, now, now)
	if err != nil {
		return fmt.Errorf("inserting task: %w", err)
	}

	// Parse and insert acceptance criteria from the body.
	criteria := extractTaskCriteria(body)
	for i, text := range criteria {
		if _, err := db.Exec(
			"INSERT INTO task_criteria (task_id, position, text) VALUES (?, ?, ?)",
			taskID, i+1, text,
		); err != nil {
			return fmt.Errorf("inserting task criteria %d: %w", i+1, err)
		}
	}

	resp := NewTaskResponse{
		ID:     taskID,
		SpecID: specID,
		Path:   taskFile,
		Status: defaultStatus,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// buildTaskMarkdown generates task.md with YAML frontmatter and H1 title.
// The title is NOT in frontmatter — the H1 heading IS the title.
func buildTaskMarkdown(id, specID, title, summary, status, createdBy, timestamp string, position int, linkedTasks, dependsOn []string, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "id: %s\n", id)
	fmt.Fprintf(&b, "spec_id: %s\n", specID)
	fmt.Fprintf(&b, "status: %s\n", status)
	fmt.Fprintf(&b, "summary: %s\n", summary)
	fmt.Fprintf(&b, "position: %d\n", position)
	if len(linkedTasks) > 0 {
		fmt.Fprintf(&b, "linked_tasks:\n")
		for _, lt := range linkedTasks {
			fmt.Fprintf(&b, "  - %s\n", lt)
		}
	}
	if len(dependsOn) > 0 {
		fmt.Fprintf(&b, "depends_on:\n")
		for _, dep := range dependsOn {
			fmt.Fprintf(&b, "  - %s\n", dep)
		}
	}
	fmt.Fprintf(&b, "created_by: %s\n", createdBy)
	fmt.Fprintf(&b, "created_at: %s\n", timestamp)
	fmt.Fprintf(&b, "updated_at: %s\n", timestamp)
	fmt.Fprintf(&b, "---\n\n# %s\n\n%s\n", title, body)
	return b.String()
}

// extractTaskCriteria parses checkbox items from ## Acceptance Criteria.
// Task criteria use checkbox syntax: `- [ ] text` or `- [x] text`.
// Returns only the text portion; checked state is not relevant at creation.
func extractTaskCriteria(body string) []string {
	var criteria []string
	inSection := false

	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			inSection = strings.EqualFold(heading, "Acceptance Criteria")
			continue
		}

		if !inSection {
			continue
		}

		// A new H1 or H2 ends the section.
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "### ") {
			break
		}

		trimmed := strings.TrimSpace(line)
		// Parse checkbox items: - [ ] text or - [x] text
		text := parseCheckboxItem(trimmed)
		if text != "" {
			criteria = append(criteria, text)
		}
	}

	return criteria
}

// parseCheckboxItem extracts the text from a markdown checkbox item.
// Returns "" if the line is not a checkbox item.
func parseCheckboxItem(line string) string {
	for _, prefix := range []string{"- [ ] ", "- [x] ", "- [X] "} {
		if strings.HasPrefix(line, prefix) {
			text := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if text != "" {
				return text
			}
		}
	}
	return ""
}
