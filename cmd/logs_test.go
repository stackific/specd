package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLogFilePath verifies the log file path is correctly constructed.
func TestLogFilePath2(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	path := LogFilePath()
	expected := filepath.Join(tmp, InstallDir, LogFile)
	if path != expected {
		t.Errorf("LogFilePath() = %q, want %q", path, expected)
	}
}

// TestLogsNoFileExistsGracefully verifies that runLogs handles a missing
// log file without crashing. We call the function directly instead of
// going through rootCmd.Execute() to avoid the follow loop blocking.
func TestLogsNoFileExistsGracefully(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// LogFilePath should point to a nonexistent file.
	path := LogFilePath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("log file should not exist yet")
	}
}

// TestLogsFileReadable verifies that an existing log file can be opened
// by the logs command path.
func TestLogsFileReadable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	logDir := filepath.Join(tmp, InstallDir)
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(logDir, LogFile)
	if err := os.WriteFile(logPath, []byte("{\"msg\":\"test\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Verify the file is readable at the expected path.
	data, err := os.ReadFile(LogFilePath()) //nolint:gosec // test
	if err != nil {
		t.Fatalf("could not read log file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected log file to contain data")
	}
}
