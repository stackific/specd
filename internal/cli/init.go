// Package cli — init.go registers the init command for creating new
// specd workspaces with directory structure, database, and gitignore.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

func init() {
	var force bool

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new specd workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			w, err := workspace.Init(path, workspace.InitOptions{Force: force})
			if err != nil {
				return err
			}
			defer w.Close()

			if jsonOutput {
				printJSON(map[string]string{
					"root":    w.Root,
					"message": "workspace initialized",
				})
			} else {
				fmt.Printf("Initialized specd workspace at %s\n", w.Root)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "reinitialize existing workspace")
	rootCmd.AddCommand(cmd)
}
