// logger.go initializes structured logging to ~/.specd/specd.log using
// the standard library log/slog package. The log file is JSON lines format
// for easy parsing and tailing.
//
// Log level defaults to Info. Set SPECD_DEBUG=1 to enable Debug output.
// The log file is truncated when it exceeds MaxLogSize to avoid unbounded growth.
package cmd

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// logFile holds the open log file handle so it can be closed at shutdown.
var logFile *os.File

// InitLogger sets up slog to write JSON lines to ~/.specd/specd.log.
// Must be called early in the command lifecycle (PersistentPreRunE).
// Safe to call multiple times — subsequent calls are no-ops.
func InitLogger() {
	if logFile != nil {
		return // already initialized
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return // can't log if we can't find home — fail silently
	}

	logDir := filepath.Join(home, InstallDir)
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return
	}

	logPath := filepath.Join(logDir, LogFile)

	// Truncate if the log file exceeds MaxLogSize.
	truncateIfNeeded(logPath)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // path built from UserHomeDir + hardcoded suffix
	if err != nil {
		return
	}
	logFile = f

	level := slog.LevelInfo
	if os.Getenv("SPECD_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// CloseLogger flushes and closes the log file.
func CloseLogger() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}

// truncateIfNeeded resets the log file if it exceeds MaxLogSize.
// This is a simple size-based rotation — no archive, just reset.
func truncateIfNeeded(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // file doesn't exist yet — nothing to truncate
	}
	if info.Size() > MaxLogSize {
		_ = os.Truncate(path, 0)
	}
}

// LogWriter returns a writer that appends to the log file, or io.Discard
// if the logger is not initialized. Used by the logs command to locate
// the log file path.
func LogWriter() io.Writer {
	if logFile != nil {
		return logFile
	}
	return io.Discard
}

// LogFilePath returns the absolute path to the log file.
func LogFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, InstallDir, LogFile)
}
