package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// permissionsForLevel returns directory and file permissions.
// Provider skill directories (.claude/, .agents/, .gemini/) are always
// world-readable — repo-level ones are committed to VCS, and user-level
// ones must be readable by the AI tools that consume them.
func permissionsForLevel(_ string) (dirPerm, filePerm fs.FileMode) {
	return 0o755, 0o644
}

// Provider describes an AI coding tool and where its skill files live.
type Provider struct {
	Name           string // display name shown in prompts (e.g. "Claude")
	Dir            string // top-level config directory (e.g. ".claude")
	CommandsSubdir string // subdirectory for skills within Dir (always "skills")
}

// Providers is the list of supported AI coding tool providers.
// All three follow the Agent Skills Standard: <Dir>/skills/<name>/SKILL.md.
var Providers = []Provider{
	{Name: ProviderClaude, Dir: ClaudeDir, CommandsSubdir: ClaudeSkillsSubdir},
	{Name: ProviderCodex, Dir: CodexDir, CommandsSubdir: CodexSkillsSubdir},
	{Name: ProviderGemini, Dir: GeminiDir, CommandsSubdir: GeminiSkillsSubdir},
}

// SkillFile describes a single skill file to be installed for a provider.
type SkillFile struct {
	RelPath string // path relative to the provider's skills directory
	Content []byte // file content (used for Windows copies)
}

// InstallSkill places a skill file into the provider's skills directory.
// On macOS/Linux it creates a symlink pointing back to the canonical copy
// in ~/.specd/skills/. On Windows it writes a copy of the file instead,
// because symlinks require elevated privileges.
func InstallSkill(provider Provider, level, canonicalPath string, sf SkillFile) error {
	var baseDir string

	// Determine the target directory based on install level.
	switch level {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		// e.g. ~/.claude/skills/
		baseDir = filepath.Join(home, provider.Dir, provider.CommandsSubdir)
	case "repo":
		// e.g. .claude/skills/ (relative to cwd)
		baseDir = filepath.Join(provider.Dir, provider.CommandsSubdir)
	default:
		return fmt.Errorf("unknown install level: %s", level)
	}

	dest := filepath.Join(baseDir, sf.RelPath)

	// Repo-level files are committed to VCS and must be world-readable.
	// User-level files are private to the current user.
	dirPerm, filePerm := permissionsForLevel(level)

	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(dest), dirPerm); err != nil { //nolint:gosec // perm depends on install level
		return fmt.Errorf("creating directory %s: %w", filepath.Dir(dest), err)
	}

	// Clean up any previous installation at this path.
	_ = os.Remove(dest)

	if runtime.GOOS == "windows" {
		// Windows: copy the file content directly.
		if err := os.WriteFile(dest, sf.Content, filePerm); err != nil { //nolint:gosec // perm depends on install level
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	} else {
		// macOS/Linux: symlink to the canonical file in ~/.specd/skills/.
		if err := os.Symlink(canonicalPath, dest); err != nil {
			return fmt.Errorf("symlinking %s -> %s: %w", dest, canonicalPath, err)
		}
	}

	return nil
}
