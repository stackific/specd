// Package cli — cite.go registers the cite and uncite commands for adding
// and removing KB chunk citations on specs and tasks.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	// cite <spec-id|task-id> <KB-N:position>...
	rootCmd.AddCommand(&cobra.Command{
		Use:   "cite <id> <KB-N:position>...",
		Short: "Add KB chunk citations to a spec or task",
		Long:  "Cite specific KB chunks from a spec or task. Format: KB-4:12 (doc KB-4, chunk position 12).",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			id := args[0]
			var refs []workspace.CitationInput
			for _, ref := range args[1:] {
				parsed, err := workspace.ParseCitationRef(ref)
				if err != nil {
					return err
				}
				refs = append(refs, *parsed)
			}

			if err := w.Cite(id, refs); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{
					"id":    id,
					"cited": args[1:],
				})
			} else {
				for _, ref := range args[1:] {
					fmt.Printf("Cited %s from %s\n", ref, id)
				}
			}
			return nil
		},
	})

	// uncite <spec-id|task-id> <KB-N:position>...
	rootCmd.AddCommand(&cobra.Command{
		Use:   "uncite <id> <KB-N:position>...",
		Short: "Remove KB chunk citations from a spec or task",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			id := args[0]
			var refs []workspace.CitationInput
			for _, ref := range args[1:] {
				parsed, err := workspace.ParseCitationRef(ref)
				if err != nil {
					return err
				}
				refs = append(refs, *parsed)
			}

			if err := w.Uncite(id, refs); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{
					"id":      id,
					"uncited": args[1:],
				})
			} else {
				for _, ref := range args[1:] {
					fmt.Printf("Uncited %s from %s\n", ref, id)
				}
			}
			return nil
		},
	})
}
