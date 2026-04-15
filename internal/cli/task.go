// Package cli — task.go registers the new-task command with support for
// --link, --depends-on, --cite, and --dry-run flags.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	var (
		taskSpecID    string
		taskTitle     string
		taskSummary   string
		taskBody      string
		taskStatus    string
		taskLinks     []string
		taskDependsOn []string
		taskCites     []string
		taskDryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "new-task",
		Short: "Create a new task under a spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			if taskDryRun {
				response := map[string]any{
					"dry_run": true,
					"spec_id": taskSpecID,
					"title":   taskTitle,
					"summary": taskSummary,
				}
				if jsonOutput {
					printJSON(response)
				} else {
					fmt.Printf("[dry-run] Would create task: %s (under %s)\n", taskTitle, taskSpecID)
				}
				return nil
			}

			result, err := w.NewTask(workspace.NewTaskInput{
				SpecID:  taskSpecID,
				Title:   taskTitle,
				Summary: taskSummary,
				Body:    taskBody,
				Status:  taskStatus,
			})
			if err != nil {
				return err
			}

			// Apply links.
			for _, linkID := range taskLinks {
				if err := w.Link(result.ID, linkID); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: link %s: %v\n", linkID, err)
				}
			}

			// Apply dependencies.
			if len(taskDependsOn) > 0 {
				if err := w.Depend(result.ID, taskDependsOn); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: depend: %v\n", err)
				}
			}

			// Apply citations.
			for _, citeRef := range taskCites {
				parsed, err := workspace.ParseCitationRef(citeRef)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: cite %s: %v\n", citeRef, err)
					continue
				}
				if err := w.Cite(result.ID, []workspace.CitationInput{*parsed}); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: cite %s: %v\n", citeRef, err)
				}
			}

			if jsonOutput {
				printJSON(result)
			} else {
				fmt.Printf("Created %s at %s\n", result.ID, result.Path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&taskSpecID, "spec-id", "", "parent spec ID (required)")
	cmd.Flags().StringVar(&taskTitle, "title", "", "task title (required)")
	cmd.Flags().StringVar(&taskSummary, "summary", "", "one-line summary (required)")
	cmd.Flags().StringVar(&taskBody, "body", "", "markdown body")
	cmd.Flags().StringVar(&taskStatus, "status", "backlog", "initial status")
	cmd.Flags().StringSliceVar(&taskLinks, "link", nil, "link to TASK-N (repeatable)")
	cmd.Flags().StringSliceVar(&taskDependsOn, "depends-on", nil, "depends on TASK-N (repeatable)")
	cmd.Flags().StringSliceVar(&taskCites, "cite", nil, "cite KB-N:position (repeatable)")
	cmd.Flags().BoolVar(&taskDryRun, "dry-run", false, "preview without creating")
	cmd.MarkFlagRequired("spec-id")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("summary")
	rootCmd.AddCommand(cmd)
}
