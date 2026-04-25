package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGlobalConfigRoundTrip verifies that saving and loading global config
// produces consistent results, and that a missing file returns an empty config.
func TestGlobalConfigRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", orig) }()

	// Before any config file exists, Load should return an empty config.
	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}
	if cfg.Username != "" {
		t.Fatalf("expected empty username, got %q", cfg.Username)
	}

	// Write a username and persist it.
	cfg.Username = "testuser"
	if err := SaveGlobalConfig(cfg); err != nil {
		t.Fatalf("SaveGlobalConfig: %v", err)
	}

	// Confirm the file was actually created on disk.
	p := filepath.Join(tmp, InstallDir, ConfigFile)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Re-load and verify the value survived the round-trip.
	cfg2, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig after save: %v", err)
	}
	if cfg2.Username != "testuser" {
		t.Fatalf("expected username %q, got %q", "testuser", cfg2.Username)
	}
}

// TestProjectConfigRoundTrip verifies that the .specd.json marker can be
// written and read back, and that a missing marker returns nil (not an error).
func TestProjectConfigRoundTrip(t *testing.T) {
	tmp := t.TempDir()

	// No marker yet — should return nil without error.
	proj, err := LoadProjectConfig(tmp)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if proj != nil {
		t.Fatal("expected nil project config before init")
	}

	// Write a project marker.
	if err := SaveProjectConfig(tmp, &ProjectConfig{Dir: "myspecd"}); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	// Confirm the marker file exists.
	p := filepath.Join(tmp, ProjectMarker)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("project marker not created: %v", err)
	}

	// Re-load and verify.
	proj, err = LoadProjectConfig(tmp)
	if err != nil {
		t.Fatalf("LoadProjectConfig after save: %v", err)
	}
	if proj == nil {
		t.Fatal("expected non-nil project config after save")
	}
	if proj.Dir != "myspecd" {
		t.Fatalf("expected dir %q, got %q", "myspecd", proj.Dir)
	}
}
