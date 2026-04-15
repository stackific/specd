// Package cli — next.go registers the next command which returns todo tasks
// sorted by readiness, criteria progress, and kanban position.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var (
		nextLimit  int
		nextSpecID string
	)

	cmd := &cobra.Command{
		Use:   "next",
		Short: "Get next ready tasks sorted by priority",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.Next(nextSpecID, nextLimit)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				if len(result.Tasks) == 0 {
					fmt.Println("No todo tasks found.")
					return nil
				}
				for _, item := range result.Tasks {
					readyMark := "+"
					if !item.Ready {
						readyMark = "-"
					}
					progress := ""
					if item.PartiallyDone {
						progress = fmt.Sprintf(" (%.0f%%)", item.CriteriaProgress*100)
					}
					blocked := ""
					if len(item.BlockedBy) > 0 {
						blocked = fmt.Sprintf(" [blocked by %v]", item.BlockedBy)
					}
					fmt.Printf("[%s] %s  %s%s%s\n", readyMark, item.ID, item.Title, progress, blocked)
				}
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&nextLimit, "limit", 10, "max tasks to return")
	cmd.Flags().StringVar(&nextSpecID, "spec-id", "", "filter by parent spec")
	rootCmd.AddCommand(cmd)
}
