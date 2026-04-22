package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// exemptCommands lists commands that do not require a project to be
// initialized or a username to be set. Everything else is guarded.
var exemptCommands = map[string]bool{
	"init":    true,
	"version": true,
	"skills":  true,
	"help":    true,
	"logs":    true,
}

// rootCmd is the top-level Cobra command that every subcommand hangs off.
var rootCmd = &cobra.Command{
	Use:   "specd",
	Short: "specd - a specification-driven development tool",
	// Guard: runs before every command (including subcommands) to
	// ensure the project is initialized and a username is configured.
	PersistentPreRunE: func(c *cobra.Command, _ []string) error {
		InitLogger()
		return requireProjectInit(c)
	},
	// After every command, check if a newer specd version is available.
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		CheckForUpdate()
	},
}

func init() {
	// Register the version subcommand on the root.
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command and exits on failure.
func Execute() {
	err := rootCmd.Execute()
	CloseLogger()
	if err != nil {
		os.Exit(1)
	}
}

// requireProjectInit blocks non-exempt commands unless both conditions hold:
//  1. A .specd.json marker exists in the current directory (project is initialized).
//  2. A username is set in the global config (~/.specd/config.json).
func requireProjectInit(c *cobra.Command) error {
	// Resolve the top-level command name so subcommands inherit the exemption.
	name := commandName(c)
	if exemptCommands[name] {
		return nil
	}

	// Verify the project marker exists.
	proj, err := LoadProjectConfig(".")
	if err != nil {
		return fmt.Errorf("reading project config: %w", err)
	}
	if proj == nil {
		return fmt.Errorf("specd is not initialized in this directory.\nRun: specd init")
	}

	// Verify a username has been configured.
	cfg, err := LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("reading global config: %w", err)
	}
	if cfg.Username == "" {
		return fmt.Errorf("no username configured.\nRun: specd init --username <name>")
	}

	// Sync the cache database from the spec markdown files on disk.
	slog.Debug("syncing cache", "command", commandName(c))
	if err := SyncCache(); err != nil {
		return fmt.Errorf("syncing cache: %w", err)
	}

	return nil
}

// commandName walks up the command tree to find the name of the
// top-level subcommand (direct child of rootCmd). This lets us
// exempt "skills install" by checking just "skills".
func commandName(c *cobra.Command) string {
	for c.HasParent() && c.Parent().HasParent() {
		c = c.Parent()
	}
	return c.Name()
}
