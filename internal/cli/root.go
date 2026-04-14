// Package cli defines the cobra command tree for the specd CLI.
// Each file registers its commands in an init() function.
// All commands support --json for machine-readable output.
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/workspace"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "specd",
	Short: "Specification, task, and knowledge base management",
	Long:  "A local spec, task, and knowledge base management tool for any project or workspace.",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output JSON")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// openWorkspace finds and opens the workspace from the current directory.
func openWorkspace() (*workspace.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := workspace.FindRoot(cwd)
	if err != nil {
		return nil, err
	}
	return workspace.Open(root)
}

// printJSON marshals v to JSON and prints it.
func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json encode: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
