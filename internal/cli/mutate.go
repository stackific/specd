// Package cli — mutate.go registers the update, move, rename, delete, and
// reorder commands for specs and tasks.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	// update <spec-id|task-id>
	var (
		updateTitle   string
		updateType    string
		updateSummary string
		updateBody    string
	)

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a spec or task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			id := args[0]

			if isSpecID(id) {
				input := workspace.UpdateSpecInput{}
				if cmd.Flags().Changed("title") {
					input.Title = &updateTitle
				}
				if cmd.Flags().Changed("type") {
					input.Type = &updateType
				}
				if cmd.Flags().Changed("summary") {
					input.Summary = &updateSummary
				}
				if cmd.Flags().Changed("body") {
					input.Body = &updateBody
				}

				if err := w.UpdateSpec(id, input); err != nil {
					return err
				}
			} else if isTaskID(id) {
				input := workspace.UpdateTaskInput{}
				if cmd.Flags().Changed("title") {
					input.Title = &updateTitle
				}
				if cmd.Flags().Changed("summary") {
					input.Summary = &updateSummary
				}
				if cmd.Flags().Changed("body") {
					input.Body = &updateBody
				}

				if err := w.UpdateTask(id, input); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid ID: %s", id)
			}

			if jsonOutput {
				printJSON(map[string]string{"updated": id})
			} else {
				fmt.Printf("Updated %s\n", id)
			}
			return nil
		},
	}

	updateCmd.Flags().StringVar(&updateTitle, "title", "", "new title")
	updateCmd.Flags().StringVar(&updateType, "type", "", "new type (spec only)")
	updateCmd.Flags().StringVar(&updateSummary, "summary", "", "new summary")
	updateCmd.Flags().StringVar(&updateBody, "body", "", "new body")
	rootCmd.AddCommand(updateCmd)

	// move <task-id> --status <status>
	var moveStatus string

	moveCmd := &cobra.Command{
		Use:   "move <task-id> --status <status>",
		Short: "Change a task's status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			if err := w.MoveTask(args[0], moveStatus); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]string{"moved": args[0], "status": moveStatus})
			} else {
				fmt.Printf("Moved %s to %s\n", args[0], moveStatus)
			}
			return nil
		},
	}

	moveCmd.Flags().StringVar(&moveStatus, "status", "", "target status (required)")
	moveCmd.MarkFlagRequired("status")
	rootCmd.AddCommand(moveCmd)

	// rename <spec-id|task-id> --title "<new title>"
	var renameTitle string

	renameCmd := &cobra.Command{
		Use:   "rename <id> --title <title>",
		Short: "Rename a spec or task (updates slug and file/folder name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			id := args[0]
			if isSpecID(id) {
				if err := w.RenameSpec(id, renameTitle); err != nil {
					return err
				}
			} else if isTaskID(id) {
				if err := w.RenameTask(id, renameTitle); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid ID: %s", id)
			}

			if jsonOutput {
				printJSON(map[string]string{"renamed": id, "title": renameTitle})
			} else {
				fmt.Printf("Renamed %s to %q\n", id, renameTitle)
			}
			return nil
		},
	}

	renameCmd.Flags().StringVar(&renameTitle, "title", "", "new title (required)")
	renameCmd.MarkFlagRequired("title")
	rootCmd.AddCommand(renameCmd)

	// delete <spec-id|task-id>
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Soft-delete a spec or task (moves to trash)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			id := args[0]
			if isSpecID(id) {
				if err := w.DeleteSpec(id); err != nil {
					return err
				}
			} else if isTaskID(id) {
				if err := w.DeleteTask(id); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid ID: %s", id)
			}

			if jsonOutput {
				printJSON(map[string]string{"deleted": id})
			} else {
				fmt.Printf("Deleted %s (moved to trash)\n", id)
			}
			return nil
		},
	}

	rootCmd.AddCommand(deleteCmd)

	// reorder spec|task <id> --before|--after|--to
	var (
		reorderBefore string
		reorderAfter  string
		reorderTo     int
	)

	reorderCmd := &cobra.Command{
		Use:   "reorder <spec|task> <id>",
		Short: "Change the position of a spec or task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			kind := args[0]
			id := args[1]

			var input workspace.ReorderInput
			switch {
			case cmd.Flags().Changed("before"):
				input = workspace.ReorderInput{Mode: workspace.ReorderBefore, TargetID: reorderBefore}
			case cmd.Flags().Changed("after"):
				input = workspace.ReorderInput{Mode: workspace.ReorderAfter, TargetID: reorderAfter}
			case cmd.Flags().Changed("to"):
				input = workspace.ReorderInput{Mode: workspace.ReorderTo, Position: reorderTo}
			default:
				return fmt.Errorf("specify --before, --after, or --to")
			}

			switch kind {
			case "spec":
				if err := w.ReorderSpec(id, input); err != nil {
					return err
				}
			case "task":
				if err := w.ReorderTask(id, input); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown kind %q (use 'spec' or 'task')", kind)
			}

			if jsonOutput {
				printJSON(map[string]string{"reordered": id})
			} else {
				fmt.Printf("Reordered %s\n", id)
			}
			return nil
		},
	}

	reorderCmd.Flags().StringVar(&reorderBefore, "before", "", "place before this ID")
	reorderCmd.Flags().StringVar(&reorderAfter, "after", "", "place after this ID")
	reorderCmd.Flags().IntVar(&reorderTo, "to", 0, "place at absolute position")
	rootCmd.AddCommand(reorderCmd)
}
