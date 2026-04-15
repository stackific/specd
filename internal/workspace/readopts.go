// Package workspace — readopts.go provides exported helpers for the read
// command's optional enrichment flags: --with-links, --with-progress,
// and --with-deps. These query related data from SQLite without mutating.
package workspace

// LinkedSpec is a minimal spec reference returned by GetSpecLinks.
type LinkedSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// LinkedTask is a minimal task reference returned by GetTaskLinks.
type LinkedTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// SpecProgress summarises task completion for a spec.
type SpecProgress struct {
	Total     int     `json:"total"`
	Done      int     `json:"done"`
	Cancelled int     `json:"cancelled"`
	WontFix   int     `json:"wontfix"`
	Active    int     `json:"active"` // total - cancelled - wontfix
	Percent   float64 `json:"percent"`
}

// TaskDependency is a blocker task with its readiness status.
type TaskDependency struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Ready  bool   `json:"ready"` // true if done/cancelled/wontfix
}

// GetSpecLinks returns the specs linked to specID with titles.
func (w *Workspace) GetSpecLinks(specID string) ([]LinkedSpec, error) {
	ids, err := w.getSpecLinks(specID)
	if err != nil {
		return nil, err
	}

	var links []LinkedSpec
	for _, id := range ids {
		spec, err := w.ReadSpec(id)
		if err != nil {
			continue
		}
		links = append(links, LinkedSpec{ID: spec.ID, Title: spec.Title})
	}
	return links, nil
}

// GetTaskLinks returns the tasks linked to taskID with titles.
func (w *Workspace) GetTaskLinks(taskID string) ([]LinkedTask, error) {
	ids, err := w.getTaskLinks(taskID)
	if err != nil {
		return nil, err
	}

	var links []LinkedTask
	for _, id := range ids {
		task, err := w.ReadTask(id)
		if err != nil {
			continue
		}
		links = append(links, LinkedTask{ID: task.ID, Title: task.Title, Status: task.Status})
	}
	return links, nil
}

// GetSpecProgress computes task completion progress for a spec.
func (w *Workspace) GetSpecProgress(specID string) (*SpecProgress, error) {
	tasks, err := w.ListTasks(ListTasksFilter{SpecID: specID})
	if err != nil {
		return nil, err
	}

	p := &SpecProgress{Total: len(tasks)}
	for _, t := range tasks {
		switch t.Status {
		case "done":
			p.Done++
		case "cancelled":
			p.Cancelled++
		case "wontfix":
			p.WontFix++
		}
	}
	p.Active = p.Total - p.Cancelled - p.WontFix
	if p.Active > 0 {
		p.Percent = float64(p.Done) / float64(p.Active) * 100
	}
	return p, nil
}

// GetTaskDeps returns the tasks that block taskID with readiness info.
func (w *Workspace) GetTaskDeps(taskID string) ([]TaskDependency, error) {
	ids, err := w.getTaskDependencies(taskID)
	if err != nil {
		return nil, err
	}

	var deps []TaskDependency
	for _, id := range ids {
		task, err := w.ReadTask(id)
		if err != nil {
			continue
		}
		ready := task.Status == "done" || task.Status == "cancelled" || task.Status == "wontfix"
		deps = append(deps, TaskDependency{
			ID:     task.ID,
			Title:  task.Title,
			Status: task.Status,
			Ready:  ready,
		})
	}
	return deps, nil
}
