package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func resetSearchCmdFlags() {
	searchCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// TestSearchCmdFindsSpecs verifies that `specd search --kind spec` returns
// relevant specs ranked by score.
func TestSearchCmdFindsSpecs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetSearchCmdFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
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
	rootCmd.SetArgs([]string{"new-spec", "--title", "User Authentication", "--summary", "OAuth2 login flow", "--body", "Implement OAuth2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "Invoice Generation", "--summary", "PDF billing", "--body", "Generate invoices"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Capture search output.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetSearchCmdFlags()
	rootCmd.SetArgs([]string{"search", "--kind", "spec", "--query", "authentication OAuth2"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("search: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var results SearchResults
	if err := json.Unmarshal(out[:n], &results); err != nil {
		t.Fatalf("parsing results: %v\noutput: %s", err, out[:n])
	}

	found := false
	for _, r := range results.Specs {
		if r.ID == "SPEC-1" {
			found = true
		}
	}
	if !found {
		t.Error("expected SPEC-1 in search results")
	}
	if len(results.Tasks) != 0 {
		t.Errorf("expected no task results with --kind spec, got %d", len(results.Tasks))
	}
}

// TestSearchCmdAllKinds verifies that `specd search --kind all` returns
// results from all kinds.
func TestSearchCmdAllKinds(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetSearchCmdFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
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
	rootCmd.SetArgs([]string{"new-spec", "--title", "Authentication", "--summary", "Login flow", "--body", "OAuth2 auth"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Capture.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetSearchCmdFlags()
	rootCmd.SetArgs([]string{"search", "--query", "authentication"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("search: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var results SearchResults
	if err := json.Unmarshal(out[:n], &results); err != nil {
		t.Fatalf("parsing: %v", err)
	}

	// Should have spec results. Tasks/KB may be empty but should be present as arrays.
	if len(results.Specs) == 0 {
		t.Error("expected spec results with --kind all")
	}
}

// TestSearchCmdRespectsLimit verifies the --limit flag.
func TestSearchCmdRespectsLimit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetSearchCmdFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
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

	for i := 0; i < 5; i++ {
		resetNewSpecFlags()
		rootCmd.SetArgs([]string{
			"new-spec",
			"--title", fmt.Sprintf("System Module %d", i),
			"--summary", "System component",
			"--body", "A system module",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetSearchCmdFlags()
	rootCmd.SetArgs([]string{"search", "--kind", "spec", "--query", "system module", "--limit", "2"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("search: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var results SearchResults
	if err := json.Unmarshal(out[:n], &results); err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if len(results.Specs) > 2 {
		t.Errorf("expected at most 2 results with --limit 2, got %d", len(results.Specs))
	}
}

// TestSearchCmdNegativeCase verifies that unrelated search terms don't
// return false positives.
func TestSearchCmdNegativeCase(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetNewSpecFlags()
	resetSearchCmdFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
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

	// Create a spec about auth only.
	resetNewSpecFlags()
	rootCmd.SetArgs([]string{"new-spec", "--title", "User Authentication", "--summary", "OAuth2 login", "--body", "Implement OAuth2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Search for something completely unrelated.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetSearchCmdFlags()
	rootCmd.SetArgs([]string{"search", "--kind", "spec", "--query", "quantum entanglement photon"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("search: %v", err)
	}

	out := make([]byte, 8192)
	n, _ := r.Read(out)

	var results SearchResults
	if err := json.Unmarshal(out[:n], &results); err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if len(results.Specs) != 0 {
		t.Errorf("expected no results for unrelated query, got %d", len(results.Specs))
	}
}

// TestSearchCmdNoResults verifies empty result with no data.
func TestSearchCmdNoResults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	resetInitFlags()
	resetSearchCmdFlags()

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil { //nolint:gosec // test
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

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetSearchCmdFlags()
	rootCmd.SetArgs([]string{"search", "--query", "anything at all"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("search: %v", err)
	}

	out := make([]byte, 4096)
	n, _ := r.Read(out)

	var results SearchResults
	if err := json.Unmarshal(out[:n], &results); err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if len(results.Specs) != 0 || len(results.Tasks) != 0 || len(results.KB) != 0 {
		t.Error("expected all empty results on empty database")
	}
}
