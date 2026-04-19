package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "specd",
	Short: "specd - a specification-driven development tool",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("Hello from specd!")
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		CheckForUpdate()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
