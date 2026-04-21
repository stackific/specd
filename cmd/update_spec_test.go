package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

// resetUpdateSpecFlags clears flag state to prevent leakage between tests.
func resetUpdateSpecFlags() {
	updateSpecCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// TestUpdateSpecChangesType verifies that update-spec updates the type
// in both the database and the spec.md file.
func TestUpdateSpecChangesType(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetUpdateSpecFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init + create spec.
	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Test", "--summary", "Test spec", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec: %v", err)
	}

	// Update the type.
	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-1", "--type", "functional"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-spec: %v", err)
	}

	// Verify DB was updated.
	db, err := sql.Open("sqlite", filepath.Join("specd", CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var specType string
	_ = db.QueryRow("SELECT type FROM specs WHERE id = 'SPEC-1'").Scan(&specType)
	if specType != "functional" {
		t.Errorf("expected type %q in DB, got %q", "functional", specType)
	}

	// Verify spec.md was updated.
	specFile := filepath.Join("specd", "specs", "spec-1", "spec.md")
	data, err := os.ReadFile(specFile) //nolint:gosec // test reads from controlled temp path
	if err != nil {
		t.Fatalf("reading spec file: %v", err)
	}
	if !strings.Contains(string(data), "type: functional") {
		t.Error("expected spec.md to contain 'type: functional'")
	}
}

// TestUpdateSpecLinksSpecs verifies that update-spec creates bidirectional
// spec_links entries.
func TestUpdateSpecLinksSpecs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetUpdateSpecFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Init + create two specs.
	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec A", "--summary", "First", "--body", "A"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec 1: %v", err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec B", "--summary", "Second", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec 2: %v", err)
	}

	// Link them.
	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-2", "--type", "functional", "--link-specs", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-spec: %v", err)
	}

	// Verify bidirectional links in DB.
	db, err := sql.Open("sqlite", filepath.Join("specd", CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 bidirectional links, got %d", count)
	}
}
