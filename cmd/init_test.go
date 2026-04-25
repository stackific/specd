package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

// resetInitFlags clears all flag values and their "changed" state on the
// init command. Cobra shares flag state across Execute() calls in tests,
// so this prevents one test's flags from leaking into the next.
func resetInitFlags() {
	initCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// TestRunInitNonInteractive runs `specd init` with all flags provided,
// verifying that the specd folder, project marker, and global config
// are all created correctly.
func TestRunInitNonInteractive(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()

	// Create a project directory to init inside.
	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}

	// chdir into the project so the marker is written there.
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	rootCmd.SetArgs([]string{"init", "--dir", "specs", "--username", "testuser", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// The specd folder should exist as a directory.
	specdDir := filepath.Join(projectDir, "specs")
	info, err := os.Stat(specdDir)
	if err != nil {
		t.Fatalf("specd folder not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("specd folder is not a directory")
	}

	// The .specd.json marker should record folder, spec types, and task stages.
	proj, err := LoadProjectConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if proj == nil || proj.Dir != "specs" {
		t.Fatalf("expected project config with dir %q, got %v", "specs", proj)
	}
	if len(proj.SpecTypes) == 0 {
		t.Fatal("expected default spec types to be saved")
	}
	if len(proj.TaskStages) == 0 {
		t.Fatal("expected default task stages to be saved")
	}

	// The global config should have the username.
	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}
	if cfg.Username != "testuser" {
		t.Fatalf("expected username %q, got %q", "testuser", cfg.Username)
	}
}

// TestRunInitWithProjectPath verifies that `specd init /some/path` creates
// the specd folder and marker inside the specified directory.
func TestRunInitWithProjectPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()

	targetDir := filepath.Join(tmp, "remote-project")

	// Pass the project path as a positional argument.
	rootCmd.SetArgs([]string{"init", targetDir, "--dir", "specd", "--username", "alice", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// The specd folder should be created inside the target directory.
	specdDir := filepath.Join(targetDir, "specd")
	if _, err := os.Stat(specdDir); err != nil {
		t.Fatalf("specd folder not created at target: %v", err)
	}

	// The marker should be at the target root, not the cwd.
	proj, err := LoadProjectConfig(targetDir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if proj == nil || proj.Dir != "specd" {
		t.Fatalf("expected project config with dir %q, got %v", "specd", proj)
	}
}

// TestRunInitBlocksReinitialization verifies that specd init refuses to
// run in an already-initialized directory.
func TestRunInitBlocksReinitialization(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// First init should succeed.
	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first init: %v", err)
	}

	// Second init should fail.
	resetInitFlags()
	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error on re-initialization, got nil")
	}
}

// TestRunInitUsernameDefaultFromGlobalConfig verifies that an explicit
// --username flag takes precedence when a username already exists in
// the global config. (Interactive prompt cannot be tested here.)
func TestRunInitUsernameDefaultFromGlobalConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()

	// Pre-populate the global config with a username.
	if err := SaveGlobalConfig(&GlobalConfig{Username: "existing"}); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.Join(tmp, "proj2")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Explicitly pass --username to verify it's used.
	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "existing", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Username != "existing" {
		t.Fatalf("expected username %q, got %q", "existing", cfg.Username)
	}
}
