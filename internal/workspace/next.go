// Package workspace — next.go implements the `next` command which returns
// todo tasks sorted by readiness, criteria progress, and kanban position.
// It detects dependency cycles and reports them as errors.
package workspace

import (
	"fmt"
	"sort"
)

// NextTaskItem represents a task in the next-up queue with sequencing metadata.
type NextTaskItem struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	SpecID           string   `json:"spec_id"`
	Ready            bool     `json:"ready"`
	PartiallyDone    bool     `json:"partially_done"`
	CriteriaProgress float64  `json:"criteria_progress"`
	BlockedBy        []string `json:"blocked_by"`
}

// NextResult is the JSON response from the next command.
type NextResult struct {
	Tasks []NextTaskItem `json:"tasks"`
}

// Next returns todo tasks sorted for execution priority.
// Sort order (each key breaks ties of the previous):
//  1. Ready (all deps in done/cancelled/wontfix) before not-ready.
//  2. Within ready: partially-done (≥1 criterion checked) before fresh.
//  3. Within partially-done: higher criteria completion percentage first.
//  4. Within equally-done: kanban position ascending.
//
// Dependency cycles halt with an error that includes the cycle path.
func (w *Workspace) Next(specID string, limit int) (*NextResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Check for dependency cycles among all tasks.
	cyclePath, err := w.detectDepCycles()
	if err != nil {
		return nil, err
	}
	if cyclePath != "" {
		return nil, fmt.Errorf("dependency cycle detected: %s", cyclePath)
	}

	// Fetch todo tasks.
	filter := ListTasksFilter{Status: "todo"}
	if specID != "" {
		filter.SpecID = specID
	}
	tasks, err := w.ListTasks(filter)
	if err != nil {
		return nil, fmt.Errorf("list todo tasks: %w", err)
	}

	// Build NextTaskItems with readiness and criteria info.
	items := make([]NextTaskItem, 0, len(tasks))
	for _, t := range tasks {
		item := NextTaskItem{
			ID:     t.ID,
			Title:  t.Title,
			SpecID: t.SpecID,
		}

		// Check dependency readiness.
		deps, err := w.GetTaskDeps(t.ID)
		if err != nil {
			return nil, fmt.Errorf("get deps for %s: %w", t.ID, err)
		}

		item.Ready = true
		for _, d := range deps {
			if !d.Ready {
				item.Ready = false
				item.BlockedBy = append(item.BlockedBy, d.ID)
			}
		}
		if item.BlockedBy == nil {
			item.BlockedBy = []string{}
		}

		// Check criteria progress.
		criteria, err := w.ListCriteria(t.ID)
		if err != nil {
			return nil, fmt.Errorf("list criteria for %s: %w", t.ID, err)
		}

		if len(criteria) > 0 {
			checked := 0
			for _, c := range criteria {
				if c.Checked {
					checked++
				}
			}
			item.CriteriaProgress = float64(checked) / float64(len(criteria))
			item.PartiallyDone = checked > 0
		}

		items = append(items, item)
	}

	// Sort by the 4-level priority. Tasks share position from ListTasks
	// which already returns them ordered by position ASC, so we use the
	// slice index as the tiebreaker (preserves original position order).
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]

		// 1. Ready before not-ready.
		if a.Ready != b.Ready {
			return a.Ready
		}

		// 2. Partially-done before fresh.
		if a.PartiallyDone != b.PartiallyDone {
			return a.PartiallyDone
		}

		// 3. Higher criteria progress first.
		if a.CriteriaProgress != b.CriteriaProgress {
			return a.CriteriaProgress > b.CriteriaProgress
		}

		// 4. Kanban position (preserved by SliceStable from ListTasks order).
		return false
	})

	// Apply limit.
	if len(items) > limit {
		items = items[:limit]
	}

	return &NextResult{Tasks: items}, nil
}

// detectDepCycles scans the entire task_dependencies graph for cycles using
// iterative DFS with coloring (white/gray/black). Returns the cycle path
// as a string if found, or empty string if no cycles exist.
func (w *Workspace) detectDepCycles() (string, error) {
	// Load the full dependency graph.
	rows, err := w.DB.Query("SELECT blocker_task, blocked_task FROM task_dependencies")
	if err != nil {
		return "", fmt.Errorf("load dependencies: %w", err)
	}
	defer rows.Close()

	// Build adjacency list: task -> tasks it blocks.
	graph := map[string][]string{}
	nodes := map[string]bool{}
	for rows.Next() {
		var blocker, blocked string
		if err := rows.Scan(&blocker, &blocked); err != nil {
			return "", err
		}
		graph[blocker] = append(graph[blocker], blocked)
		nodes[blocker] = true
		nodes[blocked] = true
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", nil
	}

	// DFS with coloring: 0=white (unvisited), 1=gray (in stack), 2=black (done).
	color := map[string]int{}
	parent := map[string]string{}

	for node := range nodes {
		if color[node] != 0 {
			continue
		}

		stack := []string{node}
		for len(stack) > 0 {
			v := stack[len(stack)-1]

			if color[v] == 0 {
				color[v] = 1 // gray
				for _, next := range graph[v] {
					if color[next] == 1 {
						// Found a cycle — reconstruct the path.
						path := next
						cur := v
						for cur != next {
							path = cur + " -> " + path
							cur = parent[cur]
						}
						path = next + " -> " + path
						return path, nil
					}
					if color[next] == 0 {
						parent[next] = v
						stack = append(stack, next)
					}
				}
			} else {
				stack = stack[:len(stack)-1]
				color[v] = 2 // black
			}
		}
	}

	return "", nil
}
