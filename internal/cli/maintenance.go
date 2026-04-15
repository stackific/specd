// Package cli — maintenance.go registers the maintenance commands: lint, tidy,
// rebuild, status, merge-fixup, and trash (list, restore, purge, purge-all).
package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	// lint
	lintCmd := &cobra.Command{
		Use:   "lint",
		Short: "Run read-only consistency checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.Lint()
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				if len(result.Issues) == 0 {
					fmt.Println("No issues found.")
				} else {
					for _, issue := range result.Issues {
						icon := "⚠"
						if issue.Severity == "error" {
							icon = "✗"
						}
						id := issue.ID
						if id != "" {
							id = " " + id
						}
						fmt.Printf("[%s]%s %s: %s\n", icon, id, issue.Category, issue.Message)
					}
					fmt.Printf("\n%d errors, %d warnings\n", result.Counts.Errors, result.Counts.Warnings)
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(lintCmd)

	// tidy
	tidyCmd := &cobra.Command{
		Use:   "tidy",
		Short: "Run lint and update last tidy timestamp",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.Tidy()
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				if len(result.Issues) == 0 {
					fmt.Println("No issues found. Tidy timestamp updated.")
				} else {
					for _, issue := range result.Issues {
						icon := "⚠"
						if issue.Severity == "error" {
							icon = "✗"
						}
						id := issue.ID
						if id != "" {
							id = " " + id
						}
						fmt.Printf("[%s]%s %s: %s\n", icon, id, issue.Category, issue.Message)
					}
					fmt.Printf("\n%d errors, %d warnings. Tidy timestamp updated.\n",
						result.Counts.Errors, result.Counts.Warnings)
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(tidyCmd)

	// rebuild
	var rebuildForce bool
	rebuildCmd := &cobra.Command{
		Use:   "rebuild",
		Short: "Wipe cache and re-parse workspace from markdown files",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.Rebuild(rebuildForce)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				fmt.Printf("Rebuilt: %d specs, %d tasks, %d KB docs (%d chunks)\n",
					result.Specs, result.Tasks, result.KBDocs, result.KBChunks)
				if len(result.RejectedFiles) > 0 {
					fmt.Printf("Rejected files: %d\n", len(result.RejectedFiles))
					for _, f := range result.RejectedFiles {
						fmt.Printf("  %s\n", f)
					}
				}
			}
			return nil
		},
	}
	rebuildCmd.Flags().BoolVar(&rebuildForce, "force", false, "force rebuild even if cache exists")
	rootCmd.AddCommand(rebuildCmd)

	// status
	var statusDetailed bool
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show project summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.Status(statusDetailed)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				fmt.Print(workspace.FormatStatus(result))
			}
			return nil
		},
	}
	statusCmd.Flags().BoolVar(&statusDetailed, "detailed", false, "include lint summary")
	rootCmd.AddCommand(statusCmd)

	// merge-fixup
	mergeFixupCmd := &cobra.Command{
		Use:   "merge-fixup",
		Short: "Detect and repair ID collisions after git merge",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.MergeFixup()
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				total := len(result.DuplicateSpecs) + len(result.DuplicateTasks) + len(result.DuplicateKB)
				if total == 0 {
					fmt.Println("No duplicate IDs found.")
				} else {
					for _, dup := range result.DuplicateSpecs {
						fmt.Printf("Duplicate spec %s at: %v\n", dup.ID, dup.Paths)
					}
					for _, dup := range result.DuplicateTasks {
						fmt.Printf("Duplicate task %s at: %v\n", dup.ID, dup.Paths)
					}
					for _, dup := range result.DuplicateKB {
						fmt.Printf("Duplicate KB %s at: %v\n", dup.ID, dup.Paths)
					}
				}
				for _, r := range result.Renumbered {
					fmt.Printf("Renumbered %s -> %s (%s -> %s)\n", r.OldID, r.NewID, r.OldPath, r.NewPath)
				}
				if len(result.Renumbered) > 0 {
					fmt.Println("Database rebuilt after renumbering.")
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(mergeFixupCmd)

	// trash (parent command with subcommands)
	trashCmd := &cobra.Command{
		Use:   "trash",
		Short: "Manage soft-deleted items",
	}

	// trash list
	var trashListKind, trashListOlderThan string
	trashListCmd := &cobra.Command{
		Use:   "list",
		Short: "List soft-deleted items",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			items, err := w.ListTrash(workspace.TrashListFilter{
				Kind:      trashListKind,
				OlderThan: trashListOlderThan,
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(items)
			} else {
				if len(items) == 0 {
					fmt.Println("Trash is empty.")
				} else {
					for _, item := range items {
						fmt.Printf("#%d  [%s]  %s  %s  (deleted %s by %s)\n",
							item.ID, item.Kind, item.OriginalID, item.OriginalPath,
							item.DeletedAt, item.DeletedBy)
					}
				}
			}
			return nil
		},
	}
	trashListCmd.Flags().StringVar(&trashListKind, "kind", "", "filter by kind (spec|task|kb)")
	trashListCmd.Flags().StringVar(&trashListOlderThan, "older-than", "", "filter by age (e.g. 30d)")

	// trash restore
	trashRestoreCmd := &cobra.Command{
		Use:   "restore <trash-id>",
		Short: "Restore a soft-deleted item from trash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			trashID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid trash ID: %s", args[0])
			}

			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			result, err := w.RestoreTrash(trashID)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(result)
			} else {
				fmt.Printf("Restored %s %s to %s\n", result.Kind, result.RestoredID, result.Path)
				if result.Warning != "" {
					fmt.Printf("Warning: %s\n", result.Warning)
				}
			}
			return nil
		},
	}

	// trash purge
	var trashPurgeOlderThan string
	trashPurgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Permanently remove old trash entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			if trashPurgeOlderThan == "" {
				trashPurgeOlderThan = "30d"
			}

			count, err := w.PurgeTrash(trashPurgeOlderThan)
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]int{"purged": count})
			} else {
				fmt.Printf("Purged %d trash entries older than %s.\n", count, trashPurgeOlderThan)
			}
			return nil
		},
	}
	trashPurgeCmd.Flags().StringVar(&trashPurgeOlderThan, "older-than", "30d", "purge entries older than (e.g. 30d)")

	// trash purge-all
	trashPurgeAllCmd := &cobra.Command{
		Use:   "purge-all",
		Short: "Permanently remove all trash entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			count, err := w.PurgeAllTrash()
			if err != nil {
				return err
			}

			if jsonOutput {
				printJSON(map[string]int{"purged": count})
			} else {
				fmt.Printf("Purged all %d trash entries.\n", count)
			}
			return nil
		},
	}

	trashCmd.AddCommand(trashListCmd, trashRestoreCmd, trashPurgeCmd, trashPurgeAllCmd)
	rootCmd.AddCommand(trashCmd)
}
