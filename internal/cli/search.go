// Package cli — search.go registers the search command for hybrid
// BM25 + trigram full-text search across specs, tasks, and KB documents.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var (
		searchKind  string
		searchLimit int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search specs, tasks, and KB documents",
		Long:  "Hybrid BM25 + trigram search across the workspace. Defaults to all kinds.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			results, err := w.Search(args[0], searchKind, searchLimit)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(results)
				return nil
			}

			printed := false
			if len(results.Specs) > 0 {
				fmt.Println("Specs:")
				for _, r := range results.Specs {
					fmt.Printf("  %-10s [%s] %s\n", r.ID, r.MatchType, r.Title)
				}
				printed = true
			}
			if len(results.Tasks) > 0 {
				fmt.Println("Tasks:")
				for _, r := range results.Tasks {
					fmt.Printf("  %-10s [%s] %s\n", r.ID, r.MatchType, r.Title)
				}
				printed = true
			}
			if len(results.KB) > 0 {
				fmt.Println("KB:")
				for _, r := range results.KB {
					fmt.Printf("  %-10s [%s] %s\n", r.ID, r.MatchType, r.Title)
				}
				printed = true
			}
			if !printed {
				fmt.Println("No results found.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&searchKind, "kind", "all", "filter by kind: spec, task, kb, or all")
	cmd.Flags().IntVar(&searchLimit, "limit", 20, "max results per kind")
	rootCmd.AddCommand(cmd)
}
