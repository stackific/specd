// Package cli — config.go registers the config command for reading and
// setting workspace configuration values like user.name.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "config <key> [value]",
		Short: "Get or set configuration values",
		Long:  "Get or set configuration values. Currently supports: user.name",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			key := args[0]

			// Map dotted key to meta key.
			metaKey, ok := configKeyMap[key]
			if !ok {
				return fmt.Errorf("unknown config key: %s", key)
			}

			if len(args) == 1 {
				// Read.
				val, err := w.DB.GetMeta(metaKey)
				if err != nil {
					return err
				}
				if jsonOutput {
					printJSON(map[string]string{key: val})
				} else {
					fmt.Println(val)
				}
				return nil
			}

			// Write.
			if err := w.DB.SetMeta(metaKey, args[1]); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(map[string]string{key: args[1], "message": "updated"})
			} else {
				fmt.Printf("%s = %s\n", key, args[1])
			}
			return nil
		},
	}

	rootCmd.AddCommand(cmd)
}

// configKeyMap translates user-facing dotted keys (e.g. "user.name")
// to the corresponding meta table keys in SQLite.
var configKeyMap = map[string]string{
	"user.name": "user_name",
}
