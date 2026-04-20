package cmd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

// resetNewSpecFlags clears flag state to prevent leakage between tests.
func resetNewSpecFlags() {
	newSpecCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// TestNewSpecCreatesFileAndDBRecord verifies that new-spec creates the
// spec.md file, inserts a row into the database, and increments the counter.
func TestNewSpecCreatesFileAndDBRecord(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()

	// Initialize a project.
	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Create a spec.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Auth Flow", "--summary", "OAuth2 login", "--body", "## Details\n\nImplement OAuth2."})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec: %v", err)
	}

	// Verify spec.md file exists.
	specFile := filepath.Join("specd", "specs", "spec-1", "spec.md")
	if _, err := os.Stat(specFile); err != nil {
		t.Fatalf("spec file not created: %v", err)
	}

	// Verify DB record.
	db, err := sql.Open("sqlite", filepath.Join("specd", CacheDBFile))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var id, slug, specType string
	err = db.QueryRow("SELECT id, slug, type FROM specs WHERE id = 'SPEC-1'").Scan(&id, &slug, &specType)
	if err != nil {
		t.Fatalf("reading spec from DB: %v", err)
	}
	if slug != "auth-flow" {
		t.Errorf("expected dash-separated slug %q, got %q", "auth-flow", slug)
	}
	// Default type should be the first spec type.
	if specType != "business" {
		t.Errorf("expected default type %q, got %q", "business", specType)
	}

	// Verify counter incremented.
	var nextID int
	_ = db.QueryRow("SELECT CAST(value AS INTEGER) FROM meta WHERE key = 'next_spec_id'").Scan(&nextID)
	if nextID != 2 {
		t.Errorf("expected next_spec_id=2, got %d", nextID)
	}
}

// TestNewSpecOutputJSON verifies the JSON response structure.
func TestNewSpecOutputJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Capture stdout by redirecting.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Test Spec", "--summary", "A test", "--body", "Body text"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("new-spec: %v", err)
	}

	// Read captured output.
	out := make([]byte, 4096)
	n, _ := r.Read(out)
	output := string(out[:n])

	// Parse JSON response.
	var resp NewSpecResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("parsing JSON response: %v\noutput: %s", err, output)
	}
	if resp.ID != "SPEC-1" {
		t.Errorf("expected ID SPEC-1, got %s", resp.ID)
	}
	if resp.Slug != "test-spec" {
		t.Errorf("expected slug test-spec, got %s", resp.Slug)
	}
	if len(resp.AvailableTypes) == 0 {
		t.Error("expected available_types to be populated")
	}
}
