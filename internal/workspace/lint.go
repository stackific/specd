// Package workspace — lint.go implements read-only consistency checks for
// the workspace. Lint detects dangling references, orphan specs/tasks,
// system-field drift, stale tidy, missing summaries, dependency cycles,
// rejected files, KB integrity issues, and citation problems.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
)

// LintIssue represents a single lint finding.
type LintIssue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Category string `json:"category"`
	ID       string `json:"id,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// LintResult holds the complete lint report.
type LintResult struct {
	Issues []LintIssue `json:"issues"`
	Counts struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
	} `json:"counts"`
}

// Lint runs read-only consistency checks across the workspace.
func (w *Workspace) Lint() (*LintResult, error) {
	result := &LintResult{}

	// 1. Dangling spec link references.
	w.lintDanglingSpecLinks(result)

	// 2. Dangling task link references.
	w.lintDanglingTaskLinks(result)

	// 3. Dangling task dependencies.
	w.lintDanglingTaskDeps(result)

	// 4. Dangling citation references.
	w.lintDanglingCitations(result)

	// 5. Orphan specs (no incoming links, no tasks).
	w.lintOrphanSpecs(result)

	// 6. Orphan tasks (parent spec missing).
	w.lintOrphanTasks(result)

	// 7. System-field drift (frontmatter vs SQLite).
	w.lintSystemFieldDrift(result)

	// 8. Stale tidy.
	w.lintStaleTidy(result)

	// 9. Missing or trivial summaries.
	w.lintMissingSummaries(result)

	// 10. Dependency cycles.
	w.lintDepCycles(result)

	// 11. Rejected files.
	w.lintRejectedFiles(result)

	// 12. KB integrity (missing source files, hash mismatch).
	w.lintKBIntegrity(result)

	// 13. Citations pointing to nonexistent KB chunks.
	w.lintCitationChunks(result)

	// Count totals.
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			result.Counts.Errors++
		case "warning":
			result.Counts.Warnings++
		}
	}

	return result, nil
}

func (w *Workspace) addIssue(r *LintResult, severity, category, id, path, msg string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: severity,
		Category: category,
		ID:       id,
		Path:     path,
		Message:  msg,
	})
}

// lintDanglingSpecLinks checks for spec links that reference nonexistent specs.
func (w *Workspace) lintDanglingSpecLinks(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT sl.from_spec, sl.to_spec
		FROM spec_links sl
		LEFT JOIN specs s ON s.id = sl.to_spec
		WHERE s.id IS NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var from, to string
		rows.Scan(&from, &to)
		w.addIssue(r, "error", "dangling_reference", from, "",
			fmt.Sprintf("spec link from %s references nonexistent %s", from, to))
	}
}

// lintDanglingTaskLinks checks for task links that reference nonexistent tasks.
func (w *Workspace) lintDanglingTaskLinks(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT tl.from_task, tl.to_task
		FROM task_links tl
		LEFT JOIN tasks t ON t.id = tl.to_task
		WHERE t.id IS NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var from, to string
		rows.Scan(&from, &to)
		w.addIssue(r, "error", "dangling_reference", from, "",
			fmt.Sprintf("task link from %s references nonexistent %s", from, to))
	}
}

// lintDanglingTaskDeps checks for task dependencies on nonexistent tasks.
func (w *Workspace) lintDanglingTaskDeps(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT td.blocked_task, td.blocker_task
		FROM task_dependencies td
		LEFT JOIN tasks t ON t.id = td.blocker_task
		WHERE t.id IS NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var blocked, blocker string
		rows.Scan(&blocked, &blocker)
		w.addIssue(r, "error", "dangling_reference", blocked, "",
			fmt.Sprintf("task %s depends on nonexistent %s", blocked, blocker))
	}
}

// lintDanglingCitations checks for citations referencing nonexistent KB docs.
func (w *Workspace) lintDanglingCitations(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT c.from_kind, c.from_id, c.kb_doc_id
		FROM citations c
		LEFT JOIN kb_docs d ON d.id = c.kb_doc_id
		WHERE d.id IS NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var kind, id, kbID string
		rows.Scan(&kind, &id, &kbID)
		w.addIssue(r, "error", "dangling_reference", id, "",
			fmt.Sprintf("%s %s cites nonexistent KB doc %s", kind, id, kbID))
	}
}

// lintOrphanSpecs reports specs with no incoming links and no tasks.
func (w *Workspace) lintOrphanSpecs(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT s.id, s.title
		FROM specs s
		WHERE s.id NOT IN (SELECT to_spec FROM spec_links)
		  AND s.id NOT IN (SELECT from_spec FROM spec_links)
		  AND s.id NOT IN (SELECT DISTINCT spec_id FROM tasks)`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title string
		rows.Scan(&id, &title)
		w.addIssue(r, "warning", "orphan_spec", id, "",
			fmt.Sprintf("spec %s (%s) has no links and no tasks", id, title))
	}
}

