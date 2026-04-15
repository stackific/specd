// Package cli — watch.go registers the watch command for running the
// file watcher standalone (without the HTTP server).
package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/stackific/specd/internal/watcher"
)

func init() {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch workspace for file changes and sync to SQLite",
		Long:  "Starts a file watcher that monitors specd/ for changes and syncs them to the SQLite cache.",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			logger := log.New(os.Stderr, "", log.LstdFlags)
			wt, err := watcher.New(w, logger)
			if err != nil {
				return fmt.Errorf("create watcher: %w", err)
			}

			if err := wt.Start(); err != nil {
				return fmt.Errorf("start watcher: %w", err)
			}

			fmt.Fprintln(os.Stderr, "Watching workspace for changes... (Ctrl+C to stop)")

			// Wait for interrupt.
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			<-sig

			fmt.Fprintln(os.Stderr, "\nStopping watcher...")
			return wt.Stop()
		},
	}

	rootCmd.AddCommand(cmd)
}
