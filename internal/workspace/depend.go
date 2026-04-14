package workspace

import (
	"fmt"

	"github.com/stackific/specd/internal/frontmatter"
)

// Depend declares that taskID depends on (is blocked by) each of the blockerIDs.
// Rejects if any dependency would create a cycle. Updates SQLite and frontmatter.
func (w *Workspace) Depend(taskID string, blockerIDs []string) error {
	return w.WithLock(func() error {
		// Verify all tasks exist.
		if _, err := w.ReadTask(taskID); err != nil {
			return err
		}
		for _, bid := range blockerIDs {
			if _, err := w.ReadTask(bid); err != nil {
				return err
			}
		}

		// Check for cycles before inserting.
		for _, bid := range blockerIDs {
			if err := w.checkDepCycle(bid, taskID); err != nil {
				return err
			}
		}

		for _, bid := range blockerIDs {
			_, err := w.DB.Exec(
				"INSERT OR IGNORE INTO task_dependencies (blocker_task, blocked_task) VALUES (?, ?)",
				bid, taskID)
			if err != nil {
				return fmt.Errorf("insert dependency: %w", err)
			}
		}

		return w.syncTaskDependsOn(taskID)
	})
}

// Undepend removes dependencies from taskID on the given blockerIDs.
// Updates SQLite and frontmatter.
func (w *Workspace) Undepend(taskID string, blockerIDs []string) error {
	return w.WithLock(func() error {
		for _, bid := range blockerIDs {
			w.DB.Exec(
				"DELETE FROM task_dependencies WHERE blocker_task = ? AND blocked_task = ?",
				bid, taskID)
		}
		return w.syncTaskDependsOn(taskID)
	})
}

// checkDepCycle detects whether adding an edge from blocker -> blocked
// would create a cycle. It does a DFS from blocked following existing
// dependencies to see if it can reach blocker.
func (w *Workspace) checkDepCycle(blocker, blocked string) error {
	visited := map[string]bool{}
	stack := []string{blocked}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current == blocker {
			return fmt.Errorf("dependency cycle detected: %s -> %s would create a cycle", blocker, blocked)
		}

		if visited[current] {
			continue
		}
		visited[current] = true

		// Find what current blocks (current is a blocker_task for others).
		rows, err := w.DB.Query(
			"SELECT blocked_task FROM task_dependencies WHERE blocker_task = ?", current)
		if err != nil {
			return err
		}
		for rows.Next() {
			var next string
			rows.Scan(&next)
			stack = append(stack, next)
		}
		rows.Close()
	}

	return nil
}

// syncTaskDependsOn reads current dependencies from SQLite and updates
// the task's depends_on frontmatter field.
func (w *Workspace) syncTaskDependsOn(taskID string) error {
	task, err := w.ReadTask(taskID)
	if err != nil {
		return err
	}

	deps, err := w.getTaskDependencies(taskID)
	if err != nil {
		return err
	}

	return w.rewriteTaskFrontmatter(task, func(fm *frontmatter.TaskFrontmatter) {
		fm.DependsOn = deps
	})
}

// getTaskDependencies returns the IDs of tasks that block the given task.
func (w *Workspace) getTaskDependencies(taskID string) ([]string, error) {
	rows, err := w.DB.Query(
		"SELECT blocker_task FROM task_dependencies WHERE blocked_task = ? ORDER BY blocker_task",
		taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