// lintOrphanTasks reports tasks whose parent spec no longer exists.
func (w *Workspace) lintOrphanTasks(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT t.id, t.title, t.spec_id
		FROM tasks t
		LEFT JOIN specs s ON s.id = t.spec_id
		WHERE s.id IS NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, specID string
		rows.Scan(&id, &title, &specID)
		w.addIssue(r, "error", "orphan_task", id, "",
			fmt.Sprintf("task %s (%s) references missing parent spec %s", id, title, specID))
	}
}

// lintSystemFieldDrift compares frontmatter system-managed fields with SQLite.
func (w *Workspace) lintSystemFieldDrift(r *LintResult) {
	// Check specs.
	specs, err := w.ListSpecs(ListSpecsFilter{})
	if err != nil {
		return
	}
	for _, s := range specs {
		absPath := filepath.Join(w.Root, s.Path)
		data, err := os.ReadFile(absPath)
		if err != nil {
			w.addIssue(r, "error", "missing_file", s.ID, s.Path,
				fmt.Sprintf("spec %s file missing: %s", s.ID, s.Path))
			continue
		}
		doc, err := frontmatter.Parse(string(data))
		if err != nil {
			w.addIssue(r, "error", "parse_error", s.ID, s.Path,
				fmt.Sprintf("spec %s has unparseable frontmatter", s.ID))
			continue
		}
		fm, err := frontmatter.DecodeSpec(doc.RawFrontmatter)
		if err != nil {
			w.addIssue(r, "error", "parse_error", s.ID, s.Path,
				fmt.Sprintf("spec %s frontmatter decode error: %v", s.ID, err))
			continue
		}

		// Check system-managed linked_specs field.
		dbLinks, _ := w.getSpecLinks(s.ID)
		if !stringSlicesEqual(fm.LinkedSpecs, dbLinks) {
			w.addIssue(r, "warning", "system_field_drift", s.ID, s.Path,
				fmt.Sprintf("spec %s linked_specs frontmatter disagrees with SQLite", s.ID))
		}
	}

	// Check tasks.
	tasks, err := w.ListTasks(ListTasksFilter{})
	if err != nil {
		return
	}
	for _, t := range tasks {
		absPath := filepath.Join(w.Root, t.Path)
		data, err := os.ReadFile(absPath)
		if err != nil {
			w.addIssue(r, "error", "missing_file", t.ID, t.Path,
				fmt.Sprintf("task %s file missing: %s", t.ID, t.Path))
			continue
		}
		doc, err := frontmatter.Parse(string(data))
		if err != nil {
			w.addIssue(r, "error", "parse_error", t.ID, t.Path,
				fmt.Sprintf("task %s has unparseable frontmatter", t.ID))
			continue
		}
		fm, err := frontmatter.DecodeTask(doc.RawFrontmatter)
		if err != nil {
			w.addIssue(r, "error", "parse_error", t.ID, t.Path,
				fmt.Sprintf("task %s frontmatter decode error: %v", t.ID, err))
			continue
		}

		// Check linked_tasks.
		dbLinks, _ := w.getTaskLinks(t.ID)
		if !stringSlicesEqual(fm.LinkedTasks, dbLinks) {
			w.addIssue(r, "warning", "system_field_drift", t.ID, t.Path,
				fmt.Sprintf("task %s linked_tasks frontmatter disagrees with SQLite", t.ID))
		}

		// Check depends_on.
		dbDeps, _ := w.getTaskDependencies(t.ID)
		if !stringSlicesEqual(fm.DependsOn, dbDeps) {
			w.addIssue(r, "warning", "system_field_drift", t.ID, t.Path,
				fmt.Sprintf("task %s depends_on frontmatter disagrees with SQLite", t.ID))
		}
	}
}

