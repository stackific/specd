package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	criteriaCmd := &cobra.Command{
		Use:   "criteria",
		Short: "Manage task acceptance criteria",
	}

	// criteria list <task-id>
	criteriaCmd.AddCommand(&cobra.Command{
		Use:   "list <task-id>",
		Short: "List acceptance criteria for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			criteria, err := w.ListCriteria(args[0])
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(criteria)
			} else {
				for _, c := range criteria {
					check := "[ ]"
					if c.Checked {
						check = "[x]"
					}
					fmt.Printf("%d. %s %s\n", c.Position, check, c.Text)
				}
				if len(criteria) == 0 {
					fmt.Println("No acceptance criteria.")
				}
			}
			return nil
		},
	})

	// criteria add <task-id> "<text>"
	criteriaCmd.AddCommand(&cobra.Command{
		Use:   "add <task-id> <text>",
		Short: "Add an acceptance criterion to a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			c, err := w.AddCriterion(args[0], args[1])
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(c)
			} else {
				fmt.Printf("Added criterion %d to %s\n", c.Position, c.TaskID)
			}
			return nil
		},
	})

	// criteria check <task-id> <position>
	criteriaCmd.AddCommand(&cobra.Command{
		Use:   "check <task-id> <position>",
		Short: "Mark a criterion as done",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			pos, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("position must be a number: %w", err)
			}

			if err := w.CheckCriterion(args[0], pos); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{"task_id": args[0], "position": pos, "checked": true})
			} else {
				fmt.Printf("Checked criterion %d on %s\n", pos, args[0])
			}
			return nil
		},
	})

	// criteria uncheck <task-id> <position>
	criteriaCmd.AddCommand(&cobra.Command{
		Use:   "uncheck <task-id> <position>",
		Short: "Mark a criterion as not done",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			pos, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("position must be a number: %w", err)
			}

			if err := w.UncheckCriterion(args[0], pos); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{"task_id": args[0], "position": pos, "checked": false})
			} else {
				fmt.Printf("Unchecked criterion %d on %s\n", pos, args[0])
			}
			return nil
		},
	})

	// criteria remove <task-id> <position>
	criteriaCmd.AddCommand(&cobra.Command{
		Use:   "remove <task-id> <position>",
		Short: "Remove a criterion from a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			pos, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("position must be a number: %w", err)
			}

			if err := w.RemoveCriterion(args[0], pos); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{"task_id": args[0], "position": pos, "removed": true})
			} else {
				fmt.Printf("Removed criterion %d from %s\n", pos, args[0])
			}
			return nil
		},
	})

	rootCmd.AddCommand(criteriaCmd)
}
