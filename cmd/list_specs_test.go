package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func resetListSpecsFlags() {
	listSpecsCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// setupProjectWithSpecs creates a project with N specs.
func setupProjectWithSpecs(t *testing.T, count int) {
	t.Helper()
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
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"init", "--folder", "specd", "--username", "tester", "--skip-skills"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < count; i++ {
		resetNewSpecFlags()
		title := "Spec " + string(rune('A'+i))
		rootCmd.SetArgs([]string{"new-spec", "--title", title, "--summary", "Summary " + title, "--body", "Body"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("new-spec %d: %v", i+1, err)
		}
	}
}

// captureListSpecs runs list-specs and returns the parsed response.
func captureListSpecs(t *testing.T, args ...string) ListSpecsResponse {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	resetListSpecsFlags()
	fullArgs := append([]string{"list-specs"}, args...)
	rootCmd.SetArgs(fullArgs)
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("list-specs: %v", err)
	}

	out := make([]byte, 16384)
	n, _ := r.Read(out)

	var resp ListSpecsResponse
	if err := json.Unmarshal(out[:n], &resp); err != nil {
		t.Fatalf("parsing response: %v\noutput: %s", err, out[:n])
	}
	return resp
}

// TestListSpecsReturnsAll verifies listing all specs.
func TestListSpecsReturnsAll(t *testing.T) {
	setupProjectWithSpecs(t, 3)
	resp := captureListSpecs(t)

	if resp.TotalCount != 3 {
		t.Errorf("expected 3 total, got %d", resp.TotalCount)
	}
	if len(resp.Specs) != 3 {
		t.Errorf("expected 3 specs, got %d", len(resp.Specs))
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
}

// TestListSpecsPagination verifies page size and page number work.
func TestListSpecsPagination(t *testing.T) {
	setupProjectWithSpecs(t, 5)

	// Page 1 with page-size 2.
	resp := captureListSpecs(t, "--page", "1", "--page-size", "2")
	if len(resp.Specs) != 2 {
		t.Errorf("page 1: expected 2 specs, got %d", len(resp.Specs))
	}
	if resp.TotalCount != 5 {
		t.Errorf("expected total 5, got %d", resp.TotalCount)
	}
	if resp.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", resp.TotalPages)
	}

	// Page 3 with page-size 2 should return 1 spec.
	resp = captureListSpecs(t, "--page", "3", "--page-size", "2")
	if len(resp.Specs) != 1 {
		t.Errorf("page 3: expected 1 spec, got %d", len(resp.Specs))
	}
}

// TestListSpecsEmpty verifies empty project returns empty array.
func TestListSpecsEmpty(t *testing.T) {
	setupProjectWithSpecs(t, 0)
	resp := captureListSpecs(t)

	if resp.TotalCount != 0 {
		t.Errorf("expected 0 total, got %d", resp.TotalCount)
	}
	if resp.Specs == nil {
		t.Error("specs should be empty array, not null")
	}
}

// TestListSpecsBeyondLastPage verifies requesting a page past the end
// returns an empty result set.
func TestListSpecsBeyondLastPage(t *testing.T) {
	setupProjectWithSpecs(t, 3)
	resp := captureListSpecs(t, "--page", "99", "--page-size", "10")

	if len(resp.Specs) != 0 {
		t.Errorf("expected 0 specs on page 99, got %d", len(resp.Specs))
	}
	if resp.TotalCount != 3 {
		t.Errorf("total should still be 3, got %d", resp.TotalCount)
	}
}