// lintStaleTidy reports if last_tidy_at is older than 7 days.
func (w *Workspace) lintStaleTidy(r *LintResult) {
	val, err := w.DB.GetMeta("last_tidy_at")
	if err != nil {
		return
	}
	lastTidy, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return
	}
	if time.Since(lastTidy) > 7*24*time.Hour {
		w.addIssue(r, "warning", "stale_tidy", "", "",
			fmt.Sprintf("last tidy was %s (%s ago); consider running specd tidy",
				lastTidy.Format("2006-01-02"), formatDuration(time.Since(lastTidy))))
	}
}

// lintMissingSummaries checks for empty or trivially short summaries.
func (w *Workspace) lintMissingSummaries(r *LintResult) {
	// Specs.
	rows, err := w.DB.Query("SELECT id, summary FROM specs")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, summary string
		rows.Scan(&id, &summary)
		trimmed := strings.TrimSpace(summary)
		if trimmed == "" || !strings.Contains(trimmed, " ") {
			w.addIssue(r, "warning", "missing_summary", id, "",
				fmt.Sprintf("spec %s has a missing or trivial summary", id))
		}
	}

	// Tasks.
	rows2, err := w.DB.Query("SELECT id, summary FROM tasks")
	if err != nil {
		return
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, summary string
		rows2.Scan(&id, &summary)
		trimmed := strings.TrimSpace(summary)
		if trimmed == "" || !strings.Contains(trimmed, " ") {
			w.addIssue(r, "warning", "missing_summary", id, "",
				fmt.Sprintf("task %s has a missing or trivial summary", id))
		}
	}
}

// lintDepCycles detects dependency cycles among tasks.
func (w *Workspace) lintDepCycles(r *LintResult) {
	// Build adjacency list: blocker -> [blocked tasks].
	rows, err := w.DB.Query("SELECT blocker_task, blocked_task FROM task_dependencies")
	if err != nil {
		return
	}
	defer rows.Close()

	graph := map[string][]string{}
	nodes := map[string]bool{}
	for rows.Next() {
		var blocker, blocked string
		rows.Scan(&blocker, &blocked)
		graph[blocker] = append(graph[blocker], blocked)
		nodes[blocker] = true
		nodes[blocked] = true
	}

	// DFS cycle detection with path tracking.
	white := 0 // unvisited
	gray := 1  // in current path
	black := 2 // fully explored
	color := map[string]int{}
	parent := map[string]string{}

	var cyclePaths [][]string

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = gray
		for _, v := range graph[u] {
			if color[v] == gray {
				// Found cycle — trace back.
				cycle := []string{v, u}
				cur := u
				for cur != v {
					cur = parent[cur]
					cycle = append(cycle, cur)
				}
				// Reverse for readable order.
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}
				cyclePaths = append(cyclePaths, cycle)
			} else if color[v] == white {
				parent[v] = u
				dfs(v)
			}
		}
		color[u] = black
	}

	for node := range nodes {
		if color[node] == white {
			dfs(node)
		}
	}

	for _, cycle := range cyclePaths {
		w.addIssue(r, "error", "dependency_cycle", cycle[0], "",
			fmt.Sprintf("dependency cycle: %s", strings.Join(cycle, " -> ")))
	}
}

