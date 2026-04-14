package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	var (
		taskSpecID  string
		taskTitle   string
		taskSummary string
		taskBody    string
		taskStatus  string
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
	cmd.MarkFlagRequired("spec-id")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("summary")
	rootCmd.AddCommand(cmd)
}
