package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// TestExtractCanonicalSkills verifies that the embedded skills filesystem
// is correctly extracted to ~/.specd/skills/ with the right directory structure.
func TestExtractCanonicalSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Replace the real embedded FS with a test double.
	skillsFS = testSkillsFS()

	dir, err := extractCanonicalSkills()
	if err != nil {
		t.Fatalf("extractCanonicalSkills: %v", err)
	}

	// The skill should be at <dir>/test-skill/SKILL.md.
	skillMD := filepath.Join(dir, "test-skill", "SKILL.md")
	data, err := os.ReadFile(skillMD) //nolint:gosec // test helper with controlled path
	if err != nil {
		t.Fatalf("reading extracted skill: %v", err)
	}
	if !strings.Contains(string(data), "test-skill") {
		t.Fatalf("expected skill content to contain 'test-skill', got %q", string(data))
	}
}

// TestListSkillNames verifies that only directories containing a SKILL.md
// are recognized as skills; directories without it and loose files are ignored.
func TestListSkillNames(t *testing.T) {
	tmp := t.TempDir()

	// Create a valid skill directory with a SKILL.md.
	skillDir := filepath.Join(tmp, "my-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create a directory without SKILL.md — should be ignored.
	if err := os.MkdirAll(filepath.Join(tmp, "not-a-skill"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Create a regular file — should also be ignored.
	if err := os.WriteFile(filepath.Join(tmp, "readme.md"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	names, err := listSkillNames(tmp)
	if err != nil {
		t.Fatalf("listSkillNames: %v", err)
	}
	if len(names) != 1 || names[0] != "my-skill" {
		t.Fatalf("expected [my-skill], got %v", names)
	}
}

// TestInstallSkillForProviderClaude verifies user-level skill installation
// for the Claude provider (symlink on macOS/Linux, copy on Windows).
func TestInstallSkillForProviderClaude(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	canonDir := setupTestSkill(t, tmp)

	claude := Providers[0] // Claude
	if err := installSkillForProvider(claude, "user", canonDir, "test-skill"); err != nil {
		t.Fatalf("installSkillForProvider: %v", err)
	}

	// Verify the SKILL.md is reachable at the expected path.
	installed := filepath.Join(tmp, ClaudeDir, ClaudeSkillsSubdir, "test-skill", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("installed skill not found: %v", err)
	}
}

// TestInstallSkillForProviderCodex verifies user-level skill installation
// for the OpenAI Codex provider.
func TestInstallSkillForProviderCodex(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	canonDir := setupTestSkill(t, tmp)

	codex := Providers[1] // Codex
	if err := installSkillForProvider(codex, "user", canonDir, "test-skill"); err != nil {
		t.Fatalf("installSkillForProvider: %v", err)
	}

	installed := filepath.Join(tmp, CodexDir, CodexSkillsSubdir, "test-skill", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("installed skill not found: %v", err)
	}
}

// TestInstallSkillForProviderGemini verifies user-level skill installation
// for the Gemini provider.
func TestInstallSkillForProviderGemini(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	canonDir := setupTestSkill(t, tmp)

	gemini := Providers[2] // Gemini
	if err := installSkillForProvider(gemini, "user", canonDir, "test-skill"); err != nil {
		t.Fatalf("installSkillForProvider: %v", err)
	}

	installed := filepath.Join(tmp, GeminiDir, GeminiSkillsSubdir, "test-skill", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("installed skill not found: %v", err)
	}
}

// TestInstallSkillRepoLevel verifies that repo-level installation places
// the skill relative to the current working directory, not the home directory.
func TestInstallSkillRepoLevel(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	canonDir := setupTestSkill(t, tmp)

	// Simulate being inside a project repository.
	origDir, _ := os.Getwd()
	repoDir := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil { //nolint:gosec // test repo directory
		t.Fatal(err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	claude := Providers[0]
	if err := installSkillForProvider(claude, "repo", canonDir, "test-skill"); err != nil {
		t.Fatalf("installSkillForProvider repo: %v", err)
	}

	// The skill should be under the repo dir, not under HOME.
	installed := filepath.Join(repoDir, ClaudeDir, ClaudeSkillsSubdir, "test-skill", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("repo-level skill not found: %v", err)
	}
}

// setupTestSkill creates a minimal canonical skill directory under the
// given home directory's ~/.specd/skills/ for use in installation tests.
func setupTestSkill(t *testing.T, home string) string {
	t.Helper()
	canonDir := filepath.Join(home, InstallDir, SkillsDir)
	skillDir := filepath.Join(canonDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	content := []byte("---\nname: test-skill\ndescription: A test skill.\n---\n\nTest instructions.\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	return canonDir
}

// testSkillsFS returns a fake embedded filesystem containing a single
// test skill. Used to replace the real skillsFS in unit tests.
func testSkillsFS() fstest.MapFS {
	return fstest.MapFS{
		"skills/test-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: test-skill\ndescription: A test skill.\n---\n\nTest instructions.\n"),
		},
	}
}
