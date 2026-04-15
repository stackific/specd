// Package workspace — reorder.go implements position reordering for specs
// and tasks. Supports --before, --after, and --to positioning modes.
// Specs have a global position order; tasks are ordered per-status column.
package workspace

import "fmt"

// ReorderMode specifies how to reposition an item.
type ReorderMode int

const (
	// ReorderBefore places the item immediately before the target.
	ReorderBefore ReorderMode = iota
	// ReorderAfter places the item immediately after the target.
	ReorderAfter
	// ReorderTo places the item at an absolute position (0-based).
	ReorderTo
)

// ReorderInput holds the parameters for a reorder operation.
type ReorderInput struct {
	Mode     ReorderMode
	TargetID string // for Before/After — the ID to position relative to
	Position int    // for To — the absolute position
}

// ReorderSpec changes a spec's position in the global ordering.
func (w *Workspace) ReorderSpec(specID string, input ReorderInput) error {
	return w.WithLock(func() error {
		spec, err := w.ReadSpec(specID)
		if err != nil {
			return err
		}

		// Load all specs ordered by position.
		specs, err := w.ListSpecs(ListSpecsFilter{})
		if err != nil {
			return err
		}

		// Build ordered ID list, removing the target spec.
		var ordered []string
		for _, s := range specs {
			if s.ID != specID {
				ordered = append(ordered, s.ID)
			}
		}

		// Determine insertion index.
		insertIdx, err := resolveInsertIndex(ordered, input, spec.ID)
		if err != nil {
			return err
		}

		// Insert the spec at the new position.
		result := make([]string, 0, len(ordered)+1)
		result = append(result, ordered[:insertIdx]...)
		result = append(result, specID)
		result = append(result, ordered[insertIdx:]...)

		// Update positions.
		for i, id := range result {
			if _, err := w.DB.Exec("UPDATE specs SET position = ? WHERE id = ?", i, id); err != nil {
				return fmt.Errorf("update spec position: %w", err)
			}
		}
		return nil
	})
}

// ReorderTask changes a task's position within its status column.
func (w *Workspace) ReorderTask(taskID string, input ReorderInput) error {
	return w.WithLock(func() error {
		task, err := w.ReadTask(taskID)
		if err != nil {
			return err
		}

		// Load all tasks in the same status column.
		tasks, err := w.ListTasks(ListTasksFilter{Status: task.Status})
		if err != nil {
			return err
		}

		var ordered []string
		for _, t := range tasks {
			if t.ID != taskID {
				ordered = append(ordered, t.ID)
			}
		}

		insertIdx, err := resolveInsertIndex(ordered, input, task.ID)
		if err != nil {
			return err
		}

		result := make([]string, 0, len(ordered)+1)
		result = append(result, ordered[:insertIdx]...)
		result = append(result, taskID)
		result = append(result, ordered[insertIdx:]...)

		for i, id := range result {
			if _, err := w.DB.Exec("UPDATE tasks SET position = ? WHERE id = ?", i, id); err != nil {
				return fmt.Errorf("update task position: %w", err)
			}
		}
		return nil
	})
}

// resolveInsertIndex determines where to insert an item in an ordered list
// based on the reorder mode.
func resolveInsertIndex(ordered []string, input ReorderInput, selfID string) (int, error) {
	switch input.Mode {
	case ReorderBefore:
		for i, id := range ordered {
			if id == input.TargetID {
				return i, nil
			}
		}
		return 0, fmt.Errorf("target %s not found", input.TargetID)

	case ReorderAfter:
		for i, id := range ordered {
			if id == input.TargetID {
				return i + 1, nil
			}
		}
		return 0, fmt.Errorf("target %s not found", input.TargetID)

	case ReorderTo:
		pos := input.Position
		if pos < 0 {
			pos = 0
		}
		if pos > len(ordered) {
			pos = len(ordered)
		}
		return pos, nil

	default:
		return 0, fmt.Errorf("invalid reorder mode")
	}
}
