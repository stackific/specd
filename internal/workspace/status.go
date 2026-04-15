// Package workspace — status.go implements the project status command,
// returning counts by kind and status, tidy state, lint summary, and trash stats.
package workspace

import (
	"fmt"
	"time"
)

// StatusResult holds the full project status report.
type StatusResult struct {
	Specs    StatusSpecs    `json:"specs"`
	Tasks    StatusTasks    `json:"tasks"`
	KB       StatusKB       `json:"kb"`
	Trash    StatusTrash    `json:"trash"`
	Tidy     StatusTidy     `json:"tidy"`
	Lint     *StatusLint    `json:"lint,omitempty"` // only if --detailed
	Rejected int            `json:"rejected_files"`
}

// StatusSpecs holds spec counts.
type StatusSpecs struct {
	Total      int `json:"total"`
	Business   int `json:"business"`
	Technical  int `json:"technical"`
	NonTech    int `json:"non_technical"`
}

// StatusTasks holds task counts by status.
type StatusTasks struct {
	Total               int `json:"total"`
	Backlog             int `json:"backlog"`
	Todo                int `json:"todo"`
	InProgress          int `json:"in_progress"`
	Blocked             int `json:"blocked"`
	PendingVerification int `json:"pending_verification"`
	Done                int `json:"done"`
	Cancelled           int `json:"cancelled"`
	WontFix             int `json:"wontfix"`
}

// StatusKB holds KB document counts.
type StatusKB struct {
	Total      int `json:"total"`
	Markdown   int `json:"markdown"`
	HTML       int `json:"html"`
	PDF        int `json:"pdf"`
	Text       int `json:"text"`
	Chunks     int `json:"chunks"`
	Connections int `json:"connections"`
}

// StatusTrash holds trash item counts.
type StatusTrash struct {
	Total int `json:"total"`
	Specs int `json:"specs"`
	Tasks int `json:"tasks"`
	KB    int `json:"kb"`
}

// StatusTidy holds tidy timing info.
type StatusTidy struct {
	LastTidyAt string  `json:"last_tidy_at"`
	DaysAgo    int     `json:"days_ago"`
	Stale      bool    `json:"stale"`
	Reminder   *string `json:"reminder,omitempty"`
}

// StatusLint holds a lint summary (only in detailed mode).
type StatusLint struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// Status returns a full project status report.
func (w *Workspace) Status(detailed bool) (*StatusResult, error) {
	result := &StatusResult{}

	// Spec counts.
	w.DB.QueryRow("SELECT COUNT(*) FROM specs").Scan(&result.Specs.Total)
	w.DB.QueryRow("SELECT COUNT(*) FROM specs WHERE type = 'business'").Scan(&result.Specs.Business)
	w.DB.QueryRow("SELECT COUNT(*) FROM specs WHERE type = 'technical'").Scan(&result.Specs.Technical)
	w.DB.QueryRow("SELECT COUNT(*) FROM specs WHERE type = 'non-technical'").Scan(&result.Specs.NonTech)

	// Task counts by status.
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&result.Tasks.Total)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'backlog'").Scan(&result.Tasks.Backlog)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'todo'").Scan(&result.Tasks.Todo)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'in_progress'").Scan(&result.Tasks.InProgress)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'blocked'").Scan(&result.Tasks.Blocked)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'pending_verification'").Scan(&result.Tasks.PendingVerification)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'done'").Scan(&result.Tasks.Done)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'cancelled'").Scan(&result.Tasks.Cancelled)
	w.DB.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'wontfix'").Scan(&result.Tasks.WontFix)

	// KB counts.
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_docs").Scan(&result.KB.Total)
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_docs WHERE source_type = 'md'").Scan(&result.KB.Markdown)
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_docs WHERE source_type = 'html'").Scan(&result.KB.HTML)
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_docs WHERE source_type = 'pdf'").Scan(&result.KB.PDF)
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_docs WHERE source_type = 'txt'").Scan(&result.KB.Text)
	w.DB.QueryRow("SELECT COUNT(*) FROM kb_chunks").Scan(&result.KB.Chunks)
	w.DB.QueryRow("SELECT COUNT(*) FROM chunk_connections").Scan(&result.KB.Connections)

	// Trash counts.
	w.DB.QueryRow("SELECT COUNT(*) FROM trash").Scan(&result.Trash.Total)
	w.DB.QueryRow("SELECT COUNT(*) FROM trash WHERE kind = 'spec'").Scan(&result.Trash.Specs)
	w.DB.QueryRow("SELECT COUNT(*) FROM trash WHERE kind = 'task'").Scan(&result.Trash.Tasks)
	w.DB.QueryRow("SELECT COUNT(*) FROM trash WHERE kind = 'kb'").Scan(&result.Trash.KB)

	// Rejected files.
	w.DB.QueryRow("SELECT COUNT(*) FROM rejected_files").Scan(&result.Rejected)

	// Tidy info.
	val, err := w.DB.GetMeta("last_tidy_at")
	if err == nil {
		result.Tidy.LastTidyAt = val
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			result.Tidy.DaysAgo = int(time.Since(t).Hours() / 24)
			result.Tidy.Stale = time.Since(t) > 7*24*time.Hour
		}
	}
	result.Tidy.Reminder = w.TidyReminder()

	// Detailed mode: run lint.
	if detailed {
		lint, err := w.Lint()
		if err == nil {
			result.Lint = &StatusLint{
				Errors:   lint.Counts.Errors,
				Warnings: lint.Counts.Warnings,
			}
		}
	}

	return result, nil
}

// FormatStatus returns a human-readable status string.
func FormatStatus(s *StatusResult) string {
	out := fmt.Sprintf("Specs: %d (%d business, %d technical, %d non-technical)\n",
		s.Specs.Total, s.Specs.Business, s.Specs.Technical, s.Specs.NonTech)

	out += fmt.Sprintf("Tasks: %d total\n", s.Tasks.Total)
	out += fmt.Sprintf("  backlog: %d, todo: %d, in_progress: %d, blocked: %d\n",
		s.Tasks.Backlog, s.Tasks.Todo, s.Tasks.InProgress, s.Tasks.Blocked)
	out += fmt.Sprintf("  pending_verification: %d, done: %d, cancelled: %d, wontfix: %d\n",
		s.Tasks.PendingVerification, s.Tasks.Done, s.Tasks.Cancelled, s.Tasks.WontFix)

	out += fmt.Sprintf("KB: %d docs (%d md, %d html, %d pdf, %d txt), %d chunks, %d connections\n",
		s.KB.Total, s.KB.Markdown, s.KB.HTML, s.KB.PDF, s.KB.Text,
		s.KB.Chunks, s.KB.Connections)

	if s.Trash.Total > 0 {
		out += fmt.Sprintf("Trash: %d items (%d specs, %d tasks, %d kb)\n",
			s.Trash.Total, s.Trash.Specs, s.Trash.Tasks, s.Trash.KB)
	}

	if s.Rejected > 0 {
		out += fmt.Sprintf("Rejected files: %d\n", s.Rejected)
	}

	if s.Tidy.Stale {
		out += fmt.Sprintf("Tidy: STALE (last %dd ago)\n", s.Tidy.DaysAgo)
	} else {
		out += fmt.Sprintf("Tidy: OK (last %dd ago)\n", s.Tidy.DaysAgo)
	}

	if s.Lint != nil {
		out += fmt.Sprintf("Lint: %d errors, %d warnings\n", s.Lint.Errors, s.Lint.Warnings)
	}

	return out
}
