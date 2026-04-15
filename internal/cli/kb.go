package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	kbCmd := &cobra.Command{
		Use:   "kb",
		Short: "Knowledge base management",
	}
	rootCmd.AddCommand(kbCmd)

	// kb add
	var (
		addTitle string
		addNote  string
	)

	addCmd := &cobra.Command{
		Use:   "add <path-or-url>",
		Short: "Add a document to the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.KBAdd(workspace.KBAddInput{
				Source: args[0],
				Title:  addTitle,
				Note:   addNote,
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				fmt.Printf("Added %s (%s) at %s — %d chunks\n",
					result.ID, result.SourceType, result.Path, result.ChunkCount)
				if result.PageCount != nil {
					fmt.Printf("Pages: %d\n", *result.PageCount)
				}
			}
			return nil
		},
	}

	addCmd.Flags().StringVar(&addTitle, "title", "", "document title (defaults to filename)")
	addCmd.Flags().StringVar(&addNote, "note", "", "optional note about the document")
	kbCmd.AddCommand(addCmd)

	// kb list
	var listSourceType string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List KB documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			docs, err := w.KBList(workspace.KBListFilter{
				SourceType: listSourceType,
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(docs)
			} else {
				for _, d := range docs {
					fmt.Printf("%-10s [%s] %s\n", d.ID, d.SourceType, d.Title)
				}
				if len(docs) == 0 {
					fmt.Println("No KB documents found.")
				}
			}
			return nil
		},
	}

	listCmd.Flags().StringVar(&listSourceType, "source-type", "", "filter by source type: md, html, pdf, txt")
	kbCmd.AddCommand(listCmd)

	// kb read
	var readChunk int

	readCmd := &cobra.Command{
		Use:   "read <kb-id>",
		Short: "Read a KB document and its chunks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			var chunkPtr *int
			if cmd.Flags().Changed("chunk") {
				chunkPtr = &readChunk
			}

			result, err := w.KBRead(args[0], chunkPtr)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				printKBDoc(&result.Doc)
				fmt.Println()
				for _, c := range result.Chunks {
					fmt.Printf("--- Chunk %d (chars %d–%d)", c.Position, c.CharStart, c.CharEnd)
					if c.Page != nil {
						fmt.Printf(" page %d", *c.Page)
					}
					fmt.Println(" ---")
					text := c.Text
					if len(text) > 500 {
						text = text[:500] + "..."
					}
					fmt.Println(text)
					fmt.Println()
				}
			}
			return nil
		},
	}

	readCmd.Flags().IntVar(&readChunk, "chunk", 0, "read a specific chunk by position")
	kbCmd.AddCommand(readCmd)

	// kb search
	var searchLimit int

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search KB documents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			results, err := w.KBSearch(args[0], searchLimit)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(results)
			} else {
				for _, r := range results {
					fmt.Printf("%-10s chunk %-3d [%s] %s\n",
						r.DocID, r.ChunkPosition, r.MatchType, r.DocTitle)
					text := r.Text
					if len(text) > 120 {
						text = text[:120] + "..."
					}
					fmt.Printf("  %s\n", text)
				}
				if len(results) == 0 {
					fmt.Println("No results found.")
				}
			}
			return nil
		},
	}

	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "max results")
	kbCmd.AddCommand(searchCmd)

	// kb remove
	removeCmd := &cobra.Command{
		Use:   "remove <kb-id>",
		Short: "Remove a KB document (soft delete to trash)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			if err := w.KBRemove(args[0]); err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]string{"removed": args[0]})
			} else {
				fmt.Printf("Removed %s (moved to trash)\n", args[0])
			}
			return nil
		},
	}

	kbCmd.AddCommand(removeCmd)

	// kb connections
	var (
		connChunk int
		connLimit int
	)

	connectionsCmd := &cobra.Command{
		Use:   "connections <kb-id>",
		Short: "Show chunk-to-chunk connections for a KB document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			var chunkPtr *int
			if cmd.Flags().Changed("chunk") {
				chunkPtr = &connChunk
			}

			results, err := w.KBConnections(args[0], chunkPtr, connLimit)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(results)
			} else {
				for _, r := range results {
					fmt.Printf("  chunk %d → %s chunk %d (%.2f) %s\n",
						r.FromChunkID, r.ToDocID, r.ToPosition, r.Strength, r.ToDocTitle)
					text := r.ToText
					if len(text) > 100 {
						text = text[:100] + "..."
					}
					fmt.Printf("    %s\n", text)
				}
				if len(results) == 0 {
					fmt.Println("No connections found.")
				}
			}
			return nil
		},
	}

	connectionsCmd.Flags().IntVar(&connChunk, "chunk", 0, "filter by chunk position")
	connectionsCmd.Flags().IntVar(&connLimit, "limit", 20, "max results")
	kbCmd.AddCommand(connectionsCmd)

	// kb rebuild-connections
	var (
		rebuildThreshold float64
		rebuildTopK      int
	)

	rebuildConnCmd := &cobra.Command{
		Use:   "rebuild-connections",
		Short: "Recompute all chunk-to-chunk TF-IDF connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			count, err := w.KBRebuildConnections(rebuildThreshold, rebuildTopK)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]any{
					"connections": count,
					"threshold":  rebuildThreshold,
					"top_k":      rebuildTopK,
				})
			} else {
				fmt.Printf("Rebuilt %d connection pairs (threshold=%.2f, top-k=%d)\n",
					count, rebuildThreshold, rebuildTopK)
			}
			return nil
		},
	}

	rebuildConnCmd.Flags().Float64Var(&rebuildThreshold, "threshold", 0.3, "minimum cosine similarity")
	rebuildConnCmd.Flags().IntVar(&rebuildTopK, "top-k", 10, "max connections per chunk")
	kbCmd.AddCommand(rebuildConnCmd)
}

// printKBDoc writes a human-readable KB document summary to stdout.
func printKBDoc(d *workspace.KBDoc) {
	fmt.Printf("%s: %s\n", d.ID, d.Title)
	fmt.Printf("Type: %s\n", d.SourceType)
	fmt.Printf("Path: %s\n", d.Path)
	if d.Note != nil {
		fmt.Printf("Note: %s\n", *d.Note)
	}
	if d.PageCount != nil {
		fmt.Printf("Pages: %d\n", *d.PageCount)
	}
}