// lintRejectedFiles reports files in the rejected_files table.
func (w *Workspace) lintRejectedFiles(r *LintResult) {
	rows, err := w.DB.Query("SELECT path, reason FROM rejected_files")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var path, reason string
		rows.Scan(&path, &reason)
		w.addIssue(r, "warning", "rejected_file", "", path,
			fmt.Sprintf("rejected file: %s (%s)", path, reason))
	}
}

// lintKBIntegrity checks KB docs for missing source files and hash mismatches.
func (w *Workspace) lintKBIntegrity(r *LintResult) {
	docs, err := w.KBList(KBListFilter{})
	if err != nil {
		return
	}

	for _, doc := range docs {
		absPath := filepath.Join(w.Root, doc.Path)
		data, err := os.ReadFile(absPath)
		if err != nil {
			w.addIssue(r, "error", "kb_missing_file", doc.ID, doc.Path,
				fmt.Sprintf("KB doc %s source file missing: %s", doc.ID, doc.Path))
			continue
		}

		h := hash.Bytes(data)
		if h != doc.ContentHash {
			w.addIssue(r, "warning", "kb_hash_mismatch", doc.ID, doc.Path,
				fmt.Sprintf("KB doc %s content hash mismatch (file changed outside specd)", doc.ID))
		}
	}
}

// lintCitationChunks checks citations pointing to nonexistent KB chunks.
func (w *Workspace) lintCitationChunks(r *LintResult) {
	rows, err := w.DB.Query(`
		SELECT c.from_kind, c.from_id, c.kb_doc_id, c.chunk_position
		FROM citations c
		LEFT JOIN kb_chunks k ON k.doc_id = c.kb_doc_id AND k.position = c.chunk_position
		WHERE k.id IS NULL AND c.kb_doc_id IN (SELECT id FROM kb_docs)`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var kind, id, kbID string
		var pos int
		rows.Scan(&kind, &id, &kbID, &pos)
		w.addIssue(r, "error", "dangling_citation", id, "",
			fmt.Sprintf("%s %s cites nonexistent chunk %d in %s", kind, id, pos, kbID))
	}
}

// Tidy runs lint and updates last_tidy_at.
func (w *Workspace) Tidy() (*LintResult, error) {
	result, err := w.Lint()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := w.DB.SetMeta("last_tidy_at", now); err != nil {
		return nil, fmt.Errorf("update last_tidy_at: %w", err)
	}

	return result, nil
}

// TidyReminder returns a reminder string if last_tidy_at is older than 7 days,
// or nil if tidy is recent enough.
func (w *Workspace) TidyReminder() *string {
	val, err := w.DB.GetMeta("last_tidy_at")
	if err != nil {
		return nil
	}
	lastTidy, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return nil
	}
	if time.Since(lastTidy) > 7*24*time.Hour {
		msg := fmt.Sprintf("Last tidy was %s ago. Consider running specd tidy.",
			formatDuration(time.Since(lastTidy)))
		return &msg
	}
	return nil
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// stringSlicesEqual compares two string slices for equality, treating nil
// and empty as equivalent.
func stringSlicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// specIDPattern matches SPEC-N directory names.
var specIDPattern = regexp.MustCompile(`^SPEC-(\d+)-`)

// taskIDPattern matches TASK-N file names.
var taskIDPattern = regexp.MustCompile(`^TASK-(\d+)-`)

// kbIDPattern matches KB-N file names.
var kbIDPattern = regexp.MustCompile(`^KB-(\d+)-`)

// parseIDNumber extracts the numeric part from an ID like "SPEC-42".
func parseIDNumber(id string) int {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[1])
	return n
}
