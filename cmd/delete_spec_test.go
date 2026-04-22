package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

// resetDeleteSpecFlags clears flag state to prevent leakage between tests.
func resetDeleteSpecFlags() {
	deleteSpecCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupProjectWithSpec initializes a project and creates a spec, returning
// the project dir. The caller must chdir into the project dir first.
func setupProjectWithSpec(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetDeleteSpecFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test project directory
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Test Spec", "--summary", "A test", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec: %v", err)
	}

	return projectDir
}

// TestDeleteSpecRemovesFromDB verifies that delete-spec removes the spec
// row and its cascaded dependents from the database.
func TestDeleteSpecRemovesFromDB(t *testing.T) {
	_ = setupProjectWithSpec(t)

	resetDeleteSpecFlags()
	rootCmd.SetArgs([]string{"delete-spec", "--id", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-spec: %v", err)
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-1'").Scan(&count)
	if count != 0 {
		t.Error("spec should be deleted from DB")
	}
}

// TestDeleteSpecRemovesDirectory verifies that delete-spec removes the
// spec directory from disk.
func TestDeleteSpecRemovesDirectory(t *testing.T) {
	_ = setupProjectWithSpec(t)

	specDir := filepath.Join("specd", "specs", "spec-1")
	if _, err := os.Stat(specDir); err != nil {
		t.Fatalf("spec directory should exist before delete: %v", err)
	}

	resetDeleteSpecFlags()
	rootCmd.SetArgs([]string{"delete-spec", "--id", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-spec: %v", err)
	}

	if _, err := os.Stat(specDir); !os.IsNotExist(err) {
		t.Error("spec directory should be removed from disk after delete")
	}
}

// TestDeleteSpecCascadesToLinks verifies that deleting a spec also removes
// its spec_links entries via ON DELETE CASCADE.
func TestDeleteSpecCascadesToLinks(t *testing.T) {
	_ = setupProjectWithSpec(t)

	// Create a second spec and link them.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec B", "--summary", "Second", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	resetUpdateSpecFlags()
	rootCmd.SetArgs([]string{"update-spec", "--id", "SPEC-2", "--link-specs", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Verify links exist before delete.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	var linkCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links").Scan(&linkCount)
	_ = db.Close()
	if linkCount == 0 {
		t.Fatal("expected spec_links to exist before delete")
	}

	// Delete SPEC-1 — should cascade to its links.
	resetDeleteSpecFlags()
	rootCmd.SetArgs([]string{"delete-spec", "--id", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-spec: %v", err)
	}

	db, err = sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// All links involving SPEC-1 should be gone.
	var remaining int
	_ = db.QueryRow("SELECT COUNT(*) FROM spec_links WHERE from_spec = 'SPEC-1' OR to_spec = 'SPEC-1'").Scan(&remaining)
	if remaining != 0 {
		t.Errorf("expected 0 links involving SPEC-1, got %d", remaining)
	}
}

// TestDeleteSpecOtherSpecSurvives verifies that deleting one spec does not
// affect unrelated specs — the other spec should remain in DB and on disk.
func TestDeleteSpecOtherSpecSurvives(t *testing.T) {
	_ = setupProjectWithSpec(t)

	// Create a second spec.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Spec B", "--summary", "Keep this", "--body", "B"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Delete SPEC-1.
	resetDeleteSpecFlags()
	rootCmd.SetArgs([]string{"delete-spec", "--id", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-spec: %v", err)
	}

	// SPEC-2 should still exist in DB.
	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-2'").Scan(&count)
	if count != 1 {
		t.Error("SPEC-2 should survive deletion of SPEC-1")
	}

	// SPEC-2 directory should still exist on disk.
	if _, err := os.Stat(filepath.Join("specd", "specs", "spec-2")); err != nil {
		t.Error("SPEC-2 directory should still exist on disk")
	}
}

// TestDeleteSpecIDsNotReused verifies that after deleting a spec,
// creating a new spec gets the next ID, not the deleted one.
func TestDeleteSpecIDsNotReused(t *testing.T) {
	_ = setupProjectWithSpec(t)

	// Delete SPEC-1.
	resetDeleteSpecFlags()
	rootCmd.SetArgs([]string{"delete-spec", "--id", "SPEC-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("delete-spec: %v", err)
	}

	// Create a new spec — should get SPEC-2, not SPEC-1.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "New After Delete", "--summary", "Should be SPEC-2", "--body", "Body"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("new-spec after delete: %v", err)
	}

	db, err := sql.Open("sqlite", CacheDBFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-1'").Scan(&count)
	if count != 0 {
		t.Error("SPEC-1 should not be reused")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM specs WHERE id = 'SPEC-2'").Scan(&count)
	if count != 1 {
		t.Error("new spec should be SPEC-2")
	}
}

// TestDeleteSpecNotFound verifies that deleting a nonexistent spec returns
// an error.
func TestDeleteSpecNotFound(t *testing.T) {
	_ = setupProjectWithSpec(t)

	resetDeleteSpecFlags()
	rootCmd.SetArgs([]string{"delete-spec", "--id", "SPEC-999"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when deleting nonexistent spec")
	}
}
