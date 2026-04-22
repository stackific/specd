package cmd

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// skillsFS holds the embedded skills/ filesystem, set from main.go at startup.
var skillsFS fs.FS

// SetSkillsFS injects the embedded skills filesystem. Called once from main()
// before any commands run.
func SetSkillsFS(fsys fs.FS) {
	skillsFS = fsys
}

// skillsCmd is the parent command for all skills-related subcommands.
var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage specd skills for AI coding tools",
}

// skillsInstallCmd installs skills interactively via `specd skills install`.
var skillsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install specd skills for AI providers",
	RunE:  runSkillsInstall,
}

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
}

// runSkillsInstall is the standalone entry point for `specd skills install`.
// It prompts the user then delegates to installSkills.
func runSkillsInstall(_ *cobra.Command, _ []string) error {
	selectedProviders, level, err := promptSkillsInstall()
	if err != nil {
		return err
	}
	return installSkills(selectedProviders, level)
}

// promptSkillsInstall shows an interactive form where the user selects
// which AI providers to install skills for and whether to install at
// user-level (~/ global) or repo-level (current project only).
func promptSkillsInstall() (providers []string, level string, err error) {
	// Build the option list from the registered providers.
	providerOptions := make([]huh.Option[string], len(Providers))
	for i, p := range Providers {
		providerOptions[i] = huh.NewOption(p.Name, p.Name)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select AI providers").
				Options(providerOptions...).
				Value(&providers),
			huh.NewSelect[string]().
				Title("Install level").
				Options(
					huh.NewOption("User-level (~/.specd shared across projects)", "user"),
					huh.NewOption("Repository-level (current project only)", "repo"),
				).
				Value(&level),
		),
	)

	if err := form.Run(); err != nil {
		return nil, "", err
	}

	return providers, level, nil
}

// installSkills extracts the canonical skills from the embedded filesystem
// to ~/.specd/skills/, then symlinks (or copies on Windows) each skill
// into the selected providers' skills directories.
func installSkills(selectedProviders []string, level string) error {
	if len(selectedProviders) == 0 {
		fmt.Println("No providers selected.")
		return nil
	}

	slog.Info("installing skills", "providers", selectedProviders, "level", level)

	// Write all embedded skills to ~/.specd/skills/ (the canonical location).
	canonicalDir, err := extractCanonicalSkills()
	if err != nil {
		return fmt.Errorf("extracting skills: %w", err)
	}

	// Discover which skills are available (each is a <name>/SKILL.md directory).
	skillNames, err := listSkillNames(canonicalDir)
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}

	// Install every skill for every selected provider.
	var providerNames []string
	for _, provName := range selectedProviders {
		provider, ok := findProvider(provName)
		if !ok {
			continue
		}

		for _, name := range skillNames {
			if err := installSkillForProvider(provider, level, canonicalDir, name); err != nil {
				return fmt.Errorf("installing %s for %s: %w", name, provider.Name, err)
			}
		}
		providerNames = append(providerNames, provider.Name)

		// For user-level installs, show the path so the user knows where files went.
		if level == "user" {
			fmt.Printf("  %s: ~/%s/%s/\n", provider.Name, provider.Dir, provider.CommandsSubdir)
		}
	}

	// Print a single summary line instead of one per skill.
	if level == "user" {
		fmt.Printf("Skills installed for %s.\n", strings.Join(providerNames, ", "))
	} else {
		fmt.Printf("Skills installed for %s (%s).\n", strings.Join(providerNames, ", "), level)
	}
	return nil
}

// extractCanonicalSkills copies all embedded skills to ~/.specd/skills/,
// preserving the Agent Skills Standard directory layout (<name>/SKILL.md).
// This canonical copy is what symlinks point to on macOS/Linux.
func extractCanonicalSkills() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	destDir := filepath.Join(home, InstallDir, SkillsDir)

	// Walk the embedded FS and mirror every file/directory to disk.
	err = fs.WalkDir(skillsFS, EmbedSkillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute the path relative to the embedded skills root.
		rel, err := filepath.Rel(EmbedSkillsDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil // skip the root itself
		}

		dest := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0o750)
		}

		// Read from the embedded FS and write to disk.
		data, err := fs.ReadFile(skillsFS, path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return err
		}

		return os.WriteFile(dest, data, 0o600)
	})
	if err != nil {
		return "", fmt.Errorf("extracting embedded skills: %w", err)
	}

	return destDir, nil
}

// listSkillNames scans the canonical directory and returns the name of
// every subdirectory that contains a SKILL.md file.
func listSkillNames(canonicalDir string) ([]string, error) {
	entries, err := os.ReadDir(canonicalDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // skip loose files
		}
		// A valid skill must have a SKILL.md at its root.
		skillFile := filepath.Join(canonicalDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}

// installSkillForProvider symlinks (or copies on Windows) a single skill
// directory from the canonical location into the provider's skills directory.
func installSkillForProvider(provider Provider, level, canonicalDir, skillName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Determine the target base directory.
	var baseDir string
	switch level {
	case "user":
		baseDir = filepath.Join(home, provider.Dir, provider.CommandsSubdir)
	case "repo":
		baseDir = filepath.Join(provider.Dir, provider.CommandsSubdir)
	default:
		return fmt.Errorf("unknown install level: %s", level)
	}

	src := filepath.Join(canonicalDir, skillName) // e.g. ~/.specd/skills/specd-init
	dest := filepath.Join(baseDir, skillName)     // e.g. ~/.claude/skills/specd-init

	dirPerm, _ := permissionsForLevel(level)
	if err := os.MkdirAll(baseDir, dirPerm); err != nil { //nolint:gosec // perm depends on install level
		return err
	}

	// Remove any previous install (stale symlink or copied directory).
	_ = os.RemoveAll(dest)

	return linkOrCopyDir(src, dest, level)
}

// linkOrCopyDir creates a symlink on macOS/Linux or a recursive copy on Windows.
func linkOrCopyDir(src, dest, level string) error {
	if os.PathSeparator != '\\' {
		return os.Symlink(src, dest)
	}
	return copyDir(src, dest, level)
}

// copyDir recursively copies a directory tree from src to dest.
// Used on Windows where symlinks require elevated privileges.
func copyDir(src, dest, level string) error {
	dirPerm, filePerm := permissionsForLevel(level)
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute the corresponding target path.
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dest, rel)

		if d.IsDir() {
			return os.MkdirAll(target, dirPerm) //nolint:gosec // perm depends on install level
		}

		data, err := os.ReadFile(path) //nolint:gosec // controlled path
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, filePerm) //nolint:gosec // perm depends on install level
	})
}

// findProvider looks up a provider by display name.
func findProvider(name string) (Provider, bool) {
	for _, p := range Providers {
		if p.Name == name {
			return p, true
		}
	}
	return Provider{}, false
}
