// get_spec.go implements `specd get-spec --id SPEC-N`. Returns a single spec
// as JSON including its claims (acceptance criteria), linked specs, and all
// child tasks with their criteria. This is the primary data source for AI
// skills that need full spec context (e.g. specd-new-tasks gap analysis).
package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// getSpecCmd implements `specd get-spec --id SPEC-1`.
// Returns a single spec by ID as JSON, including linked specs.
var getSpecCmd = &cobra.Command{
	Use:   "get-spec",
	Short: "Get a spec by ID",
	RunE:  runGetSpec,
}

func init() {
	getSpecCmd.Flags().String("id", "", "spec ID (required)")
	_ = getSpecCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(getSpecCmd)
}

// GetSpecTaskCriterion is a single acceptance criterion on a task.
type GetSpecTaskCriterion struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
	Checked  int    `json:"checked"`
}

// GetSpecTask is a task belonging to the spec, included in the get-spec response.
type GetSpecTask struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Status   string                 `json:"status"`
	Summary  string                 `json:"summary"`
	Criteria []GetSpecTaskCriterion `json:"criteria"`
}

// GetSpecClaim is a single acceptance criterion from the spec.
type GetSpecClaim struct {
	Position int    `json:"position"`
	Text     string `json:"text"`
}

// GetSpecResponse is the JSON output of the get-spec command.
type GetSpecResponse struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Type        string         `json:"type"`
	Summary     string         `json:"summary"`
	Body        string         `json:"body"`
	Path        string         `json:"path"`
	Position    int            `json:"position"`
	LinkedSpecs []string       `json:"linked_specs"`
	Claims      []GetSpecClaim `json:"claims"`
	Tasks       []GetSpecTask  `json:"tasks"`
	CreatedBy   string         `json:"created_by,omitempty"`
	UpdatedBy   string         `json:"updated_by,omitempty"`
	ContentHash string         `json:"content_hash"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

func runGetSpec(c *cobra.Command, _ []string) error {
	specID, _ := c.Flags().GetString("id")

	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Read the spec row.
	var resp GetSpecResponse
	var updatedBy *string
	err = db.QueryRow(`
		SELECT id, title, type, summary, body, path, position,
		       created_by, updated_by, content_hash, created_at, updated_at
		FROM specs WHERE id = ?`, specID).Scan(
		&resp.ID, &resp.Title, &resp.Type, &resp.Summary,
		&resp.Body, &resp.Path, &resp.Position,
		&resp.CreatedBy, &updatedBy, &resp.ContentHash,
		&resp.CreatedAt, &resp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("spec %s not found: %w", specID, err)
	}
	if updatedBy != nil {
		resp.UpdatedBy = *updatedBy
	}

	if resp.LinkedSpecs, err = loadLinkedSpecs(db, specID); err != nil {
		return err
	}
	if resp.Claims, err = loadSpecClaims(db, specID); err != nil {
		return err
	}
	if resp.Tasks, err = loadSpecTasks(db, specID); err != nil {
		return err
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Println(string(out))

	return nil
}

// loadLinkedSpecs reads spec link IDs for the given spec.
func loadLinkedSpecs(db *sql.DB, specID string) ([]string, error) {
	rows, err := db.Query("SELECT to_spec FROM spec_links WHERE from_spec = ? ORDER BY to_spec", specID)
	if err != nil {
		return nil, fmt.Errorf("reading spec links: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []string{}
	for rows.Next() {
		var toSpec string
		if err := rows.Scan(&toSpec); err != nil {
			return nil, fmt.Errorf("scanning spec link: %w", err)
		}
		result = append(result, toSpec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating spec links: %w", err)
	}
	return result, nil
}

// loadSpecClaims reads acceptance criteria claims for the given spec.
func loadSpecClaims(db *sql.DB, specID string) ([]GetSpecClaim, error) {
	rows, err := db.Query(
		"SELECT position, text FROM spec_claims WHERE spec_id = ? ORDER BY position", specID)
	if err != nil {
		return nil, fmt.Errorf("reading spec claims: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []GetSpecClaim{}
	for rows.Next() {
		var cl GetSpecClaim
		if err := rows.Scan(&cl.Position, &cl.Text); err != nil {
			return nil, fmt.Errorf("scanning spec claim: %w", err)
		}
		result = append(result, cl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating spec claims: %w", err)
	}
	return result, nil
}

// loadSpecTasks reads tasks and their criteria for the given spec.
func loadSpecTasks(db *sql.DB, specID string) ([]GetSpecTask, error) {
	rows, err := db.Query(`
		SELECT id, title, status, summary
		FROM tasks WHERE spec_id = ? ORDER BY position`, specID)
	if err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []GetSpecTask
	for rows.Next() {
		var t GetSpecTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Summary); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		t.Criteria = []GetSpecTaskCriterion{}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tasks: %w", err)
	}

	// Load criteria for each task.
	for i, t := range tasks {
		criteria, err := loadTaskCriteria(db, t.ID)
		if err != nil {
			return nil, err
		}
		tasks[i].Criteria = criteria
	}

	if tasks == nil {
		tasks = []GetSpecTask{}
	}
	return tasks, nil
}

// loadTaskCriteria reads acceptance criteria for a single task.
func loadTaskCriteria(db *sql.DB, taskID string) ([]GetSpecTaskCriterion, error) {
	rows, err := db.Query(`
		SELECT position, text, checked
		FROM task_criteria WHERE task_id = ? ORDER BY position`, taskID)
	if err != nil {
		return nil, fmt.Errorf("reading criteria for %s: %w", taskID, err)
	}
	defer func() { _ = rows.Close() }()

	result := []GetSpecTaskCriterion{}
	for rows.Next() {
		var cr GetSpecTaskCriterion
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
