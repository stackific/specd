// Package cli — link.go registers the link, unlink, and candidates
// commands for managing undirected relationships between specs or tasks.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	// link <from-id> <to-id>...
	rootCmd.AddCommand(&cobra.Command{
		Use:   "link <from-id> <to-id>...",
		Short: "Link specs or tasks together",
		Long:  "Create undirected links between specs (SPEC-to-SPEC) or tasks (TASK-to-TASK).",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			fromID := args[0]
			for _, toID := range args[1:] {
				if err := w.Link(fromID, toID); err != nil {
					return err
				}
				if !jsonOutput {
					fmt.Printf("Linked %s <-> %s\n", fromID, toID)
				}
			}

			if jsonOutput {
				printJSON(map[string]any{
					"from":   fromID,
					"linked": args[1:],
				})
			}
			return nil
		},
	})

	// unlink <from-id> <to-id>...
	rootCmd.AddCommand(&cobra.Command{
		Use:   "unlink <from-id> <to-id>...",
		Short: "Remove links between specs or tasks",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			fromID := args[0]
			for _, toID := range args[1:] {
				if err := w.Unlink(fromID, toID); err != nil {
					return err
				}
				if !jsonOutput {
					fmt.Printf("Unlinked %s <-> %s\n", fromID, toID)
				}
			}

			if jsonOutput {
				printJSON(map[string]any{
					"from":     fromID,
					"unlinked": args[1:],
				})
			}
			return nil
		},
	})

	// candidates <spec-id|task-id>
	var candidatesLimit int
	candidatesCmd := &cobra.Command{
		Use:   "candidates <id>",
		Short: "Find link candidates for a spec or task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.Candidates(args[0], candidatesLimit)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				if len(result.Specs) > 0 {
					fmt.Println("Spec candidates:")
					for _, c := range result.Specs {
						fmt.Printf("  %-10s (%.0f%%) %s\n", c.ID, c.Score*100, c.Title)
					}
				}
				if len(result.Tasks) > 0 {
					fmt.Println("Task candidates:")
					for _, c := range result.Tasks {
						fmt.Printf("  %-10s (%.0f%%) %s\n", c.ID, c.Score*100, c.Title)
					}
				}
				if len(result.KBChunks) > 0 {
					fmt.Println("KB chunk candidates:")
					for _, c := range result.KBChunks {
						fmt.Printf("  %s chunk %d (%.2f) %s\n", c.DocID, c.ChunkPosition, c.Score, c.DocTitle)
						text := c.Text
						if len(text) > 100 {
							text = text[:100] + "..."
						}
						fmt.Printf("    %s\n", text)
					}
				}
				if len(result.Specs) == 0 && len(result.Tasks) == 0 && len(result.KBChunks) == 0 {
					fmt.Println("No candidates found.")
				}
			}
			return nil
		},
	}
	candidatesCmd.Flags().IntVar(&candidatesLimit, "limit", 20, "max candidates to return")
	rootCmd.AddCommand(candidatesCmd)
}
