package cmd

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
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
	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
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
	db, err := sql.Open("sqlite", CacheDBFile)
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
	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
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
	db, err := sql.Open("sqlite", CacheDBFile)
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

// TestUpdateSpecRewritesLinkedSpecsInFile verifies that update-spec with
// --link-specs writes linked_specs into the spec.md frontmatter.
func TestUpdateSpecRewritesLinkedSpecsInFile(t *testing.T) {
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

	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Create two specs.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec A", "--summary", "First", "--body", "A"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec B", "--summary", "Second", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Link them.
	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-2", "--link-specs", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-spec: %v", err)
	}

	// Read the rewritten spec.md and verify linked_specs is in frontmatter.
	specFile := filepath.Join("specd", "specs", "spec-2", "spec.md")
	data, err := os.ReadFile(specFile) //nolint:gosec // test reads from controlled temp path
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "linked_specs:") {
		t.Error("expected linked_specs in rewritten spec.md frontmatter")
	}
	if !strings.Contains(content, "  - SPEC-1") {
		t.Error("expected SPEC-1 in linked_specs list")
	}

	// Verify content_hash was updated in DB to match the rewritten file.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var dbHash string
	_ = db.QueryRow("SELECT content_hash FROM specs WHERE id = 'SPEC-2'").Scan(&dbHash)

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(data))
	if dbHash != expectedHash {
		t.Errorf("content_hash mismatch after rewrite: DB=%s, file=%s", dbHash, expectedHash)
	}
}

// TestUpdateSpecUnlinkSpecs verifies that --unlink-specs removes bidirectional
// spec_links and updates the spec.md file.
func TestUpdateSpecUnlinkSpecs(t *testing.T) {
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

	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Create two specs and link them.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec A", "--summary", "First", "--body", "A"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec B", "--summary", "Second", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-1", "--link-specs", "SPEC-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Verify link exists.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	var linkCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links").Scan(&linkCount)
	_ = db.Close()
	if linkCount == 0 {
		t.Fatal("expected links to exist before unlink")
	}

	// Unlink them.
	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-1", "--unlink-specs", "SPEC-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-spec unlink: %v", err)
	}

	// Verify links are gone.
	db, err = sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links").Scan(&linkCount)
	if linkCount != 0 {
		t.Errorf("expected 0 links after unlink, got %d", linkCount)
	}

	// Verify the spec.md no longer contains linked_specs.
	data, err := os.ReadFile(filepath.Join("specd", "specs", "spec-1", "spec.md")) //nolint:gosec // test
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "linked_specs:") {
		t.Error("spec.md should not contain linked_specs after unlinking all")
	}
}

// TestUpdateSpecResponseIncludesSummaries verifies that the update-spec
// response includes title and summary for linked specs, not just IDs.
func TestUpdateSpecResponseIncludesSummaries(t *testing.T) {
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

	rootCmd.SetArgs([]string{"init", "--dir", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Auth", "--summary", "OAuth2 flow", "--body", "A"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Sessions", "--summary", "Token mgmt", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Capture output to parse JSON response.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-1", "--link-specs", "SPEC-2"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("update-spec: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var resp UpdateSpecResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v\noutput: %s", err, out[:n])
	}

	if len(resp.LinkedSpecs) != 1 {
		t.Fatalf("expected 1 linked spec, got %d", len(resp.LinkedSpecs))
	}
	if resp.LinkedSpecs[0].ID != "SPEC-2" {
		t.Errorf("expected linked spec ID SPEC-2, got %s", resp.LinkedSpecs[0].ID)
	}
	if resp.LinkedSpecs[0].Title != "Sessions" {
		t.Errorf("expected linked spec title 'Sessions', got %q", resp.LinkedSpecs[0].Title)
	}
	if resp.LinkedSpecs[0].Summary != "Token mgmt" {
		t.Errorf("expected linked spec summary 'Token mgmt', got %q", resp.LinkedSpecs[0].Summary)
	}
}
