package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	// depend <task-id> --on <task-id>...
	var dependOn []string
	dependCmd := &cobra.Command{
		Use:   "depend <task-id>",
		Short: "Declare task dependencies",
		Long:  "Declare that a task depends on (is blocked by) other tasks.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(dependOn) == 0 {
				return fmt.Errorf("--on is required")
			}

			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			if err := w.Depend(args[0], dependOn); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{
					"task_id":    args[0],
					"depends_on": dependOn,
				})
			} else {
				for _, on := range dependOn {
					fmt.Printf("%s now depends on %s\n", args[0], on)
				}
			}
			return nil
		},
	}
	dependCmd.Flags().StringSliceVar(&dependOn, "on", nil, "blocker task IDs (required)")
	rootCmd.AddCommand(dependCmd)

	// undepend <task-id> --on <task-id>...
	var undependOn []string
	undependCmd := &cobra.Command{
		Use:   "undepend <task-id>",
		Short: "Remove task dependencies",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(undependOn) == 0 {
				return fmt.Errorf("--on is required")
			}

			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			if err := w.Undepend(args[0], undependOn); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{
					"task_id": args[0],
					"removed": undependOn,
				})
			} else {
				for _, on := range undependOn {
					fmt.Printf("%s no longer depends on %s\n", args[0], on)
				}
			}
			return nil
		},
	}
	undependCmd.Flags().StringSliceVar(&undependOn, "on", nil, "blocker task IDs to remove (required)")
	rootCmd.AddCommand(undependCmd)
}
