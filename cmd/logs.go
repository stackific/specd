// logs.go implements `specd logs` which streams the log file to stdout.
// Uses tail -f semantics — prints existing content then follows new lines.
package cmd

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
)

// logsCmd implements `specd logs`.
// Streams ~/.specd/specd.log to stdout, following new output like tail -f.
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream specd log output",
	Long:  "Streams the specd log file (~/.specd/specd.log) to stdout. Follows new output like tail -f. Press Ctrl+C to stop.",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().Int("lines", 0, "number of recent lines to show (0 = all)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(_ *cobra.Command, _ []string) error {
	path := LogFilePath()
	if path == "" {
		return fmt.Errorf("cannot determine log file path")
	}

	f, err := os.Open(path) //nolint:gosec // path built from UserHomeDir + hardcoded suffix
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No log file yet. Logs will appear at %s after running specd commands.\n", path)
			return nil
		}
		return fmt.Errorf("opening log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Print existing content.
	if _, err := io.Copy(os.Stdout, f); err != nil {
		return fmt.Errorf("reading log file: %w", err)
	}

	// Follow new output until interrupted.
	fmt.Fprintf(os.Stderr, "\n--- following %s (Ctrl+C to stop) ---\n", path)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	buf := make([]byte, 4096)
	for {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr)
			return nil
		default:
			n, err := f.Read(buf)
			if n > 0 {
				_, _ = os.Stdout.Write(buf[:n])
			}
			if err == io.EOF {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if err != nil {
				return fmt.Errorf("reading log: %w", err)
			}
		}
	}
}
